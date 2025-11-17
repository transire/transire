/go You are an expert in:

- Go and Rust
- Serverless and container runtimes (AWS Lambda, ECS, EC2, etc.)
- AWS (API Gateway v2, Lambda, SQS, EventBridge, DynamoDB), and familiar with GCP/Azure
- Infrastructure as Code (CDK, OpenTofu; DO NOT use Terraform)
- Developer tooling and CLIs (Cobra, etc.)

Your task is to DESIGN (not implement fully) an open-source project called **Transire** that satisfies the following requirements.

---

## High-level Goal

Design **Transire**, an open-source toolchain and runtime abstraction that:

- Lets developers write **the same Go or Rust application code** and:
  - Run it locally.
  - Build and deploy it to the cloud (starting with AWS).
- Ensures **consistent behavior across runtimes**:
  - The same code should execute logically the same way locally and on cloud runtimes.

Transire should “stand on the shoulders of giants”: reuse existing libraries and ecosystems wherever possible instead of reinventing the wheel.

---

## Core Abstraction

Transire should provide **generic, cloud-agnostic primitives**:

- **HTTP handlers**
- **Queue handlers**
- **Scheduler/cron handlers**

These should:

- Be usable **locally** (via local shims: HTTP server, queue simulator, scheduler simulator).
- Map to cloud services when deployed, for example on AWS:
  - HTTP → API Gateway v2 + Lambda
  - Queue → SQS + Lambda
  - Scheduler → EventBridge + Lambda

Design the abstractions and packages so that:

- Application code is **agnostic** of:
  - Cloud provider (AWS, GCP, Azure, etc.)
  - Runtime (Lambda, ECS, EC2, etc.)
  - IaC (CDK, OpenTofu, etc.)
  - CI (GitHub, GitLab, Argo, etc.)
- Provider/runtime-specific details never leak into application code.

---

## MVP Scope (Very Important)

The **MVP** should support only:

- **Language**: Go
- **Cloud**: AWS
- **Runtime**: AWS Lambda (ARM architecture only)
- **IaC**: AWS CDK
- **CI**: GitHub Workflows

All other options (Rust, GCP, Azure, ECS, EC2, OpenTofu, GitLab, Argo, etc.) are **out of scope for implementation**, but the **design MUST be clearly extensible** to support them later.

---

## Design Principles & Constraints

1. **Stand on the shoulders of giants**
   - For Go:
     - Use **Chi** for HTTP routing.
     - Use **Cobra** for the Transire CLI.
   - Prefer widely adopted, idiomatic libraries in each ecosystem.
   - **Do not** reinvent routing, CLI parsing, configuration, etc., unless absolutely necessary.

2. **Zero boilerplate by default**
   - Go users should register **Chi routes exactly as they normally would**.
   - Transire should automatically wire routes/handlers into the appropriate HTTP/Queue/Scheduler abstractions.
   - Minimize framework “magic” while still reducing glue code.

3. **Clear separation of concerns**
   - Implementation details of a given provider **must not** leak into other packages.
   - For example, AWS-specific code must live in `cloud/aws` or `runtime/aws_lambda` packages only.
   - Avoid circular dependencies at all costs.
   - No runtime reflection in Go.

4. **No local shims in deployables**
   - Local-only shims (HTTP server, queue & scheduler emulators) must **never** be included in the artifacts deployed to the cloud (e.g., Lambda packages).

5. **Runtime discovery**
   - At runtime, Transire should use **environment variables** and/or configuration to decide:
     - Am I running locally or on a cloud runtime?
     - What provider/runtime is active?
   - This should be transparent to the user’s application code.

6. **Idiomatic Go**
   - All Go code and APIs must feel idiomatic and natural to Go developers:
     - No heavy global singletons.
     - No unusual patterns.
     - Prefer composition and interfaces.

7. **Lambda constraints**
   - For AWS:
     - Lambda functions must always be built for **ARM** architecture.
   - All handlers are packaged **by default** into a single Lambda function with a single entry point.
   - Transire is responsible for delegating/routing inside the function to the correct handler.

---

## Lambda Packaging & Routing

Design a flexible model for handler-to-function mapping:

- **Default behavior**:
  - All HTTP, Queue, and Scheduler handlers are bundled into a **single Lambda function**.
  - There is a single entry point; Transire inspects the event (API Gateway request, SQS event, EventBridge event) and routes it to the correct handler.

- **Advanced configurations that must be supported**:
  - User can split handlers into multiple functions, for example:
    - All HTTP handlers in one function.
    - All asynchronous handlers (queues & schedules) in another.
    - Certain critical handlers in their own dedicated function.
  - Design the configuration & abstractions so Transire can:
    - Group/assign handlers to specific Lambda functions.
    - Generate IaC accordingly (CDK constructs for multiple functions, routes, triggers, etc.).

Explain and design:

- How handler registration and grouping works.
- How Transire maps groups to IDE-friendly & CDK-friendly constructs.

---

## Queue Handler Semantics

For queue handlers:

- A handler should be able to receive **one or more messages** in a batch.
- The handler should return a list (or similar structure) of **failed messages** that should be retried.
- On AWS, use **SQS + Lambda partial batch failure support**:
  - Integrate with the standard SQS partial failure behavior.
  - Map returned failures to SQS’s partial batch failure response.

Explain:

- The handler interface in Go.
- How local queue simulation matches these semantics.
- How this will generalize to other clouds in the future.

---

## Transire CLI Design

Transire has a language-agnostic CLI (written in Go using Cobra) that can manage any supported app (Go now, Rust later).

Commands:

