# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Transire is a cloud-agnostic Go application framework and runtime abstraction that enables developers to write applications once and run them consistently across local development and cloud platforms (starting with AWS Lambda). The framework integrates with Chi for HTTP routing and provides abstractions for queue handlers and scheduled tasks.

**Key Philosophy**: Stand on the shoulders of giants by using proven libraries (Chi, Cobra, AWS CDK) rather than reinventing the wheel.

## Development Commands

### Building the CLI

```bash
# Build the CLI tool
go build -o transire-cli cmd/transire/main.go

# Build for specific example
cd examples/simple-api
go build -o simple-api .
```

### Running Applications Locally

```bash
# Run with hot reload (watches for file changes)
./transire-cli run

# Run with custom config
./transire-cli run -c transire.yaml

# Run from example directory
cd examples/simple-api && ../../transire-cli run
```

### Building for Deployment

```bash
# Build artifacts for Lambda (ARM64)
./transire-cli build

# Build with custom output directory
./transire-cli build --output ./dist
```

### Deployment

```bash
# Deploy to AWS (requires AWS credentials)
./transire-cli deploy

# Deploy to specific region and environment
./transire-cli deploy --region us-west-2 --environment production

# Preview changes without applying
./transire-cli deploy --dry-run
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./pkg/transire
go test ./internal/cli/runner

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestAppRouter ./pkg/transire
```

## Architecture

### Core Package Structure

```
pkg/transire/          # Public SDK - user-facing APIs
├── app.go            # Main App abstraction with Chi router
├── interfaces.go     # Core interfaces (QueueHandler, SchedulerHandler, Provider, Runtime)
├── config.go         # Configuration types and YAML unmarshaling
├── runtime.go        # Runtime detection (local vs Lambda vs future clouds)
├── local_runtime.go  # Local development runtime with HTTP server
└── lambda_runtime.go # AWS Lambda runtime adapter

internal/
├── providers/aws/    # AWS-specific implementations
│   ├── provider.go          # AWS Provider interface implementation
│   ├── lambda_builder.go    # Builds Lambda deployment artifacts
│   ├── lambda_runtime.go    # Lambda execution environment adapter
│   ├── cdk_generator.go     # Generates CDK infrastructure code
│   └── cdk_deployer.go      # Deploys via CDK
└── cli/
    ├── commands/     # Cobra command implementations (init, run, build, deploy, dev)
    ├── runner/       # Local dev runner with hot reload
    ├── discovery/    # Handler discovery from user code
    └── scaffold/     # Project initialization templates

cmd/transire/         # CLI entry point using Cobra
examples/
├── simple-api/       # Basic API with queue and schedule handlers
└── todo-app/         # More complete example application
```

### Runtime Architecture

**Runtime Detection Flow**:
1. Check `TRANSIRE_RUNTIME` env var for explicit override
2. Check `AWS_LAMBDA_FUNCTION_NAME` for Lambda
3. Check `K_SERVICE` for Google Cloud Run (future)
4. Check `FUNCTIONS_WORKER_RUNTIME` for Azure Functions (future)
5. Default to local development mode

**Local Runtime** (`pkg/transire/local_runtime.go`):
- Runs HTTP server on configured port (default 3000)
- Provides queue simulator endpoints for testing queue handlers
- Provides schedule simulator endpoints for testing scheduled tasks
- Enables hot reload via file watching

**Lambda Runtime** (`pkg/transire/lambda_runtime.go`):
- Adapts AWS Lambda events to transire abstractions
- Routes API Gateway events to Chi router
- Routes SQS events to registered QueueHandlers
- Routes EventBridge events to registered SchedulerHandlers

### Handler Abstractions

**HTTP Handlers**: Use Chi router directly, no special interface required
```go
r := app.Router()
r.Get("/health", healthHandler)  // Standard http.HandlerFunc
```

**Queue Handlers**: Implement `QueueHandler` interface
```go
type QueueHandler interface {
    HandleMessages(ctx context.Context, messages []Message) ([]string, error)
    QueueName() string
    Config() QueueConfig
}
```
- Process messages in batches
- Return message IDs that failed (will be retried)
- Config defines visibility timeout, max receive count, batch size

