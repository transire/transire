# Transire Design Document

## Overview

Transire is an open-source toolchain and runtime abstraction that enables developers to write cloud-agnostic Go (and eventually Rust) applications that run consistently across local development environments and cloud platforms. The project "stands on the shoulders of giants" by leveraging existing, mature libraries and ecosystems.

## High-Level Goals

- **Runtime Consistency**: The same application code executes logically identical across local development and cloud deployments
- **Cloud Agnostic**: Application code remains independent of cloud providers, runtimes, and infrastructure tools
- **Zero Boilerplate**: Developers use familiar patterns (Chi routing, standard Go idioms) without framework-specific abstractions
- **Extensible Architecture**: Clean separation of concerns enables adding new languages, providers, and runtimes without breaking changes

## Architecture Overview

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │    │  Transire CLI   │    │  Local Shims    │
│     Code        │    │                 │    │                 │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ • Chi Routes    │    │ • init          │    │ • HTTP Server   │
│ • Queue Handlers│◄──►│ • run           │◄──►│ • Queue Sim     │
│ • Schedulers    │    │ • build         │    │ • Scheduler Sim │
│                 │    │ • deploy        │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌─────────────────┐
│ Transire Core   │    │ Provider Plugins│
│                 │    │                 │
├─────────────────┤    ├─────────────────┤
│ • App Registry  │    │ • AWS Lambda    │
│ • Handler Iface │◄──►│ • API Gateway   │
│ • Runtime Detect│    │ • SQS           │
│                 │    │ • EventBridge   │
└─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  IaC Generator  │
                       │                 │
                       ├─────────────────┤
                       │ • CDK (TypeScript)│
                       │ • Auto-generation│
                       │ • User Extensions│
                       └─────────────────┘
```

### Component Interactions

1. **Application Code** registers handlers with the Transire Core using familiar patterns
2. **Transire CLI** discovers handlers, builds artifacts, and manages deployments
3. **Local Shims** provide development-time simulation of cloud services
4. **Provider Plugins** implement cloud-specific runtime and deployment logic
5. **IaC Generator** creates infrastructure definitions from handler registrations

### Dependency Flow

```
Application Code
       │
       ▼
Transire Core ◄──── CLI
       │              │
       ▼              ▼
Provider Plugin ──► IaC Generator
       │
       ▼
Cloud Runtime
```

No circular dependencies exist. The CLI can introspect both application code and core registrations to drive builds and deployments.

## Package Structure

```
transire/
├── pkg/                          # Public APIs
│   ├── transire/                 # Core SDK
│   │   ├── app.go               # Main app abstraction
│   │   ├── handler.go           # Handler interfaces
│   │   ├── runtime.go           # Runtime detection
│   │   └── config.go            # Configuration types
│   ├── providers/               # Provider interfaces
│   │   └── provider.go         # Provider interface definition
│   └── queue/                   # Queue message types
│       └── message.go
├── internal/                    # Internal implementation
│   ├── providers/              # Provider implementations
│   │   └── aws/               # AWS-specific code
│   │       ├── lambda/        # Lambda runtime support
│   │       ├── apigateway/    # API Gateway integration
│   │       ├── sqs/           # SQS integration
│   │       └── eventbridge/   # EventBridge integration
│   ├── local/                 # Local development shims
│   │   ├── http/             # Local HTTP server
│   │   ├── queue/            # Queue simulator
│   │   └── scheduler/        # Scheduler simulator
│   ├── build/                # Build system
│   │   └── lambda/           # Lambda packaging
│   ├── iac/                  # Infrastructure as Code
│   │   └── cdk/              # CDK generation
│   └── cli/                  # CLI internals
│       ├── discovery/        # Project discovery
│       ├── scaffold/         # Project scaffolding
│       └── commands/         # Command implementations
├── cmd/                       # CLI commands
│   └── transire/             # Main CLI entry point
│       └── main.go
├── examples/                  # Example applications
│   ├── simple-api/           # Basic HTTP API example
│   ├── queue-processor/      # Queue handling example
│   └── full-app/            # Complete application example
└── tools/                    # Development tools
    └── codegen/              # Code generation utilities
```

### Package Boundaries

- **`pkg/`**: Public APIs that applications import
- **`internal/providers/aws/`**: All AWS-specific code isolated here
- **`internal/local/`**: Local development shims (never included in cloud deployments)
- **`cmd/transire/`**: CLI entry point
- **`internal/cli/`**: CLI implementation details

## Core Interfaces and Types

### Handler Interfaces

```go
// pkg/transire/handler.go

package transire

import (
    "context"
    "net/http"
    "github.com/your-org/transire/pkg/queue"
)

// HTTPHandler is a standard http.Handler with metadata
type HTTPHandler interface {
    http.Handler
    // Optional: handler can provide metadata
    Metadata() HTTPHandlerMetadata
}

// HTTPHandlerMetadata provides routing and configuration info
type HTTPHandlerMetadata struct {
    Methods     []string          // HTTP methods (GET, POST, etc.)
    Path        string           // Route path
    Middlewares []Middleware     // Applied middlewares
}

// QueueHandler processes messages in batches
type QueueHandler interface {
    // HandleMessages processes a batch of messages
    // Returns message IDs that failed and should be retried
    HandleMessages(ctx context.Context, messages []queue.Message) ([]string, error)

    // QueueName returns the logical queue name
    QueueName() string

    // Configuration for the queue
    Config() QueueConfig
}

