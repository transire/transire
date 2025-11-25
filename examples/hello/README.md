# Transire app

This project was bootstrapped by "transire init". It shows HTTP, queue, and schedule handlers all producing queue messages.

## Local quickstart

- Run: `transire run --port 8080` (auto-restarts on code changes)
- HTTP: `curl "http://localhost:8080/?msg=hi"`
- Send to queue: `transire send work "manual message"` (defaults to env=local)
- Trigger schedule: `transire trigger heartbeat` (defaults to env=local)

## AWS (profile: transire-sandbox)

- Deploy: `transire deploy --profile transire-sandbox --env dev`
- Find endpoints/queues: `transire info --env dev --profile transire-sandbox --region us-east-1`
- HTTP: `curl "https://<api-endpoint>/?msg=hi"`
- Queue: `transire send work "manual message" --env dev --profile transire-sandbox --region us-east-1`
- Schedule: `transire trigger heartbeat --env dev --profile transire-sandbox --region us-east-1`
