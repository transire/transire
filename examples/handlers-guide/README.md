# Handlers Guide

This app mirrors the docs: HTTP, queue, and schedule handlers all send queue messages so you can see the full flow end to end.

## Local quickstart

- Run: `transire run --port 8080` (set `TRANSIRE_PORT` if you change the port so `transire send|trigger` target the right host).
- HTTP: `curl "http://localhost:8080/?msg=hi"`
- Queue: `transire send work "manual message"`
- Schedule: `transire trigger heartbeat`

## AWS (profile: transire-sandbox)

- Deploy: `AWS_REGION=us-east-1 AWS_DEFAULT_REGION=us-east-1 transire deploy --profile transire-sandbox --env dev`
- Discover endpoints: `transire info --env dev --profile transire-sandbox` (add `AWS_REGION`/`AWS_DEFAULT_REGION` here too if your profile has no default region)
- HTTP: `curl "https://<api-endpoint>/?msg=hi"`
- Queue: `transire send work "manual message" --env dev --profile transire-sandbox` (prefix with `AWS_REGION`/`AWS_DEFAULT_REGION` if needed)
- Schedule: `transire trigger heartbeat --env dev --profile transire-sandbox` (same region note)
