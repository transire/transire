# All-handlers demo

This app exercises every Transire handler type. HTTP requests enqueue work, the schedule handler enqueues a heartbeat every minute, queue handlers fan messages out to `notifications` and then to `notification-log`.

## Local development

- Run the app: `transire run --port 8080` (or `go run ./cmd/app`).
- Hit it: `curl "http://localhost:8080/?msg=hello-local"`.
- Watch the log output for chained queue deliveries.
- Tests: `go test ./...`.

## AWS

- Deploy: `transire deploy --profile transire-sandbox --env dev`.
- Fetch endpoints/queues: `transire info --env dev --profile transire-sandbox --region eu-west-2`.
- Exercise HTTP: `curl "https://<api-endpoint>/?msg=hello-aws"`.
- Exercise queues: `transire send work-events '{"source":"cli","detail":"aws send"}' --profile transire-sandbox --env dev --region eu-west-2`.
- Trigger schedule on-demand: `transire trigger heartbeat --profile transire-sandbox --env dev --region eu-west-2`.