// SchedulerHandler handles scheduled/cron events
type SchedulerHandler interface {
    // HandleSchedule executes on the defined schedule
    HandleSchedule(ctx context.Context, event ScheduleEvent) error

    // Schedule returns cron expression or interval
    Schedule() string

    // Name returns unique identifier for this scheduled task
    Name() string
}

// Message represents a queue message
type Message interface {
    ID() string
    Body() []byte
    Attributes() map[string]string
}

// ScheduleEvent provides context for scheduled execution
type ScheduleEvent struct {
    ScheduledTime time.Time
    Name          string
    Payload       []byte
}
```

### App Registration

```go
// pkg/transire/app.go

package transire

import (
    "context"
    "github.com/go-chi/chi/v5"
)

// App is the main application abstraction
type App struct {
    router        *chi.Mux
    queueHandlers []QueueHandler
    schedHandlers []SchedulerHandler
    config        *Config
    provider      providers.Provider
}

// New creates a new Transire application
func New(opts ...Option) *App {
    app := &App{
        router: chi.NewRouter(),
        config: defaultConfig(),
    }

    for _, opt := range opts {
        opt(app)
    }

    return app
}

// Router returns the Chi router for HTTP route registration
func (a *App) Router() *chi.Mux {
    return a.router
}

// RegisterQueueHandler adds a queue handler
func (a *App) RegisterQueueHandler(handler QueueHandler) {
    a.queueHandlers = append(a.queueHandlers, handler)
}

// RegisterScheduleHandler adds a scheduled handler
func (a *App) RegisterScheduleHandler(handler SchedulerHandler) {
    a.schedHandlers = append(a.schedHandlers, handler)
}

// Run starts the application (local or cloud)
func (a *App) Run(ctx context.Context) error {
    // Runtime detection
    runtime := detectRuntime()

    switch runtime {
    case RuntimeLocal:
        return a.runLocal(ctx)
    case RuntimeAWSLambda:
        return a.runLambda(ctx)
    default:
        return fmt.Errorf("unsupported runtime: %v", runtime)
    }
}

// runLocal starts local development shims
func (a *App) runLocal(ctx context.Context) error {
    // Start HTTP server with Chi router
    // Start queue simulator
    // Start scheduler simulator
    // Block until context cancellation
}

// runLambda handles Lambda events
func (a *App) runLambda(ctx context.Context) error {
    // AWS Lambda runtime integration
    // Event routing to appropriate handlers
}
```

### Provider Interface

```go
// pkg/providers/provider.go

package providers

// Provider abstracts cloud provider implementations
type Provider interface {
    // Name returns provider identifier (aws, gcp, azure)
    Name() string

    // Runtime returns supported runtime (lambda, ecs, cloudrun)
    Runtime() string

    // BuildArtifacts creates deployable artifacts
    BuildArtifacts(ctx context.Context, config BuildConfig) error

    // GenerateIaC creates infrastructure definitions
    GenerateIaC(ctx context.Context, config IaCConfig) error

    // Deploy applies infrastructure and artifacts
    Deploy(ctx context.Context, config DeployConfig) error
}

// BuildConfig configures artifact building
type BuildConfig struct {
    AppPath      string
    OutputDir    string
    Architecture string // arm64 for Lambda
    Environment  map[string]string
}

// IaCConfig configures infrastructure generation
type IaCConfig struct {
    StackName     string
    HTTPHandlers  []HTTPHandlerSpec
    QueueHandlers []QueueHandlerSpec
    ScheduleHandlers []ScheduleHandlerSpec
    Extensions    map[string]interface{} // User customizations
}
```

### Runtime Detection

```go
// pkg/transire/runtime.go

package transire

import "os"

// Runtime represents the execution environment
type Runtime string

const (
    RuntimeLocal     Runtime = "local"
    RuntimeAWSLambda Runtime = "aws_lambda"
    RuntimeGCPRun    Runtime = "gcp_cloudrun"  // Future
)

// detectRuntime determines current execution environment
func detectRuntime() Runtime {
    // Check for AWS Lambda environment
    if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
        return RuntimeAWSLambda
    }

    // Check for Google Cloud Run environment
    if os.Getenv("K_SERVICE") != "" {
        return RuntimeGCPRun
    }

    // Default to local development
    return RuntimeLocal
}
```

## Example Developer Experience

### Simple HTTP API

```go
// main.go
package main

import (
    "context"
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/your-org/transire/pkg/transire"
)

func main() {
    // Create Transire app
    app := transire.New()

    // Get Chi router - use exactly like normal Chi
    r := app.Router()

    // Standard Chi middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Standard Chi routes
    r.Get("/health", healthHandler)
    r.Post("/users", createUserHandler)
    r.Get("/users/{id}", getUserHandler)

    // Run the app (works locally and in Lambda)
    if err := app.Run(context.Background()); err != nil {
        panic(err)
    }
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // Standard HTTP handler implementation
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "id")
    // Handle user retrieval
}
```

### Queue Handler Example

```go
// handlers/email.go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/your-org/transire/pkg/queue"
    "github.com/your-org/transire/pkg/transire"
)

