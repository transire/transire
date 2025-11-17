# Transire Design

**Transire** is an open-source toolchain and runtime abstraction that enables developers to write cloud-agnostic Go applications that run consistently across local development environments and cloud platforms.

## Project Status

🚧 **This repository contains the complete design specification for Transire.**

The actual implementation is not yet available. This design serves as the foundation for building the Transire project.

## Quick Overview

Transire lets you:

- Write standard Go applications using Chi routing
- Run the same code locally and in the cloud (starting with AWS Lambda)
- Deploy with zero infrastructure configuration
- Scale from simple APIs to complex multi-function architectures

### Example Application

```go
func main() {
    app := transire.New()
    r := app.Router()

    // Standard Chi routes - no framework lock-in
    r.Get("/health", healthHandler)
    r.Post("/users", createUserHandler)

    // Register background handlers
    app.RegisterQueueHandler(&EmailHandler{})
    app.RegisterScheduleHandler(&CleanupHandler{})

    // Works locally and on Lambda
    app.Run(context.Background())
}
```

### CLI Experience

```bash
# Create new project
transire init my-app

# Run locally with simulated cloud services
transire run

# Build and deploy to AWS
transire build
transire deploy
```

## Key Design Principles

1. **Stand on the shoulders of giants** - Use Chi, Cobra, CDK, and other proven libraries
2. **Zero boilerplate by default** - Familiar patterns without framework-specific abstractions
3. **Clear separation of concerns** - Cloud-specific code never leaks into application logic
4. **Runtime consistency** - Same behavior locally and in production
5. **Extensible architecture** - Support for future languages, clouds, and runtimes

## Architecture Highlights

### Core Abstractions

- **HTTP Handlers**: Standard `http.Handler` interface with Chi routing
- **Queue Handlers**: Batch-based message processing with failure tracking
- **Scheduler Handlers**: Cron/scheduled task execution
- **Provider Plugins**: Cloud-specific runtime and deployment implementations

### MVP Scope

- ✅ **Language**: Go
- ✅ **Cloud**: AWS (Lambda, API Gateway v2, SQS, EventBridge)
- ✅ **IaC**: AWS CDK (TypeScript)
- ✅ **CI**: GitHub Workflows
- 🔮 **Future**: Rust, GCP, Azure, OpenTofu, etc.

### Function Packaging

**Default**: Single Lambda function handles all HTTP, queue, and scheduled events with intelligent routing.

**Advanced**: Configure multiple functions for different concerns:

```yaml
functions:
  web:
    include:
      - http_handlers: "*"
    memory_mb: 512

  background:
    include:
      - queue_handlers: "*"
      - schedule_handlers: "*"
    memory_mb: 1024
    timeout_seconds: 300
```

## Repository Structure

```
├── DESIGN.md                     # Complete design specification
├── examples/
│   ├── simple-api/              # Basic HTTP API example
│   │   ├── main.go             # Chi routes + handlers
│   │   ├── handlers.go         # Queue/schedule handlers
│   │   └── transire.yaml       # Configuration
│   └── full-app/               # Advanced multi-function example
├── pkg/transire/               # Public API design
│   ├── interfaces.go          # Core interfaces
│   └── config.go             # Configuration system
├── cmd/transire/              # CLI design
│   └── main.go               # Cobra-based CLI
└── infrastructure/            # CDK extension examples
    └── extensions/           # User customization examples
```

## Design Documents

- **[Complete Design Specification](DESIGN.md)** - Comprehensive architecture and implementation plan
- **[Example Applications](examples/)** - Sample code demonstrating developer experience
- **[Package Structure](pkg/)** - Core interface definitions
- **[CLI Design](cmd/)** - Command-line tool specification

## Getting Started with the Design

1. **Read the [Design Document](DESIGN.md)** - Complete architecture and rationale
2. **Explore [Examples](examples/)** - See the intended developer experience
3. **Review [Interfaces](pkg/transire/interfaces.go)** - Core abstractions and types
4. **Check [Configuration](examples/simple-api/transire.yaml)** - YAML-based project configuration

## Implementation Roadmap

This design provides the foundation for implementing Transire in phases:

### Phase 1: Core MVP
- [ ] Go SDK with Chi integration
- [ ] AWS Lambda runtime support
- [ ] Local development shims
- [ ] Basic CLI (init, run, build, deploy)

### Phase 2: Production Features
- [ ] CDK auto-generation
- [ ] Multi-function packaging
- [ ] Queue/scheduler handlers
- [ ] AWS provider implementation

### Phase 3: Extensibility
- [ ] Provider plugin system
- [ ] Rust language support
- [ ] GCP/Azure providers
- [ ] OpenTofu IaC support

## Contributing to the Design

This design is a living document. Contributions, feedback, and improvements are welcome:

1. Review the design for completeness and clarity
2. Suggest improvements to interfaces and abstractions
3. Propose additional examples or use cases
4. Identify potential implementation challenges

## Why Transire?

**Transire** (Latin: "to go across, pass through") reflects the project's goal of seamlessly transitioning applications across different runtime environments while maintaining consistent behavior and developer experience.

## License

This design is released under [MIT License](LICENSE) to encourage open-source implementation and community contributions.

---

**Note**: This repository contains design specifications only. Implementation will be tracked in separate repositories as development begins.