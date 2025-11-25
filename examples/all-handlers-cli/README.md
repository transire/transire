# CLI-driven all-handlers demo

This app shows every Transire handler type working together. HTTP requests enqueue work, queue handlers fan messages to `notifications` and `notification-log`, and the `heartbeat` schedule also enqueues work.

## Local quickstart

- Start the app: `transire run --port 8080` (auto-restarts on code changes)
- Hit HTTP: `curl "http://localhost:8080/?msg=hello-local"`
- Send straight to the queue: `transire send work-events '{"source":"cli","detail":"local send"}'` (defaults to env=local)
- Trigger the schedule on demand: `transire trigger heartbeat` (defaults to env=local)

## AWS (profile: transire-sandbox)

- Deploy: `transire deploy --profile transire-sandbox --env dev`
- Find endpoints/queues: `transire info --env dev --profile transire-sandbox --region us-east-1`
- Exercise HTTP: `curl "https://<api-endpoint>/?msg=hello-aws"`
- Send a queue message: `transire send work-events '{"source":"cli","detail":"aws send"}' --env dev --profile transire-sandbox --region us-east-1`
- Trigger the schedule: `transire trigger heartbeat --env dev --profile transire-sandbox --region us-east-1`