// EmailQueueHandler processes email sending requests
type EmailQueueHandler struct{}

func (h *EmailQueueHandler) QueueName() string {
    return "email-queue"
}

func (h *EmailQueueHandler) Config() transire.QueueConfig {
    return transire.QueueConfig{
        VisibilityTimeoutSeconds: 30,
        MaxReceiveCount:         3,
        BatchSize:              10,
    }
}

func (h *EmailQueueHandler) HandleMessages(ctx context.Context, messages []queue.Message) ([]string, error) {
    var failedIDs []string

    for _, msg := range messages {
        var emailReq EmailRequest
        if err := json.Unmarshal(msg.Body(), &emailReq); err != nil {
            // Invalid message format, don't retry
            continue
        }

        if err := sendEmail(emailReq); err != nil {
            // Failed to send, add to retry list
            failedIDs = append(failedIDs, msg.ID())
        }
    }

    return failedIDs, nil
}

type EmailRequest struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

func sendEmail(req EmailRequest) error {
    // Email sending logic
    return nil
}

// Register the handler
func init() {
    app := transire.GetApp() // Get global app instance
    app.RegisterQueueHandler(&EmailQueueHandler{})
}
```

### Scheduler Handler Example

```go
// handlers/cleanup.go
package handlers

import (
    "context"
    "log"
    "time"

    "github.com/your-org/transire/pkg/transire"
)

// CleanupHandler runs daily database cleanup
type CleanupHandler struct{}

func (h *CleanupHandler) Name() string {
    return "daily-cleanup"
}

func (h *CleanupHandler) Schedule() string {
    return "0 2 * * *" // Daily at 2 AM UTC
}

func (h *CleanupHandler) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
    log.Printf("Running cleanup at %v", event.ScheduledTime)

    // Perform cleanup operations
    if err := cleanupOldData(); err != nil {
        return fmt.Errorf("cleanup failed: %w", err)
    }

    log.Println("Cleanup completed successfully")
    return nil
}

func cleanupOldData() error {
    // Database cleanup logic
    return nil
}

// Register the handler
func init() {
    app := transire.GetApp()
    app.RegisterScheduleHandler(&CleanupHandler{})
}
```

## CLI Design and Behavior

### Command Structure

```
transire
├── init [flags]           # Initialize new project
├── run [flags]            # Run locally
├── build [flags]          # Build artifacts
├── deploy [flags]         # Deploy to cloud
└── dev                    # Development utilities
    ├── queues
    │   ├── list          # List registered queues
    │   └── send <queue> <message>  # Send test message
    └── schedules
        ├── list          # List registered schedules
        └── execute <schedule>      # Trigger schedule locally
```

### `transire init`

```bash
# Create new project with defaults
$ transire init my-app

# With options
$ transire init my-app --lang=go --cloud=aws --runtime=lambda --iac=cdk --ci=github
```

**Generated Project Structure:**

```
my-app/
├── main.go                    # Minimal Transire app
├── go.mod                     # Go module
├── transire.yaml             # Transire configuration
├── infrastructure/           # CDK TypeScript app
│   ├── app.ts               # CDK entry point
│   ├── lib/
│   │   └── my-app-stack.ts  # Generated stack
│   ├── package.json
│   └── tsconfig.json
└── .github/
    └── workflows/
        └── deploy.yml       # GitHub Actions workflow
```

**Generated `main.go`:**

```go
package main

import (
    "context"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/your-org/transire/pkg/transire"
)

func main() {
    app := transire.New()

    r := app.Router()
    r.Use(middleware.Logger)
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, Transire!"))
    })

    if err := app.Run(context.Background()); err != nil {
        panic(err)
    }
}
```

**Generated `transire.yaml`:**

```yaml
# Transire configuration
name: my-app
language: go
cloud: aws
runtime: lambda
iac: cdk

# Lambda configuration
lambda:
  architecture: arm64
  timeout_seconds: 30
  memory_mb: 128

# Function grouping (optional)
functions:
  main:
    include:
      - http_handlers: "*"
      - queue_handlers: "*"
      - schedule_handlers: "*"

# Environment variables
environment:
  NODE_ENV: production

# VPC configuration (optional)
# vpc:
#   subnet_ids: ["subnet-12345", "subnet-67890"]
#   security_group_ids: ["sg-abcdef"]

