# Transire

**Cloud-agnostic Go framework for building production-ready APIs**

Write standard Go applications with Chi routing that run seamlessly across local development and cloud platforms (AWS Lambda). Zero boilerplate, no framework lock-in, just Go.

```go
func main() {
    app := transire.New()
    r := app.Router()  // Standard Chi router

    r.Get("/api/users", getUsersHandler)
    r.Post("/api/users", createUserHandler)

    // Add background handlers
    app.RegisterQueueHandler(&EmailHandler{})
    app.RegisterScheduleHandler(&DailyCleanupHandler{})

    app.Run(context.Background())  // Works locally AND on Lambda
}
```

## Why Transire?

**For Go Developers Who Want:**

- ✅ **Zero Framework Lock-in** - Use Chi, standard `http.Handler`, and familiar Go patterns
- ✅ **Local Dev Experience** - Hot reload, simulated queues, instant feedback
- ✅ **Cloud Portability** - Same code runs locally and on AWS Lambda (GCP/Azure coming)
- ✅ **Infrastructure as Code** - Auto-generated CDK, zero config to deploy
- ✅ **Production Features** - Queues (SQS), scheduled tasks (EventBridge), VPC, extensions

**Philosophy:** Stand on the shoulders of giants. Transire uses proven libraries (Chi, Cobra, AWS CDK) instead of reinventing the wheel.

## Quick Start

### 1. Install the CLI

```bash
go install github.com/transire/transire/cmd/transire@latest
```

### 2. Create a New Project

```bash
transire init my-api
cd my-api
```

### 3. Write Standard Go Code

The generated project uses Chi routing with no special abstractions:

```go
package main

import (
    "context"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/transire/transire/pkg/transire"
)

func main() {
    app := transire.New()
    r := app.Router()

    // Standard Chi middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Standard Chi routes
    r.Get("/health", healthHandler)
    r.Post("/api/users", createUserHandler)

    app.Run(context.Background())
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("OK"))
}
```

### 4. Run Locally with Hot Reload

```bash
transire run
```

Visit `http://localhost:3000` - changes auto-reload as you code.

### 5. Deploy to AWS

```bash
# Build Lambda artifacts
transire build

# Deploy infrastructure + code
transire deploy
```

Done! Your API is live on AWS Lambda with API Gateway, auto-scaling, and zero server management.

## Core Features

### HTTP Handlers

Use Chi router directly - no special interfaces or abstractions:

```go
app := transire.New()
r := app.Router()

r.Use(middleware.Logger)
r.Use(middleware.RequestID)

r.Route("/api/v1", func(r chi.Router) {
    r.Get("/users", listUsers)
    r.Post("/users", createUser)
    r.Get("/users/{id}", getUser)
})
```

### Queue Handlers

Process messages from SQS (locally simulated):

```go
type EmailHandler struct{}

func (h *EmailHandler) QueueName() string { return "email-queue" }

func (h *EmailHandler) HandleMessages(ctx context.Context, msgs []transire.Message) ([]string, error) {
    var failed []string
    for _, msg := range msgs {
        if err := sendEmail(msg.Body()); err != nil {
            failed = append(failed, msg.ID())
        }
    }
    return failed, nil  // Failed IDs will be retried
}

func (h *EmailHandler) Config() transire.QueueConfig {
    return transire.QueueConfig{
        VisibilityTimeoutSeconds: 30,
        MaxReceiveCount: 3,
        BatchSize: 10,
    }
}

// Register the handler
app.RegisterQueueHandler(&EmailHandler{})
```

Local testing:
```bash
curl -X POST http://localhost:4000/queues/email-queue \
  -d '{"message": "test@example.com"}'
```

### Scheduled Tasks

Run cron jobs with EventBridge:

```go
type DailyCleanupHandler struct{}

func (h *DailyCleanupHandler) Name() string { return "daily-cleanup" }
func (h *DailyCleanupHandler) Schedule() string { return "0 2 * * *" }  // 2 AM daily

func (h *DailyCleanupHandler) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
    return cleanupOldRecords()
}

func (h *DailyCleanupHandler) Config() transire.ScheduleConfig {
    return transire.ScheduleConfig{
        Timezone: "UTC",
        Enabled: true,
    }
}

app.RegisterScheduleHandler(&DailyCleanupHandler{})
```

