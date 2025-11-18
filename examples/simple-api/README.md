# Simple API Example

This example demonstrates a basic REST API built with Transire, featuring HTTP handlers, queue handlers, and scheduled tasks.

## Features

- **HTTP Endpoints**: Basic REST API with health check
- **Queue Handlers**: Email processing simulation
- **Scheduled Tasks**: Daily cleanup simulation
- **Local Development**: Hot reload and queue simulation

## Running Locally

From this directory:

```bash
# Install dependencies
go mod tidy

# Run with hot reload
../../transire-cli run

# Or build and run directly
go build -o simple-api .
./simple-api
```

The API will be available at:
- HTTP: `http://localhost:3000`
- Queue simulation: `http://localhost:4000`

## API Endpoints

- `GET /health` - Health check
- `GET /api/users` - List users (placeholder)
- `POST /api/users` - Create user (placeholder)

## Queue Testing

```bash
# Send test message to email queue
curl -X POST http://localhost:4000/queues/email-queue \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "subject": "Test"}'
```

## Schedule Testing

```bash
# Trigger daily cleanup manually
curl -X POST http://localhost:4000/schedules/daily-cleanup
```

## Deployment

```bash
# Build for Lambda
../../transire-cli build

# Deploy to AWS
../../transire-cli deploy
```

## License

**Example Code**: MIT License - Feel free to copy, modify, and use this example code in your own projects.

**Transire Framework**: Mozilla Public License 2.0 - The underlying Transire framework is licensed under MPL 2.0.