# Existing infrastructure references (optional)
# existing_resources:
#   dynamodb_tables:
#     - name: users-table
#       arn: "arn:aws:dynamodb:us-east-1:123456789:table/users"
#   s3_buckets:
#     - name: uploads-bucket
#       arn: "arn:aws:s3:::my-uploads-bucket"
```

### `transire run`

Starts local development environment:

1. Scans project for handler registrations
2. Starts HTTP server using Chi router (port 3000)
3. Starts queue simulator (port 4000)
4. Starts scheduler simulator
5. Watches for file changes and reloads

```bash
$ transire run
[INFO] Transire v1.0.0 starting in local mode
[INFO] Discovered handlers:
[INFO]   HTTP: 3 routes
[INFO]   Queues: 2 handlers (email-queue, notification-queue)
[INFO]   Schedules: 1 handler (daily-cleanup)
[INFO] Starting HTTP server on :3000
[INFO] Starting queue simulator on :4000
[INFO] Starting scheduler simulator
[INFO] Ready! Press Ctrl+C to stop
```

### `transire build`

Creates deployment artifacts:

1. Compiles Go code for ARM64 architecture
2. Excludes local development dependencies using build tags
3. Generates handler manifest
4. Creates Lambda ZIP packages
5. Updates CDK infrastructure definitions

```bash
$ transire build
[INFO] Building artifacts for AWS Lambda (ARM64)
[INFO] Excluding local development code
[INFO] Discovered handlers:
[INFO]   HTTP: 3 routes → API Gateway integration
[INFO]   Queues: 2 handlers → SQS integration
[INFO]   Schedules: 1 handler → EventBridge integration
[INFO] Creating Lambda package: dist/my-app.zip (2.3 MB)
[INFO] Generating CDK infrastructure
[INFO] Updated: infrastructure/lib/my-app-stack.ts
[INFO] Build completed successfully
```

### `transire deploy`

Deploys to cloud provider:

```bash
$ transire deploy
[INFO] Deploying to AWS using CDK
[INFO] Stack: my-app-stack
[INFO] Synthesizing CloudFormation template
[INFO] Deploying resources:
[INFO]   ✓ Lambda function: my-app-main
[INFO]   ✓ API Gateway: my-app-api
[INFO]   ✓ SQS queues: email-queue, notification-queue
[INFO]   ✓ EventBridge rules: daily-cleanup
[INFO] Deployment completed
[INFO] API endpoint: https://abc123.execute-api.us-east-1.amazonaws.com/prod
```

### CLI Architecture

```go
// cmd/transire/main.go
package main

import (
    "github.com/spf13/cobra"
    "github.com/your-org/transire/internal/cli/commands"
)

func main() {
    root := &cobra.Command{
        Use:   "transire",
        Short: "Cloud-agnostic application runtime",
    }

    root.AddCommand(commands.InitCommand())
    root.AddCommand(commands.RunCommand())
    root.AddCommand(commands.BuildCommand())
    root.AddCommand(commands.DeployCommand())
    root.AddCommand(commands.DevCommand())

    root.Execute()
}
```

The CLI uses project discovery to:
1. Detect project type (Go module, Rust crate, etc.)
2. Parse `transire.yaml` configuration
3. Introspect handler registrations
4. Coordinate builds and deployments

## Lambda Packaging and Routing

### Default Single Function Model

By default, all handlers are packaged into a single Lambda function:

```
Lambda Function: my-app-main
├── HTTP Routes (via API Gateway events)
├── Queue Handlers (via SQS events)
└── Schedule Handlers (via EventBridge events)
```

**Runtime Event Routing:**

```go
// internal/providers/aws/lambda/runtime.go
package lambda

import (
    "context"
    "encoding/json"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
)

// Handler is the Lambda entry point
type Handler struct {
    app *transire.App
}

func (h *Handler) Handle(ctx context.Context, event json.RawMessage) (interface{}, error) {
    // Detect event type and route accordingly
    if isAPIGatewayEvent(event) {
        return h.handleHTTP(ctx, event)
    } else if isSQSEvent(event) {
        return h.handleQueue(ctx, event)
    } else if isEventBridgeEvent(event) {
        return h.handleSchedule(ctx, event)
    }

    return nil, fmt.Errorf("unknown event type")
}

func (h *Handler) handleHTTP(ctx context.Context, event json.RawMessage) (events.APIGatewayV2HTTPResponse, error) {
    var apiEvent events.APIGatewayV2HTTPRequest
    if err := json.Unmarshal(event, &apiEvent); err != nil {
        return events.APIGatewayV2HTTPResponse{}, err
    }

    // Convert to http.Request and route through Chi
    req := convertAPIGatewayToHTTPRequest(apiEvent)

    // Use response recorder to capture response
    recorder := httptest.NewRecorder()
    h.app.Router().ServeHTTP(recorder, req)

    return convertHTTPResponseToAPIGateway(recorder.Result()), nil
}

func (h *Handler) handleQueue(ctx context.Context, event json.RawMessage) (events.SQSBatchResponse, error) {
    var sqsEvent events.SQSEvent
    if err := json.Unmarshal(event, &sqsEvent); err != nil {
        return events.SQSBatchResponse{}, err
    }

    // Convert to queue.Message and find appropriate handler
    messages := convertSQSToMessages(sqsEvent.Records)
    queueName := extractQueueName(sqsEvent.Records[0])

    handler := h.app.FindQueueHandler(queueName)
    if handler == nil {
        return events.SQSBatchResponse{}, fmt.Errorf("no handler for queue: %s", queueName)
    }

    failedIDs, err := handler.HandleMessages(ctx, messages)
    if err != nil {
        return events.SQSBatchResponse{}, err
    }

    // Convert failed IDs to SQS batch response format
    return events.SQSBatchResponse{
        BatchItemFailures: convertFailedIDsToBatchFailures(failedIDs),
    }, nil
}
```

### Advanced Function Grouping

Configuration allows splitting handlers across multiple Lambda functions:

```yaml
# transire.yaml
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
    memory_mb: 512
    timeout_seconds: 300

  critical:
    include:
      - queue_handlers: ["payment-queue", "order-queue"]
    memory_mb: 1024
    timeout_seconds: 60
    reserved_concurrency: 10