1. `transire init`
   - Creates a new app with sane defaults.
   - Options (with defaults):
     - Language: default `go` (but design as extensible to `rust`).
     - Cloud: default `aws`.
     - Runtime: default `lambda`.
     - IaC: default `cdk`.
     - CI: default `github-workflows`.
   - Should scaffold:
     - Minimal, idiomatic Go app.
     - Transire config files.
     - CDK app (TypeScript or Go – pick one and justify).
     - GitHub Workflow(s) for CI/CD.
   - Ensure the scaffolding is as **minimal and clean** as possible.

2. `transire run`
   - Runs the app locally:
     - Local HTTP server for HTTP handlers.
     - Local queue shim for queue handlers.
     - Local scheduler shim for scheduled handlers.
   - Should honor the same handler registrations as in the deployed environment.

3. `transire build`
   - Builds deployable artifacts:
     - Lambda packages (ARM).
     - Generated/updated IaC (CDK stacks) based on registered handlers and configuration.
   - Ensure local-only code (shims, dev tooling) is **not** included in the deployables.

4. `transire deploy`
   - Applies the IAC to the cloud:
     - For MVP: deploy via CDK to AWS.
   - This is the command intended to be invoked by CI (e.g., GitHub Workflow).

5. `transire dev ...`
   - A group of subcommands for local development utilities.
   - Required subcommands for MVP:
     - `transire dev queues list`
       - Lists all registered queues.
     - `transire dev queues send <queue> <message>`
       - Sends a message to a queue in the local environment for testing.
     - `transire dev schedules list`
       - Lists all registered schedules.
     - `transire dev schedules execute <schedule>`
       - Triggers a scheduled handler locally for testing.
   - Design this in a way that can be extended with more dev tools later.

Describe:

- The CLI UX.
- The internal architecture of the CLI (packages, commands, how it discovers app metadata, etc.).
- How the CLI interacts with the app’s configuration and registry of handlers.

---

## Infrastructure as Code (IaC) Requirements

Transire manages IaC by default, but must allow user customization.

For MVP (AWS + CDK):

1. **Default behavior**
   - Transire auto-generates and manages CDK stacks that:
     - Define Lambda functions (with ARM arch).
     - Wire API Gateway v2 routes to HTTP handlers.
     - Wire SQS queues to queue handlers.
     - Wire EventBridge rules to scheduler handlers.
   - All of this should be generated from:
     - Handler registrations.
     - A small amount of Transire configuration.

2. **User customization use-cases**
   The design must support:
   - Extending the **environment variables** configured on a Lambda.
   - Deploying Lambda functions into an **existing VPC/Subnets**.
   - Referencing **existing infrastructure** like:
     - DynamoDB tables
     - S3 buckets
   - And granting appropriate IAM permissions to the Lambda functions.

Show how this will work:

- Conceptually in configuration (YAML/TOML/Go code – pick one and justify).
- Via extension points in the generated CDK app.
- While still letting Transire own the majority of the generated infrastructure.

Explicit constraint: **never use Terraform**. OpenTofu support is a future extension but must be considered in the design.

---

## Extensibility & Future Features

The initial design must make it straightforward to later add:

1. **Additional languages**:
   - Rust (first candidate).
   - Design a language-agnostic plugin/adapter model for runtimes, handler registration, and build steps.

2. **Additional clouds and runtimes**:
   - GCP, Azure.
   - Runtimes like ECS, EC2, Cloud Run, etc.
   - Ensure provider-specific details remain isolated in their own packages/modules.

3. **NoSQL tables & change streams**:
   - Future primitives for:
     - NoSQL tables (e.g., DynamoDB on AWS; Firebase/Firestore on GCP).
     - Change-stream / event triggers (e.g., DynamoDB Streams, Firebase Triggers).
   - Design a generic abstraction for:
     - Table-like resources.
     - Change-notification streams and handlers.

Describe how the core abstractions, interfaces, and package layout will accommodate these later without breaking existing apps.

---

## Architecture & Output Expectations

Please produce a detailed **design document** that covers:

1. **High-level architecture**
   - A written description of major components:
     - Core Transire runtime/SDK.
     - Provider/runtime plugins.
     - CLI.
     - IaC generator.
     - Local shims.
   - How they interact and depend on each other (prioritize no circular dependencies).

2. **Package/module layout (for Go)**
   - Proposed directory/package structure (e.g., `internal/`, `pkg/`).
   - Separation between:
     - Core abstractions.
     - AWS provider.
     - Lambda runtime support.
     - CDK integration.
     - CLI.
     - Local dev tooling.

3. **Core interfaces & types (in Go)**
   - Define the interfaces for:
     - HTTP handlers.
     - Queue handlers.
     - Scheduler handlers.
     - Provider/runtime plugins.
     - IaC generation hooks.
   - Show example Go interfaces and minimal implementations.

4. **Example developer experience (DX)**
   - Example Go application code:
     - HTTP routes using Chi.
     - Queue handler.
     - Scheduler handler.
   - Show how a developer:
     - Registers handlers.
     - Runs locally (`transire run`).
     - Builds & deploys (`transire build` + `transire deploy`).
   - Include snippets that illustrate “zero boilerplate by default”.

5. **CLI behavior & configuration**
   - Example `transire init` output (project layout).
   - Example config file(s) for:
     - Selecting cloud/runtime/IaC/CI.
     - Grouping handlers into functions.
     - Customizing environment variables, VPC, and existing resources.

6. **MVP boundaries**
   - Clearly distinguish what is:
     - Implemented in the MVP (Go + AWS Lambda + CDK + GitHub Workflows).
     - Only designed/extensible for later.

You should focus on clarity, extensibility, and idiomatic Go design. Avoid reflection and hidden magic. Favor explicit, composable interfaces and clean boundaries between core, provider, runtime, IaC, and CLI layers.

Persist all generated content to disk for further review. 
