# Handler chaining demo

Bootstrapped with `transire init`, this app shows HTTP, queue, and schedule handlers that all enqueue new queue messages. The flow is `HTTP/queue/schedule -> work -> summary-log -> log-stream`.

## Local quickstart

- Run: `transire run --port 8080`
- HTTP: `curl "http://localhost:8080/?msg=hi"` (enqueues `work`)
- Send to queue: `transire send work "manual message"` (defaults to env=local)
- Trigger schedule: `transire trigger heartbeat` (defaults to env=local)
- Watch the log output as queue handlers fan out into `summary-log` then `log-stream`.
- Tests: `go test ./...`

## AWS (profile: transire-sandbox)

- Deploy: `transire deploy --profile transire-sandbox --env dev`
- Find endpoints/queues: `transire info --env dev --profile transire-sandbox --region eu-west-2`
- HTTP: `curl "https://<api-endpoint>/?msg=hi"`
- Queue: `transire send work "manual message" --env dev --profile transire-sandbox --region eu-west-2`
- Schedule: `transire trigger heartbeat --env dev --profile transire-sandbox --region eu-west-2`
- Tail: use `aws logs tail /aws/lambda/handler-chaining-dev --profile transire-sandbox --since 5m --follow --region eu-west-2` to watch queue fan-out logs.