```

This generates multiple Lambda functions with appropriate IAM roles and triggers.

## Queue Handler Semantics

### Batch Processing Model

Queue handlers process messages in batches and return failed message IDs:

```go
type QueueHandler interface {
    HandleMessages(ctx context.Context, messages []queue.Message) ([]string, error)
    QueueName() string
    Config() QueueConfig
}

type QueueConfig struct {
    VisibilityTimeoutSeconds int    // How long messages are invisible after delivery
    MaxReceiveCount         int     // Max delivery attempts before DLQ
    BatchSize              int     // Max messages per batch (1-10 for SQS)
    WaitTimeSeconds        int     // Long polling wait time
}
```

### AWS SQS Integration

Maps to SQS partial batch failure semantics:

```go
// Handler returns failed message IDs
failedIDs, err := handler.HandleMessages(ctx, messages)

// Transire converts to SQS batch response
response := events.SQSBatchResponse{
    BatchItemFailures: []events.SQSBatchItemFailure{},
}

for _, id := range failedIDs {
    response.BatchItemFailures = append(response.BatchItemFailures,
        events.SQSBatchItemFailure{ItemIdentifier: id})
}

return response, nil
```

### Local Queue Simulation

For local development, a queue simulator provides the same semantics:

```go
// internal/local/queue/simulator.go
package queue

type Simulator struct {
    queues map[string]*SimulatedQueue
}

type SimulatedQueue struct {
    name     string
    messages chan queue.Message
    handler  transire.QueueHandler
    dlq      chan queue.Message // Dead letter queue
}

func (s *Simulator) SendMessage(queueName string, body []byte) error {
    q := s.queues[queueName]
    msg := &Message{
        id:   generateID(),
        body: body,
        attributes: make(map[string]string),
        deliveryCount: 0,
    }

    select {
    case q.messages <- msg:
        return nil
    default:
        return fmt.Errorf("queue full")
    }
}

func (s *Simulator) processQueue(queueName string) {
    q := s.queues[queueName]

    for {
        // Collect messages into batch
        var batch []queue.Message
        timeout := time.After(time.Second)

        for len(batch) < q.handler.Config().BatchSize {
            select {
            case msg := <-q.messages:
                batch = append(batch, msg)
            case <-timeout:
                break
            }
        }

        if len(batch) == 0 {
            continue
        }

        // Process batch
        failedIDs, err := q.handler.HandleMessages(context.Background(), batch)
        if err != nil {
            // Requeue all messages
            for _, msg := range batch {
                s.requeueMessage(q, msg)
            }
            continue
        }

        // Requeue only failed messages
        for _, msg := range batch {
            if contains(failedIDs, msg.ID()) {
                s.requeueMessage(q, msg)
            }
        }
    }
}
```

### Future Cloud Compatibility

The same interface generalizes to other cloud providers:

- **Google Pub/Sub**: Batch acknowledgment with message IDs
- **Azure Service Bus**: Batch processing with selective acknowledgment
- **Apache Kafka**: Partition-based processing with offset management

## Infrastructure as Code (CDK Integration)

### Auto-Generation Approach

Transire generates TypeScript CDK code from handler registrations:

```typescript
// infrastructure/lib/my-app-stack.ts (generated)
import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';

export class MyAppStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Lambda function
    const mainFunction = new lambda.Function(this, 'MainFunction', {
      runtime: lambda.Runtime.PROVIDED_AL2023,
      architecture: lambda.Architecture.ARM_64,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../dist/my-app.zip'),
      timeout: cdk.Duration.seconds(30),
      memorySize: 128,
      environment: {
        NODE_ENV: 'production',
      },
    });

    // API Gateway v2
    const api = new apigatewayv2.HttpApi(this, 'HttpApi', {
      defaultIntegration: new integrations.HttpLambdaIntegration(
        'DefaultIntegration',
        mainFunction
      ),
    });

    // SQS Queues
    const emailQueue = new sqs.Queue(this, 'EmailQueue', {
      queueName: 'email-queue',
      visibilityTimeout: cdk.Duration.seconds(30),
      deadLetterQueue: {
        queue: new sqs.Queue(this, 'EmailDLQ'),
        maxReceiveCount: 3,
      },
    });

    // SQS -> Lambda event source
    mainFunction.addEventSource(
      new lambda.SqsEventSource(emailQueue, {
        batchSize: 10,
        reportBatchItemFailures: true,
      })
    );

    // EventBridge rules
    const dailyRule = new events.Rule(this, 'DailyCleanupRule', {
      schedule: events.Schedule.cron({ minute: '0', hour: '2' }),
    });
    dailyRule.addTarget(new targets.LambdaFunction(mainFunction));

    // Outputs
    new cdk.CfnOutput(this, 'ApiEndpoint', {
      value: api.apiEndpoint,
    });
  }
}
```

### User Customization

Configuration supports extending generated infrastructure:

```yaml
# transire.yaml
environment:
  DATABASE_URL: "postgres://..."
  S3_BUCKET: !Ref UploadsBucket

vpc:
  subnet_ids: ["subnet-12345", "subnet-67890"]
  security_group_ids: ["sg-abcdef"]

existing_resources:
  dynamodb_tables:
    - name: users-table
      arn: "arn:aws:dynamodb:us-east-1:123456789:table/users"
  s3_buckets:
    - name: uploads-bucket
      arn: "arn:aws:s3:::my-uploads-bucket"

