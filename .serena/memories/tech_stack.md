# Transire Tech Stack

## Core Technologies
- **Language**: Go 1.21+
- **Web Framework**: Chi v5 (github.com/go-chi/chi/v5)
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **AWS Integration**: AWS Lambda Go SDK (github.com/aws/aws-lambda-go)

## Cloud Provider Integration
- **Primary Target**: AWS Lambda
- **API Gateway**: AWS API Gateway v2
- **Messaging**: AWS SQS
- **Scheduling**: AWS EventBridge

## Infrastructure as Code
- **Primary**: AWS CDK (TypeScript)
- **Future**: OpenTofu support planned

## Development Tools
- **Configuration**: YAML-based (transire.yaml)
- **Examples**: Go modules with dependencies

## Architecture Components
- HTTP Handlers (standard http.Handler with Chi routing)
- Queue Handlers (batch-based message processing)
- Scheduler Handlers (cron/scheduled tasks)
- Provider Plugins (cloud-specific runtime/deployment)

## Repository Structure
- `/examples/` - Sample applications and configurations
- `/pkg/transire/` - Core interface definitions
- `/cmd/transire/` - CLI tool design
- Design documents in markdown format