# Transire TODO App Example

A comprehensive example application demonstrating all Transire features: HTTP handlers, Queue handlers, and Scheduler handlers.

## Features

### HTTP API (Chi Router)
- **GET** `/` - Home endpoint
- **GET** `/health` - Health check
- **POST** `/api/v1/todos` - Create a new TODO
- **GET** `/api/v1/todos` - List all TODOs
- **GET** `/api/v1/todos/{id}` - Get a specific TODO
- **PUT** `/api/v1/todos/{id}` - Update a TODO
- **DELETE** `/api/v1/todos/{id}` - Delete a TODO
- **POST** `/api/v1/todos/{id}/complete` - Mark TODO as complete

### Queue Handlers
1. **todo-reminders** - Sends reminder emails for TODOs approaching their due date
   - Batch size: 10
   - Visibility timeout: 30s
   - Max retries: 3

2. **todo-notifications** - Sends notifications for TODO events (created, updated, completed)
   - Batch size: 20
   - Visibility timeout: 60s
   - Max retries: 5

### Schedule Handlers
1. **cleanup-completed-todos** - Removes completed TODOs older than 30 days
   - Schedule: `0 3 * * *` (Daily at 3 AM UTC)
   - Timeout: 300s
   - Retries: 3

2. **daily-todo-summary** - Generates daily summary of pending/completed TODOs
   - Schedule: `0 9 * * *` (Daily at 9 AM UTC)
   - Timeout: 120s
   - Retries: 2

## Quick Start

### 1. Build the Application
```bash
go mod tidy
go build -o todo-app
```

### 2. Run Locally
```bash
./todo-app
```

The application will start on `http://localhost:3000`

### 3. Test the API
```bash
# Health check
curl http://localhost:3000/health

# Create a TODO
curl -X POST http://localhost:3000/api/v1/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy groceries","description":"Milk, eggs, bread","due_date":"2025-11-20T12:00:00Z"}'

# List TODOs
curl http://localhost:3000/api/v1/todos

# Get a specific TODO
curl http://localhost:3000/api/v1/todos/{id}

# Update a TODO
curl -X PUT http://localhost:3000/api/v1/todos/{id} \
  -H "Content-Type: application/json" \
  -d '{"description":"Milk, eggs, bread, butter"}'

# Complete a TODO
curl -X POST http://localhost:3000/api/v1/todos/{id}/complete

# Delete a TODO
curl -X DELETE http://localhost:3000/api/v1/todos/{id}
```

### 4. Test Queue Handlers

First, create a TODO to get its ID, then:

```bash
# Send a reminder (note: -c transire.yaml is optional when config is in current directory)
../../transire-cli dev queues send todo-reminders \
  '{"todo_id":"YOUR-TODO-ID","user_email":"user@example.com"}'

# Send a notification
../../transire-cli dev queues send todo-notifications \
  '{"todo_id":"YOUR-TODO-ID","user_email":"user@example.com","event_type":"created","todo_title":"Your TODO Title"}'

# List all queues
../../transire-cli dev queues list
```

### 5. Test Schedule Handlers

```bash
# Execute cleanup schedule manually
../../transire-cli dev schedules execute cleanup-completed-todos

# Execute daily summary schedule manually
../../transire-cli dev schedules execute daily-todo-summary

# List all schedules
../../transire-cli dev schedules list
```

## Running E2E Tests

```bash
# Run all E2E tests
go test -v -run TestE2E

# Run specific test suite
go test -v -run TestE2E_HTTPHandlers
go test -v -run TestE2E_QueueHandlers
go test -v -run TestE2E_ScheduleHandlers
go test -v -run TestE2E_IntegrationWorkflow
```

## Architecture

### In-Memory Storage
The app uses a simple in-memory store with a `sync.RWMutex` for thread-safe operations. In a production system, this would be replaced with a real database.

### Handler Registration
All handlers are registered with the Transire app in `main.go`:
```go
app.RegisterQueueHandler(&TodoReminderQueue{store: store})
app.RegisterQueueHandler(&TodoNotificationQueue{store: store})
app.RegisterScheduleHandler(&CleanupCompletedTodosSchedule{store: store})
app.RegisterScheduleHandler(&DailyTodoSummarySchedule{store: store})
```

### Message Types
- `TodoReminderMessage` - Reminder queue messages
- `TodoNotificationMessage` - Notification queue messages
- `TodoSummary` - Daily summary data structure

## Configuration

See `transire.yaml` for full configuration including:
- Lambda settings (memory, timeout, architecture)
- Queue configurations (batch sizes, timeouts)
- Schedule configurations (timezone, enabled status)
- Development settings (ports, log levels)

## Development Notes

### Mock Implementations
The app includes mock implementations for:
- `sendTodoReminder()` - Simulates sending reminder emails
- `sendTodoNotification()` - Simulates sending notifications

These include simulated failures for testing:
- Reminders fail for `fail-reminder@example.com`
- Notifications fail for `fail-notification@example.com`

### Failure Handling
Queue handlers properly track failed message IDs and return them for retry:
```go
var failedIDs []string
for _, msg := range messages {
    if err := processMessage(msg); err != nil {
        failedIDs = append(failedIDs, msg.ID())
    }
}
return failedIDs, nil
```

## Production Deployment

To deploy to AWS Lambda:
```bash
# Build for deployment
transire build

# Deploy to AWS
transire deploy
```

The same code runs locally and in Lambda with no changes!

## License

Part of the Transire project.