# CDK extension hooks
cdk_extensions:
  - file: "extensions/database.ts"
  - file: "extensions/monitoring.ts"
```

**Extension Example:**

```typescript
// infrastructure/extensions/database.ts
import * as cdk from 'aws-cdk-lib';
import * as rds from 'aws-cdk-lib/aws-rds';
import * as ec2 from 'aws-cdk-lib/aws-ec2';

export function addDatabaseResources(
  stack: cdk.Stack,
  mainFunction: lambda.Function
) {
  const vpc = ec2.Vpc.fromLookup(stack, 'ExistingVpc', {
    vpcId: 'vpc-12345',
  });

  const database = new rds.DatabaseInstance(stack, 'Database', {
    engine: rds.DatabaseInstanceEngine.postgres({
      version: rds.PostgresEngineVersion.VER_15,
    }),
    instanceType: ec2.InstanceType.of(ec2.InstanceClass.T3, ec2.InstanceSize.MICRO),
    vpc,
    credentials: rds.Credentials.fromGeneratedSecret('dbadmin'),
  });

  // Grant Lambda access to database
  database.connections.allowDefaultPortFrom(mainFunction);

  // Add database endpoint to Lambda environment
  mainFunction.addEnvironment('DATABASE_ENDPOINT', database.instanceEndpoint.hostname);
}
```

### Multi-Function Support

For complex function grouping, CDK generates multiple functions:

```typescript
// Generated for function groups
const webFunction = new lambda.Function(this, 'WebFunction', {
  // HTTP handlers only
});

const backgroundFunction = new lambda.Function(this, 'BackgroundFunction', {
  // Queue and schedule handlers
  timeout: cdk.Duration.minutes(5),
  memorySize: 512,
});

// API Gateway only connects to web function
const api = new apigatewayv2.HttpApi(this, 'HttpApi', {
  defaultIntegration: new integrations.HttpLambdaIntegration(
    'WebIntegration',
    webFunction
  ),
});

// SQS and EventBridge connect to background function
emailQueue.grantConsumeMessages(backgroundFunction);
backgroundFunction.addEventSource(new lambda.SqsEventSource(emailQueue));
```

## Configuration System

### YAML Configuration

Using YAML for human-friendly configuration:

```yaml
# transire.yaml - Complete example
name: my-app
language: go
cloud: aws
runtime: lambda
iac: cdk
ci: github

# Lambda function configuration
lambda:
  architecture: arm64
  timeout_seconds: 30
  memory_mb: 128

# Function grouping
functions:
  main:
    include:
      - http_handlers: "*"
      - queue_handlers: "*"
      - schedule_handlers: "*"
  # Alternative: split into multiple functions
  # web:
  #   include:
  #     - http_handlers: "*"
  #   memory_mb: 256
  # background:
  #   include:
  #     - queue_handlers: "*"
  #     - schedule_handlers: "*"
  #   memory_mb: 512
  #   timeout_seconds: 300

# Environment variables
environment:
  NODE_ENV: production
  DATABASE_URL: ${DATABASE_URL} # From environment
  API_KEY: !Ref ApiKeySecret     # CloudFormation reference

# VPC configuration
vpc:
  subnet_ids:
    - subnet-12345
    - subnet-67890
  security_group_ids:
    - sg-abcdef

# Reference existing AWS resources
existing_resources:
  dynamodb_tables:
    - name: users-table
      arn: "arn:aws:dynamodb:us-east-1:123456789:table/users"
      permissions: ["read", "write"]
  s3_buckets:
    - name: uploads-bucket
      arn: "arn:aws:s3:::my-uploads-bucket"
      permissions: ["read", "write"]
  secrets:
    - name: api-key
      arn: "arn:aws:secretsmanager:us-east-1:123456789:secret:api-key"

# Queue-specific configuration
queues:
  email-queue:
    visibility_timeout_seconds: 60
    max_receive_count: 5
    batch_size: 5
  notification-queue:
    visibility_timeout_seconds: 30
    max_receive_count: 3
    batch_size: 10

# Schedule-specific configuration
schedules:
  daily-cleanup:
    timezone: "America/New_York"
    enabled: true
  hourly-metrics:
    timezone: "UTC"
    enabled: false

# CDK extensions
cdk_extensions:
  - file: "extensions/database.ts"
  - file: "extensions/monitoring.ts"
  - file: "extensions/alarms.ts"

# Development settings
development:
  http_port: 3000
  queue_port: 4000
  auto_reload: true
  log_level: debug
```

### Configuration Validation

```go
// pkg/transire/config.go
package transire

import (
    "fmt"
    "gopkg.in/yaml.v3"
)

type Config struct {
    Name     string `yaml:"name"`
    Language string `yaml:"language"`
    Cloud    string `yaml:"cloud"`
    Runtime  string `yaml:"runtime"`
    IaC      string `yaml:"iac"`
    CI       string `yaml:"ci"`

    Lambda      LambdaConfig              `yaml:"lambda"`
    Functions   map[string]FunctionConfig `yaml:"functions"`
    Environment map[string]string         `yaml:"environment"`
    VPC         *VPCConfig                `yaml:"vpc,omitempty"`

    ExistingResources ExistingResourcesConfig `yaml:"existing_resources"`
    Queues           map[string]QueueConfig   `yaml:"queues"`
    Schedules        map[string]ScheduleConfig `yaml:"schedules"`

    CDKExtensions []ExtensionConfig `yaml:"cdk_extensions"`
    Development   DevelopmentConfig `yaml:"development"`
}