**Schedule Handlers**: Implement `SchedulerHandler` interface
```go
type SchedulerHandler interface {
    HandleSchedule(ctx context.Context, event ScheduleEvent) error
    Schedule() string  // Cron expression
    Name() string
    Config() ScheduleConfig
}
```
- Execute on cron schedule
- Config defines timezone, retry attempts, timeout

### Configuration System

Projects use `transire.yaml` for configuration:
- **name**: Project name used for stack naming
- **language**: `go` (rust planned for future)
- **cloud**: `aws` (gcp, azure planned)
- **runtime**: `lambda` (cloudrun, azure-functions planned)
- **iac**: `cdk` (opentofu planned)
- **lambda**: Lambda-specific settings (architecture, timeout, memory)
- **functions**: Function grouping (default: single function for all handlers)
- **environment**: Environment variables
- **queues**: Per-queue configuration
- **schedules**: Per-schedule configuration
- **development**: Local dev settings (ports, auto-reload)

Configuration is loaded via `pkg/transire/config.go` and uses `gopkg.in/yaml.v3` for parsing.

### Provider System

The Provider interface (`pkg/transire/interfaces.go`) abstracts cloud-specific operations:
- `BuildArtifacts()`: Create deployment packages
- `GenerateIaC()`: Generate infrastructure definitions
- `Deploy()`: Apply infrastructure and deploy
- `CreateRuntime()`: Return runtime implementation for execution environment

AWS implementation in `internal/providers/aws/provider.go`:
- Builds ARM64 binaries for Lambda
- Generates AWS CDK TypeScript code
- Deploys via CDK CLI
- Creates Lambda runtime adapter

## Key Implementation Patterns

### Adding HTTP Routes
Use Chi router directly - no framework abstractions:
```go
app := transire.New()
r := app.Router()
r.Use(middleware.Logger)  // Standard Chi middleware
r.Get("/health", healthHandler)  // Standard http.HandlerFunc
```

### Registering Queue Handlers
Create a struct implementing QueueHandler:
```go
type EmailHandler struct{}
func (h *EmailHandler) QueueName() string { return "email-queue" }
func (h *EmailHandler) Config() transire.QueueConfig { /* ... */ }
func (h *EmailHandler) HandleMessages(ctx context.Context, msgs []transire.Message) ([]string, error) {
    // Process messages, return failed IDs
}

app.RegisterQueueHandler(&EmailHandler{})
```

### Registering Schedule Handlers
Create a struct implementing SchedulerHandler:
```go
type CleanupHandler struct{}
func (h *CleanupHandler) Name() string { return "daily-cleanup" }
func (h *CleanupHandler) Schedule() string { return "0 2 * * *" }  // Cron
func (h *CleanupHandler) Config() transire.ScheduleConfig { /* ... */ }
func (h *CleanupHandler) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
    // Execute scheduled task
}

app.RegisterScheduleHandler(&CleanupHandler{})
```

### Running the Application
Single entry point works everywhere:
```go
app := transire.New()
// ... register routes and handlers ...
app.Run(context.Background())  // Auto-detects runtime
```

## Hot Reload Implementation

The CLI's `run` command (`internal/cli/commands/run.go` + `internal/cli/runner/`) provides hot reload:
1. Builds the Go application into a binary
2. Runs it as a subprocess
3. Watches `*.go` and `*.yaml` files using fsnotify
4. On change: kills process, rebuilds, restarts
5. Debounces rapid file changes

## Code Style

- Standard Go conventions (gofmt, golint)
- Minimal abstractions - prefer standard library interfaces
- Interface definitions in `pkg/transire/interfaces.go`
- Implementation code in `internal/` packages (not exposed to users)
- Provider-specific code isolated in `internal/providers/<cloud>/`
- No circular dependencies between packages

## Testing Strategy

- Unit tests for core abstractions: `pkg/transire/*_test.go`
- Builder tests: `internal/cli/runner/builder_test.go`
- E2E tests in example applications: `examples/*/e2e_test.go`
- Use standard `testing` package
- Mock external dependencies (AWS SDK, etc.)

## Future Extensibility

The architecture is designed for future expansion:
- **Languages**: Rust support planned (separate SDK, shared CLI/provider model)
- **Clouds**: GCP, Azure providers following same interface
- **Runtimes**: Cloud Run, Azure Functions, ECS/Fargate
- **IaC**: OpenTofu as alternative to CDK
- **CI**: GitLab, CircleCI templates

New providers implement the `Provider` interface in `internal/providers/<cloud>/`.