Local testing:
```bash
curl -X POST http://localhost:4000/schedules/daily-cleanup
```

## Configuration

Configure your app with `transire.yaml`:

```yaml
name: my-api
language: go
cloud: aws
runtime: lambda
iac: cdk

lambda:
  architecture: arm64
  timeout_seconds: 30
  memory_mb: 512

# Single function (default)
functions:
  main:
    include:
      - http_handlers: "*"
      - queue_handlers: "*"
      - schedule_handlers: "*"

environment:
  LOG_LEVEL: info
  DATABASE_URL: ${DATABASE_URL}

queues:
  email-queue:
    visibility_timeout_seconds: 30
    max_receive_count: 3

schedules:
  daily-cleanup:
    timezone: "America/New_York"
    enabled: true

development:
  http_port: 3000
  queue_port: 4000
  auto_reload: true
```

## Advanced Features

### Multi-Function Architecture

Split handlers into separate Lambda functions for better resource allocation:

```yaml
functions:
  web:
    include:
      - http_handlers: "*"
    memory_mb: 256
    timeout_seconds: 30

  background:
    include:
      - queue_handlers: "*"
      - schedule_handlers: "*"
    memory_mb: 1024
    timeout_seconds: 300
```

### VPC Configuration

Deploy Lambda functions in a VPC:

```yaml
lambda:
  vpc:
    subnet_ids:
      - subnet-abc123
      - subnet-def456
    security_group_ids:
      - sg-xyz789
```

### Use Existing AWS Resources

Reference existing SQS queues and EventBridge rules:

```yaml
queues:
  email-queue:
    existing_queue_arn: arn:aws:sqs:us-east-1:123456789012:my-queue

schedules:
  daily-cleanup:
    existing_rule_arn: arn:aws:events:us-east-1:123456789012:rule/my-rule
```

### Custom CDK Extensions

Extend generated infrastructure with custom CDK code:

```typescript
// infrastructure/extensions/database.ts
import * as rds from 'aws-cdk-lib/aws-rds';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import { Construct } from 'constructs';
import * as cdk from 'aws-cdk-lib';

export function extendStack(stack: cdk.Stack, vpc: ec2.IVpc) {
  const database = new rds.DatabaseInstance(stack, 'Database', {
    engine: rds.DatabaseInstanceEngine.postgres({ version: rds.PostgresEngineVersion.VER_15 }),
    vpc,
    instanceType: ec2.InstanceType.of(ec2.InstanceClass.T3, ec2.InstanceSize.MICRO),
  });

  new cdk.CfnOutput(stack, 'DatabaseEndpoint', {
    value: database.dbInstanceEndpointAddress,
  });
}
```

Enable in config:
```yaml
extensions:
  - path: ./infrastructure/extensions/database.ts
    enabled: true
```

## CLI Commands

```bash
# Create new project
transire init my-app

# Run locally with hot reload
transire run

# Run with custom config
transire run -c custom.yaml

# Build Lambda artifacts
transire build

# Build with custom output
transire build --output ./dist

# Deploy to AWS
transire deploy

# Deploy to specific region/environment
transire deploy --region us-west-2 --environment production

# Preview changes without deploying
transire deploy --dry-run
```

## Architecture

Transire provides three core abstractions:

1. **App** - Main application container with Chi router
2. **Runtime** - Abstracts execution environment (local vs Lambda vs future clouds)
3. **Provider** - Handles cloud-specific building, IaC generation, and deployment

```
┌─────────────────────────────────────────┐
│         Your Application Code           │
│  (Standard Go + Chi + Interfaces)      │
└─────────────────┬───────────────────────┘
                  │
         ┌────────▼─────────┐
         │   Transire App   │
         │  (pkg/transire)  │
         └────────┬─────────┘
                  │
    ┌─────────────┴─────────────┐
    │                           │
┌───▼────────┐          ┌──────▼───────┐
│   Local    │          │    Lambda    │
│  Runtime   │          │   Runtime    │
│ (Hot reload│          │(AWS Adapter) │
│  + Queues) │          │              │
└────────────┘          └──────────────┘
```