type LambdaConfig struct {
    Architecture string `yaml:"architecture"`
    TimeoutSeconds int  `yaml:"timeout_seconds"`
    MemoryMB      int   `yaml:"memory_mb"`
}

type FunctionConfig struct {
    Include          []IncludeSpec     `yaml:"include"`
    MemoryMB        int               `yaml:"memory_mb,omitempty"`
    TimeoutSeconds  int               `yaml:"timeout_seconds,omitempty"`
    ReservedConcurrency *int          `yaml:"reserved_concurrency,omitempty"`
    Environment     map[string]string `yaml:"environment,omitempty"`
}

type IncludeSpec struct {
    HTTPHandlers     interface{} `yaml:"http_handlers,omitempty"`     // "*" or []string
    QueueHandlers    interface{} `yaml:"queue_handlers,omitempty"`    // "*" or []string
    ScheduleHandlers interface{} `yaml:"schedule_handlers,omitempty"` // "*" or []string
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    if err := config.Validate(); err != nil {
        return nil, err
    }

    return &config, nil
}

func (c *Config) Validate() error {
    if c.Name == "" {
        return fmt.Errorf("name is required")
    }

    if c.Language != "go" {
        return fmt.Errorf("unsupported language: %s", c.Language)
    }

    if c.Cloud != "aws" {
        return fmt.Errorf("unsupported cloud: %s", c.Cloud)
    }

    // Additional validation...
    return nil
}
```

## Extensibility and Future Features

### Language Extensibility

The CLI can detect and handle multiple project types:

```go
// internal/cli/discovery/project.go
package discovery

type ProjectType string

const (
    ProjectTypeGo   ProjectType = "go"
    ProjectTypeRust ProjectType = "rust"
    // Future: python, typescript, java, etc.
)

type Project struct {
    Type     ProjectType
    Path     string
    Config   *transire.Config
    Handlers []HandlerRegistration
}

type LanguageAdapter interface {
    // Detect if this directory contains a project of this language
    Detect(dir string) bool

    // Discover handler registrations in the project
    DiscoverHandlers(project *Project) ([]HandlerRegistration, error)

    // Build deployable artifacts
    Build(project *Project, config BuildConfig) error

    // Get runtime requirements
    GetRuntime(project *Project) RuntimeRequirements
}

// Registry of language adapters
var adapters = map[ProjectType]LanguageAdapter{
    ProjectTypeGo:   &GoAdapter{},
    ProjectTypeRust: &RustAdapter{}, // Future
}

func DiscoverProject(dir string) (*Project, error) {
    for projectType, adapter := range adapters {
        if adapter.Detect(dir) {
            return &Project{
                Type: projectType,
                Path: dir,
            }, nil
        }
    }
    return nil, fmt.Errorf("no supported project found in %s", dir)
}
```

**Go Adapter Implementation:**

```go
// internal/cli/discovery/go.go
type GoAdapter struct{}

func (a *GoAdapter) Detect(dir string) bool {
    _, err := os.Stat(filepath.Join(dir, "go.mod"))
    return err == nil
}

func (a *GoAdapter) DiscoverHandlers(project *Project) ([]HandlerRegistration, error) {
    // Parse Go code to find handler registrations
    // Could use go/ast or runtime introspection
    return a.parseGoHandlers(project.Path)
}

func (a *GoAdapter) Build(project *Project, config BuildConfig) error {
    // Cross-compile for ARM64 Lambda
    cmd := exec.Command("go", "build",
        "-tags", "!local", // Exclude local development code
        "-ldflags", "-s -w", // Strip debug info
        "-o", "bootstrap", // Lambda requires this name
        ".")
    cmd.Env = append(os.Environ(),
        "GOOS=linux",
        "GOARCH=arm64",
        "CGO_ENABLED=0",
    )
    cmd.Dir = project.Path

    return cmd.Run()
}
```

### Provider Extensibility

New cloud providers implement the Provider interface:

```go
// internal/providers/gcp/provider.go
package gcp

type Provider struct {
    projectID string
    region    string
}

func (p *Provider) Name() string {
    return "gcp"
}

func (p *Provider) Runtime() string {
    return "cloudrun"
}

func (p *Provider) BuildArtifacts(ctx context.Context, config BuildConfig) error {
    // Build container images for Cloud Run
    return p.buildContainerImage(config)
}

func (p *Provider) GenerateIaC(ctx context.Context, config IaCConfig) error {
    // Generate Terraform/Pulumi for GCP resources
    return p.generateTerraform(config)
}
```

### Future NoSQL Abstraction

Design for table and change stream handlers:

```go
// Future: pkg/transire/table.go
package transire

// TableHandler manages NoSQL table operations
type TableHandler interface {
    TableName() string
    Schema() TableSchema
    Indexes() []IndexDefinition
}

// ChangeStreamHandler processes table change events
type ChangeStreamHandler interface {
    TableName() string
    HandleChanges(ctx context.Context, changes []ChangeEvent) error
}

