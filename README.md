Transire is a cloud-agnostic Go webapp framework. User code stays portable while dispatchers handle cloud specifics (local and AWS to start). HTTP handlers use chi; queue and schedule handlers share a cloud-agnostic context that can enqueue messages. Handlers are discovered from user code at build time so no handler definitions live in config.

## Install via Go

- CLI (global): `go install github.com/transire/transire/cmd/transire@latest` (binary lands in `$GOBIN` or `$(go env GOPATH)/bin`).
- Framework: `go get github.com/transire/transire@latest` inside your module to import Transire packages in existing projects.

## What you get

- CLI that scaffolds, runs locally, and inspects discovered handlers.
- Local dispatcher plus AWS dispatcher (API Gateway v2 HTTP API, SQS, EventBridge schedules) behind a shared Lambda.
- Build emits a Lambda bootstrap and CDK app; deploy drives CDK.

## Quickstart (minutes)

- Install CLI: `go install github.com/transire/transire/cmd/transire@latest`
- Scaffold: `transire init my-app && cd my-app`
- Run locally: `transire run --port 8080` (auto-restarts on code changes)
  - HTTP: `curl "http://localhost:8080/?msg=hi"`
  - Queue: `transire send work "manual message"` (defaults to env=local)
  - Schedule: `transire trigger heartbeat` (defaults to env=local)
- Deploy to AWS: `transire deploy --profile <aws-profile> --env dev`
  - Discover endpoints/queues: `transire info --env dev --profile <aws-profile> --region <region>`
  - Hit it: `curl "https://<api-endpoint>/?msg=hi"` and `transire send work "manual message" --env dev --profile <aws-profile> --region <region>`

See `examples/hello` for the scaffold output, `examples/all-handlers` for a fuller sample, and `examples/handler-chaining` for a queue-chaining demo where HTTP, queue, and schedule handlers all enqueue downstream work.

Dispatcher selection is automatic: Transire uses the AWS dispatcher when Lambda env vars are present, otherwise the local dispatcher. Set `TRANSIRE_DISPATCHER=aws|local` to override when needed.

## Build and deploy to AWS

```bash
transire build
cd dist/aws/cdk && npm install
transire deploy --profile <aws-profile> --env <env>
```

Requires AWS credentials and a bootstrapped CDK environment. CDK resources are named `${app}-${logical}-${env}` (queues, schedules, API, lambda). Transire maps logical queue names used in code to fully-qualified queue names via environment variables. Build artifacts are environment-agnostic; only deploy is env-specific.

Transire expects your main package at `./cmd/app`. If you started with an older layout, move your entrypoint there before running `transire build` or `transire deploy`.

## Developing this repository

- Run all validation locally with `./scripts/ci.sh` (gofmt check, go vet, go test for the main module and examples, and `go build ./cmd/transire`). The GitHub Actions workflow executes the same script on every pull request.
- Optional: enable the provided hook with `git config core.hooksPath .githooks` so commits run `./scripts/ci.sh` automatically.

## Licensing

- Core framework: MPL-2.0 (see `LICENSE`).
- Examples and generated starter apps: MIT (see per-example `LICENSE` files).