### Package Structure

```
pkg/transire/          # Public SDK - import this in your apps
├── app.go            # App with Chi router
├── interfaces.go     # Core interfaces (Handler, Provider, Runtime)
├── runtime.go        # Runtime detection
├── local_runtime.go  # Local dev with hot reload
├── lambda_runtime.go # AWS Lambda adapter
└── config.go         # Configuration system

cmd/transire/         # CLI tool
└── main.go          # Cobra-based CLI

internal/
├── providers/aws/    # AWS implementation
│   ├── provider.go          # Provider interface impl
│   ├── lambda_builder.go    # Build ARM64 binaries
│   ├── cdk_generator.go     # Generate CDK TypeScript
│   └── cdk_deployer.go      # Deploy via CDK CLI
└── cli/
    ├── commands/     # init, run, build, deploy
    ├── runner/       # Hot reload implementation
    └── scaffold/     # Project templates
```

## Examples

Check out the [examples](./examples) directory:

- **[simple-api](./examples/simple-api)** - Basic REST API with queue and schedule handlers
- **[todo-app](./examples/todo-app)** - Complete application with CRUD operations

Each example includes:
- Full source code
- `transire.yaml` configuration
- E2E tests
- README with instructions

## Testing

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/transire
go test ./internal/providers/aws

# Run with coverage
go test -cover ./...

# Run E2E tests in examples
cd examples/simple-api && go test -v
```

## Development

### Setting Up for Development

```bash
# Clone the repository
git clone https://github.com/transire/transire.git
cd transire

# Install dependencies
go mod download

# Install pre-commit hooks (recommended)
./scripts/install-hooks.sh
```

The pre-commit hooks run the same checks as CI to catch issues before pushing.

### Building the CLI

```bash
# Build locally
go build -o transire-cli cmd/transire/main.go

# Install for development
go install ./cmd/transire
```

### Running Examples

```bash
cd examples/simple-api

# Run locally
../../transire-cli run

# Or build and run directly
go build -o simple-api .
./simple-api
```

## Requirements

- **Go**: 1.21 or higher
- **Node.js**: 18+ (for CDK deployment)
- **AWS CLI**: Configured with credentials (for deployment)
- **Docker**: Optional, for local Lambda testing

## Roadmap

### Current (v0.1)
- ✅ Go SDK with Chi integration
- ✅ AWS Lambda runtime support
- ✅ Local development with hot reload
- ✅ Queue handlers (SQS)
- ✅ Scheduled tasks (EventBridge)
- ✅ CDK infrastructure generation
- ✅ Multi-function support
- ✅ VPC configuration
- ✅ Existing resources
- ✅ Custom extensions

### Next (v0.2)
- [ ] Built-in observability (logs, metrics, traces)
- [ ] Local DynamoDB/RDS simulation
- [ ] GitHub Actions workflows
- [ ] API documentation generation
- [ ] Request validation middleware

### Future
- [ ] Rust language support
- [ ] Google Cloud Platform provider (Cloud Run)
- [ ] Azure provider (Azure Functions)
- [ ] OpenTofu IaC alternative
- [ ] WebSocket support
- [ ] GraphQL integration

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

Areas we'd love help with:
- Additional cloud providers (GCP, Azure)
- Language SDKs (Rust, Python, TypeScript)
- Documentation and examples
- Testing and bug reports

## Why "Transire"?

**Transire** (Latin: "to go across, pass through") reflects the project's goal of seamlessly transitioning applications across different runtime environments while maintaining consistent behavior and developer experience.

## License

Mozilla Public License 2.0 - see [LICENSE](./LICENSE) for details.

## Links

- **Documentation**: [transire.github.io](https://transire.github.io/transire) (coming soon)
- **Examples**: [./examples](./examples)
- **Issues**: [GitHub Issues](https://github.com/transire/transire/issues)
- **Discussions**: [GitHub Discussions](https://github.com/transire/transire/discussions)

---

Built with ❤️ for the Go community. Star ⭐ this repo if you find it useful!