type ChangeEvent struct {
    EventType   string // INSERT, UPDATE, DELETE
    NewData     map[string]interface{}
    OldData     map[string]interface{}
    Keys        map[string]interface{}
    Timestamp   time.Time
}

// Registration
func (a *App) RegisterTableHandler(handler TableHandler) {
    a.tableHandlers = append(a.tableHandlers, handler)
}

func (a *App) RegisterChangeStreamHandler(handler ChangeStreamHandler) {
    a.changeStreamHandlers = append(a.changeStreamHandlers, handler)
}
```

**Provider-specific implementations:**

```go
// AWS DynamoDB
type DynamoDBTableHandler struct {
    tableName string
    schema    TableSchema
}

// Google Firestore
type FirestoreTableHandler struct {
    collection string
    schema     TableSchema
}
```

### Runtime Extensibility

Support for additional compute runtimes:

```go
// internal/providers/aws/ecs/provider.go
type ECSProvider struct {
    clusterName string
    vpcConfig   VPCConfig
}

func (p *ECSProvider) Runtime() string {
    return "ecs"
}

func (p *ECSProvider) BuildArtifacts(ctx context.Context, config BuildConfig) error {
    // Build container image for ECS
    return p.buildDockerImage(config)
}

func (p *ECSProvider) GenerateIaC(ctx context.Context, config IaCConfig) error {
    // Generate ECS service, task definition, ALB, etc.
    return p.generateECSResources(config)
}
```

For ECS deployment:
- HTTP handlers → ALB + ECS Service
- Queue handlers → SQS + ECS Tasks (background processing)
- Schedule handlers → EventBridge + ECS Scheduled Tasks

### IaC Extensibility

Support for multiple IaC tools:

```go
// pkg/iac/generator.go
package iac

type Generator interface {
    Name() string // "cdk", "terraform", "pulumi"
    Generate(ctx context.Context, config GenerateConfig) error
    Deploy(ctx context.Context, config DeployConfig) error
}

// internal/iac/terraform/generator.go
type TerraformGenerator struct{}

func (g *TerraformGenerator) Generate(ctx context.Context, config GenerateConfig) error {
    // Generate .tf files from handler specs
    return g.generateTerraformFiles(config)
}
```

## MVP Boundaries

### Implemented in MVP

**Core Framework:**
- ✅ Go language support
- ✅ HTTP, Queue, and Scheduler handler abstractions
- ✅ Chi router integration
- ✅ Runtime detection (local vs AWS Lambda)
- ✅ Local development shims

**AWS Provider:**
- ✅ AWS Lambda runtime (ARM64 only)
- ✅ API Gateway v2 integration
- ✅ SQS integration with partial batch failure
- ✅ EventBridge integration
- ✅ Single function packaging (default)
- ✅ Multi-function grouping (advanced)

**CLI Tool:**
- ✅ `transire init` with Go/AWS/Lambda/CDK/GitHub defaults
- ✅ `transire run` with local shims
- ✅ `transire build` with ARM64 Lambda packaging
- ✅ `transire deploy` with CDK
- ✅ `transire dev` commands for local testing

**IaC:**
- ✅ TypeScript CDK auto-generation
- ✅ User customization via YAML config
- ✅ Extension hooks for custom resources
- ✅ VPC and existing infrastructure integration

**Configuration:**
- ✅ YAML-based configuration
- ✅ Environment variable management
- ✅ Function grouping configuration
- ✅ Queue and schedule configuration

### Designed for Future Implementation

**Additional Languages:**
- 🔮 Rust language adapter
- 🔮 Python language adapter (future)
- 🔮 TypeScript language adapter (future)

**Additional Cloud Providers:**
- 🔮 Google Cloud Platform (Cloud Run, Pub/Sub, Cloud Functions)
- 🔮 Microsoft Azure (Functions, Service Bus, Logic Apps)

**Additional Runtimes:**
- 🔮 AWS ECS/Fargate
- 🔮 Google Cloud Run
- 🔮 Azure Container Instances
- 🔮 Kubernetes (provider-agnostic)

**Additional IaC:**
- 🔮 OpenTofu/Terraform support
- 🔮 Pulumi support
- 🔮 AWS SAM support

**NoSQL Abstractions:**
- 🔮 Table handler interface
- 🔮 Change stream handlers
- 🔮 DynamoDB provider implementation
- 🔮 Firestore provider implementation

**Additional CI/CD:**
- 🔮 GitLab CI support
- 🔮 Argo Workflows support
- 🔮 Azure DevOps support

The design ensures that these future features can be added through:
1. **Interface implementations** without changing core abstractions
2. **Plugin architecture** for language adapters and providers
3. **Configuration extensions** for new options
4. **Backward compatibility** for existing applications

## Summary

This design provides a comprehensive foundation for the Transire project that:

1. **Achieves the core goals** of cloud-agnostic, runtime-consistent applications
2. **Follows design principles** of leveraging existing ecosystems and minimizing boilerplate
3. **Maintains clean architecture** with clear separation of concerns and no circular dependencies
4. **Provides excellent DX** with familiar patterns and zero-config defaults
5. **Enables extensibility** for future languages, providers, and runtimes

The MVP scope is clearly defined while the architecture supports natural evolution to a comprehensive multi-language, multi-cloud platform.