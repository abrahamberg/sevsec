# `.devsec` Go Learning Tutorial

> Goal: build `.devsec` as a learning project, not as a rushed production app.
>
> Product shape: one product, two Go binaries:
>
> - `devsec-server`: Gin web/API service, auth, access control, secrets, Postgres
> - `devsec`: CLI/TUI client that calls the server, fetches allowed runtime env vars, and runs local commands

This guide is intentionally written for a C# developer who already knows architecture and wants to learn the Go way: smaller packages, fewer abstractions, explicit wiring, simple interfaces, and boring code that is easy to read.

---

## 0. The mental shift from C# to Go

In C#, it is common to start with layers, interfaces, dependency injection containers, DTO mapping libraries, attributes, middleware frameworks, abstract base classes, and generic service patterns.

In Go, try the opposite:

- Start concrete.
- Add interfaces only when they help a caller.
- Keep packages small but not tiny.
- Prefer functions and structs over frameworks.
- Prefer explicit wiring over a DI container.
- Let the compiler and tests guide structure.
- Avoid architecture that exists only because “enterprise apps usually have it”.

Your goal is not to create a perfect framework. Your goal is to understand where Go wants simplicity.

The guiding sentence:

> Make the code boring enough that the domain is the interesting part.

---

## 1. Product definition

`.devsec` is a developer secret manager for local development.

A developer runs:

```bash
devsec run checkout-service -- npm run dev
```

The CLI:

1. Reads local config.
2. Authenticates using a stored token.
3. Calls `devsec-server`.
4. Requests runtime environment variables for a project.
5. Receives only the secrets the user is allowed to use.
6. Starts the command with those values injected as environment variables.
7. Forwards stdin/stdout/stderr.
8. Exits with the same exit code as the child process.

The server:

1. Owns projects, teams, groups, users, secrets, auth, access rules, audit logs.
2. Stores encrypted secret values in Postgres.
3. Validates login or Azure identity.
4. Maps users/groups to permissions.
5. Decides whether the CLI may receive secrets.
6. Returns runtime env vars only after authorization.

Important boundary:

> The CLI never talks to Postgres and never performs authorization. The server is the security boundary.

---

## 2. First architecture diagram

```text
.devsec product

┌────────────────────────────────────┐
│ devsec CLI                         │
│                                    │
│ - parse command                    │
│ - load local config                │
│ - call server API                  │
│ - inject env vars                  │
│ - run subprocess                   │
│ - later: Bubble Tea TUI            │
└──────────────────┬─────────────────┘
                   │ HTTPS + token
                   ▼
┌────────────────────────────────────┐
│ devsec-server                      │
│                                    │
│ - Gin API                          │
│ - web UI                           │
│ - auth                             │
│ - Azure token exchange later       │
│ - project/team/group model         │
│ - access control                   │
│ - secret encryption/decryption     │
│ - audit log                        │
└──────────────────┬─────────────────┘
                   │
                   ▼
┌────────────────────────────────────┐
│ Postgres                           │
│                                    │
│ - encrypted secrets                │
│ - users/groups/projects            │
│ - tokens/sessions                  │
│ - audit events                     │
└────────────────────────────────────┘
```

---

## 3. Repository structure

Start with one Go module and two binaries.

```text
devsec/
  go.mod
  go.sum
  README.md
  Makefile

  cmd/
    devsec/
      main.go
    devsec-server/
      main.go

  internal/
    contract/
      auth.go
      runtime_env.go
      project.go
      errors.go

    cli/
      command/
      config/
      client/
      runner/
      tui/

    server/
      http/
      middleware/
      auth/
      project/
      team/
      group/
      secret/
      access/
      audit/
      storage/

  migrations/
    000001_init.up.sql
    000001_init.down.sql

  docs/
    decisions/
      0001-two-binaries-one-product.md
```

Rules:

- `cmd/devsec/main.go` should be thin.
- `cmd/devsec-server/main.go` should be thin.
- `internal/cli` is only for CLI behavior.
- `internal/server` is only for server behavior.
- `internal/contract` contains shared API request/response structs only.
- Do not put business logic in `contract`.
- Avoid `pkg/` unless you are intentionally publishing reusable packages for other modules.

A good smell:

```text
cmd/*/main.go wires dependencies and calls Run().
```

A bad smell:

```text
cmd/devsec-server/main.go contains SQL queries, Gin handlers, token validation, and business rules.
```

---

## 4. Suggested packages to explore

Use popular packages, but learn why each exists.

### CLI

- `cobra` for command structure.
- `viper` or a small custom config loader for config.
- `bubbletea` for later interactive TUI.
- `lipgloss` for TUI styling.

Do not start with Bubble Tea. First make the non-interactive CLI correct.

### Server

- `gin` for HTTP routing.
- `pgx` or `pgxpool` for Postgres access.
- `sqlc` for type-safe SQL generation.
- `golang-migrate/migrate` for database migrations.
- `slog` from the standard library for structured logging.
- OpenTelemetry later for tracing.

### Auth/security

- `golang-jwt/jwt` or another maintained JWT library if you issue JWTs.
- `crypto/aes`, `crypto/cipher`, `crypto/rand` from the standard library for encryption experiments.
- Later: Azure/OIDC validation libraries.

### Testing

- standard `testing` package first.
- `httptest` for HTTP handlers.
- `testify` if you want assertions.
- Testcontainers later if you want real Postgres integration tests.

Learning rule:

> Add a package when you can explain the problem it solves in one sentence.

---

## 5. Domain model, first draft

Keep the domain boring.

```text
User
- ID
- Email
- DisplayName
- ExternalProvider optional
- ExternalSubject optional

Team
- ID
- Name

Group
- ID
- Name
- ExternalProvider optional, e.g. azure-ad
- ExternalID optional, e.g. Azure group object ID

Project
- ID
- Name
- Slug
- TeamID

Secret
- ID
- ProjectID
- Environment
- Key
- ValueEncrypted
- CreatedAt
- UpdatedAt

AccessRule
- ID
- ProjectID
- GroupID
- Permission, e.g. read_runtime_env

AuditEvent
- ID
- ActorUserID
- Action
- ProjectID optional
- CreatedAt
- Metadata JSON
```

Keep project names/slugs stable. The CLI should call the server with a project slug, not a database ID.

---

## 6. API contract first

Because the CLI and server are separate applications, the API contract is important.

Start with this endpoint:

```http
POST /api/runtime-env
Authorization: Bearer <token>
Content-Type: application/json
```

Request:

```json
{
  "project": "checkout-service",
  "environment": "local",
  "reason": "run-command"
}
```

Response:

```json
{
  "project": "checkout-service",
  "environment": "local",
  "env": {
    "DATABASE_URL": "postgres://...",
    "REDIS_URL": "redis://..."
  }
}
```

Why `runtime-env` instead of `secrets`?

Because the intent is clearer. The client does not need to know all secret-management details. It needs environment variables for one runtime session.

Later you can add:

```text
POST /api/auth/login
POST /api/auth/azure/exchange
GET  /api/me
GET  /api/projects
POST /api/projects
POST /api/projects/{project}/secrets
GET  /api/projects/{project}/secret-keys
POST /api/access-rules
```

---

## 7. Milestones / commits

Build this in small commits. Each commit should teach one Go idea.

---

### Commit 1: Two binaries, no real behavior

Goal:

```bash
go run ./cmd/devsec --help
go run ./cmd/devsec-server
```

Explore:

- Go modules.
- `cmd/` layout.
- `main` packages.
- thin entrypoints.

Think about:

- Why are there two binaries?
- What code is shared?
- What should never be shared?

Avoid:

- Creating all packages upfront with empty files.
- Building a framework before behavior exists.

---

### Commit 2: CLI command parsing

Goal:

```bash
devsec run checkout-service -- npm run dev
```

For now, only print:

```text
project: checkout-service
command: npm run dev
```

Explore:

- Cobra commands.
- How arguments after `--` work.
- Separating command parsing from command execution.

Good package boundary:

```text
internal/cli/command
```

Possible interface later:

```go
type Runner interface {
    Run(ctx context.Context, command []string, env map[string]string) error
}
```

But do not create it until you need to test command behavior without actually running `npm`.

---

### Commit 3: CLI process runner

Goal:

```bash
devsec run demo -- env
```

For now, inject hardcoded env vars:

```text
DEVSEC_EXAMPLE=hello
```

Explore:

- `os/exec`.
- stdin/stdout/stderr forwarding.
- environment variable merging.
- returning child process exit code.
- context cancellation.

Important learning:

> Running a subprocess is infrastructure logic. Keep it separate from CLI argument parsing.

Suggested package:

```text
internal/cli/runner
```

---

### Commit 4: Server health endpoint

Goal:

```bash
curl http://localhost:8080/healthz
```

Response:

```json
{"status":"ok"}
```

Explore:

- Gin router.
- handlers.
- middleware.
- JSON response shapes.
- graceful shutdown.

Suggested package:

```text
internal/server/http
```

Keep `main.go` tiny:

```text
load config -> create server -> run
```

---

### Commit 5: CLI calls server

Goal:

```bash
devsec server health
```

or:

```bash
devsec run demo -- env
```

but before running the command, it calls:

```text
GET /healthz
```

Explore:

- `net/http` client.
- timeouts.
- context propagation.
- error handling.
- local config for server URL.

Suggested package:

```text
internal/cli/client
```

Important design:

The CLI should depend on a small client interface, not on Gin, Postgres, or server internals.

---

### Commit 6: Shared contract package

Goal:

Create request/response structs used by both CLI and server.

Example conceptual files:

```text
internal/contract/runtime_env.go
internal/contract/errors.go
```

Explore:

- JSON struct tags.
- shared API types.
- avoiding shared business logic.

Rule:

> `contract` contains nouns crossing the wire, not services.

Good:

```text
RuntimeEnvRequest
RuntimeEnvResponse
ErrorResponse
```

Bad:

```text
AccessPolicy
SecretDecryptor
UserRepository
```

---

### Commit 7: Runtime env endpoint with fake data

Goal:

Server supports:

```text
POST /api/runtime-env
```

It returns fake env vars for now.

CLI calls it and injects the returned env vars.

Explore:

- HTTP POST.
- JSON encoding/decoding.
- handler validation.
- client/server contract.
- integration testing with `httptest`.

Important:

This is the first real vertical slice.

```text
CLI -> HTTP -> server handler -> response -> CLI runner
```

No database yet. No auth yet. No encryption yet.

---

### Commit 8: Postgres and migrations

Goal:

Start Postgres with Docker Compose and run migrations.

Tables:

```sql
projects
secrets
```

Explore:

- Postgres basics.
- migrations.
- connection pooling with `pgxpool`.
- config from environment variables.

Suggested package:

```text
internal/server/storage
```

Learning point:

> In Go, database access can be simple. Do not create generic repositories before you have repeated patterns.

---

### Commit 9: sqlc queries

Goal:

Use SQL files to generate type-safe Go methods.

Explore:

- schema files.
- query files.
- generated code.
- why Go likes explicit SQL.

Example query names:

```text
GetProjectBySlug
ListSecretsForProjectEnvironment
CreateProject
CreateSecret
```

Think about:

- SQL is part of your application design.
- You do not need an ORM immediately.
- Type-safe generated SQL is a nice Go middle ground.

---

### Commit 10: Secret service

Goal:

Move secret lookup out of the HTTP handler.

Conceptual flow:

```text
handler -> secret service -> sqlc queries -> Postgres
```

Explore:

- service structs.
- constructor functions.
- context passing.
- error wrapping.

The handler should not know SQL details.

The service should not know Gin details.

Good mental shape:

```text
HTTP is transport.
Service is application behavior.
Storage is infrastructure.
```

---

### Commit 11: Interfaces, but only at boundaries

Goal:

Introduce your first useful interface.

Good candidate:

```go
type RuntimeEnvProvider interface {
    GetRuntimeEnv(ctx context.Context, project string, environment string, user UserIdentity) (map[string]string, error)
}
```

Or better: define an interface in the handler package for exactly what the handler needs.

Explore:

- consumer-defined interfaces.
- small interfaces.
- testing handlers with fake services.

Rule:

> In Go, interfaces usually belong where they are consumed, not where they are implemented.

C# habit to avoid:

```text
IProjectRepository
ISecretRepository
IUserRepository
IAccessRuleRepository
```

created before there is a test or alternate implementation.

Go habit to practice:

```text
Use concrete types until an interface removes pain.
```

---

### Commit 12: Auth token, simple version

Goal:

CLI stores a token. Server requires it.

Flow:

```bash
devsec login
```

For now, login can be fake or local.

Explore:

- auth middleware.
- `Authorization: Bearer` header.
- local CLI config file.
- not logging tokens.

Suggested packages:

```text
internal/server/auth
internal/server/middleware
internal/cli/config
```

Important:

The server identifies the user. The CLI only presents a token.

---

### Commit 13: Access service

Goal:

Before returning env vars, check:

```text
Can user X read runtime env for project Y?
```

Explore:

- application policy.
- keeping authorization centralized.
- separating access decisions from secret storage.

Suggested package:

```text
internal/server/access
```

Bad design:

```text
secret repository checks groups directly
```

Better design:

```text
runtime env service asks access service before reading/decrypting secrets
```

---

### Commit 14: Encryption at rest

Goal:

Store encrypted values in Postgres.

Explore:

- envelope thinking.
- master key from environment variable.
- `crypto/rand`.
- authenticated encryption.
- never logging plaintext.

Suggested package:

```text
internal/server/secret/crypto.go
```

Keep it simple for learning:

```text
DEVSEC_MASTER_KEY from environment
encrypted secret values in database
```

Later you can move to Azure Key Vault, HashiCorp Vault, SOPS, or another secret backend.

---

### Commit 15: Audit logging

Goal:

Every runtime env request creates an audit event.

Capture:

```text
user
project
environment
action
result allowed/denied
created_at
```

Explore:

- security observability.
- what to log.
- what never to log.

Never log:

```text
secret values
tokens
passwords
full Authorization headers
```

---

### Commit 16: Web UI, boring version

Goal:

Add simple pages:

```text
/projects
/projects/{slug}
/groups
/access-rules
```

Explore:

- Gin HTML templates.
- server-side rendering.
- forms.
- CSRF later.

Do not make this a React project unless your goal changes. For learning Go, server-rendered HTML is enough.

---

### Commit 17: Bubble Tea TUI

Goal:

```bash
devsec tui
```

Shows:

```text
- server connection status
- current user
- project list
- selectable project
- run command profile later
```

Explore:

- Bubble Tea model/update/view pattern.
- async commands.
- terminal UI state.
- keeping TUI separate from API client.

Important:

Bubble Tea should use the same CLI client package as normal commands.

```text
TUI -> cli/client -> server API
run command -> cli/runner
```

---

### Commit 18: Azure token exchange

Goal:

CLI sends an Azure token to the server. Server validates it and returns a `.devsec` token.

Flow:

```text
Azure token -> devsec-server -> validate -> map user/groups -> issue devsec token
```

Explore:

- external identity as input.
- internal authorization as server-owned policy.
- group synchronization/mapping.

Important:

The CLI should not decide permissions from Azure groups.

The server decides.

---

## 8. Package design exercises

When adding code, ask:

### Is this transport logic?

Examples:

- Gin request parsing.
- HTTP status codes.
- JSON response shape.
- route parameters.

Put in:

```text
internal/server/http
```

### Is this application behavior?

Examples:

- get runtime env for user/project/environment.
- authenticate login.
- check access.
- record audit event.

Put in:

```text
internal/server/secret
internal/server/access
internal/server/auth
internal/server/audit
```

### Is this infrastructure?

Examples:

- Postgres query.
- encryption implementation.
- HTTP client.
- file config.
- process execution.

Put in:

```text
internal/server/storage
internal/cli/client
internal/cli/config
internal/cli/runner
```

---

## 9. Interface rules for this project

Use this checklist before creating an interface:

1. Is there more than one implementation today?
2. Do I need a fake for a test?
3. Does the caller only need a tiny subset of methods?
4. Is this a real boundary, e.g. HTTP client, runner, clock, token validator?

If no, use a concrete type.

Good interfaces:

```text
Clock
TokenValidator
RuntimeEnvClient
CommandRunner
AccessChecker
```

Suspicious interfaces:

```text
IProjectService
ISecretService
IUserManager
IRepositoryBase[T]
IUnitOfWorkFactoryProvider
```

The Go style is not “no interfaces”. It is “interfaces where they make the code smaller and easier to test”.

---

## 10. Error handling style

Go has no exceptions. Design your error flow.

Use errors for domain/application cases:

```text
ErrProjectNotFound
ErrAccessDenied
ErrInvalidToken
ErrSecretNotFound
```

The service returns meaningful errors.

The HTTP layer maps them:

```text
ErrProjectNotFound -> 404
ErrAccessDenied    -> 403
ErrInvalidToken    -> 401
unknown error      -> 500
```

The CLI maps API errors to readable terminal output.

Avoid:

- panics for normal failures.
- logging the same error at every layer.
- returning raw SQL errors directly to users.

Practice:

> Add context when returning errors, but handle them at boundaries.

---

## 11. Context usage

In Go services, `context.Context` should flow through I/O paths.

Use it for:

- HTTP request lifetime.
- database queries.
- server shutdown.
- CLI cancellation.
- subprocess cancellation.

Do not store context in structs.

Pass it as the first parameter:

```go
DoSomething(ctx context.Context, ...)
```

Good exercise:

- Start `devsec run demo -- sleep 60`.
- Press Ctrl+C.
- Ensure the child process is stopped cleanly.

---

## 12. Configuration

Server config:

```text
DEVSEC_HTTP_ADDR=:8080
DEVSEC_DATABASE_URL=postgres://...
DEVSEC_MASTER_KEY=...
DEVSEC_TOKEN_SIGNING_KEY=...
```

CLI config:

```text
~/.config/devsec/config.json
```

Example:

```json
{
  "server_url": "http://localhost:8080",
  "access_token": "..."
}
```

Learning point:

> Config is infrastructure. Keep it boring.

Do not create a giant global config singleton.

---

## 13. Security notes

Even as a learning project, build safe habits.

Never:

- log secret values.
- print tokens.
- store plaintext secrets in Postgres.
- let the CLI connect directly to Postgres.
- let the CLI decide access.
- return secrets from generic admin endpoints accidentally.

Remember:

- Child processes inherit environment variables.
- Environment variables can sometimes be inspected by local tools depending on OS/settings.
- Shell history can leak command-line arguments, so secrets should not be passed as CLI args.
- Audit every secret access.

Use a specific endpoint for runtime env access so you can audit intent.

---

## 14. Testing plan

Start small.

### CLI tests

Test:

- command parsing.
- config loading.
- API client request shape.
- runner env merging.

Use fake HTTP server where useful.

### Server tests

Test:

- handler returns correct status codes.
- access denied blocks secrets.
- project not found maps to 404.
- invalid token maps to 401.

### Service tests

Test:

- runtime env service asks access checker first.
- denied users do not trigger secret decryption.
- audit event is recorded for allowed and denied requests.

### Integration tests later

Use real Postgres only when the simpler tests already pass.

---

## 15. Anti-bloat rules for yourself

Because you know you may over-architect, use these constraints:

1. No interface until the second consumer or a test needs it.
2. No generic repository.
3. No service base classes. Go does not have inheritance; do not simulate it.
4. No global container.
5. No package named `common` unless you can explain exactly what belongs there.
6. No `utils` package for domain behavior.
7. No `pkg/` until another module imports it.
8. No event bus until there is an actual async workflow.
9. No background worker until one feature requires it.
10. No Azure integration until local auth and access rules work.

Your mantra:

> Build the vertical slice first. Generalize only after pain appears.

---

## 16. Suggested final shape after learning phase

```text
internal/
  contract/
    runtime_env.go
    auth.go
    errors.go

  cli/
    command/
      root.go
      run.go
      login.go
    client/
      client.go
      runtime_env.go
      auth.go
    config/
      file.go
    runner/
      runner.go
    tui/
      model.go

  server/
    http/
      router.go
      handlers_runtime_env.go
      handlers_auth.go
    middleware/
      auth.go
      logging.go
    auth/
      service.go
      token.go
      azure.go
    project/
      service.go
    secret/
      service.go
      crypto.go
    access/
      service.go
      policy.go
    audit/
      service.go
    storage/
      postgres.go
      queries/
```

This is enough structure to learn architecture without creating a Go version of an enterprise C# template.

---

## 17. Reflection questions after each commit

Ask yourself:

1. Did I create an abstraction before I had two examples?
2. Is this package name about behavior, or is it just a technical bucket?
3. Could a new developer find the feature quickly?
4. Can I test this without starting the whole app?
5. Does the CLI know something only the server should know?
6. Does the HTTP handler contain business logic?
7. Did I pass `context.Context` through I/O code?
8. Am I logging anything sensitive?
9. Is the code simpler than my first instinct?

---

## 18. Your first concrete target

Do this first:

```bash
# terminal 1
go run ./cmd/devsec-server

# terminal 2
go run ./cmd/devsec run demo -- env
```

Expected behavior:

```text
1. CLI calls server.
2. Server returns fake env vars.
3. CLI injects env vars.
4. CLI runs `env`.
5. Output contains DEVSEC_EXAMPLE=hello.
```

This proves the architecture before you add auth, Postgres, encryption, web UI, or Bubble Tea.

That is the Go way: a small working vertical slice, then careful growth.

---

## 19. Recommended reading / package docs to check while building

- Go official module/project layout guidance.
- Effective Go sections on names, interfaces, and simplicity.
- Gin documentation.
- Cobra documentation.
- Bubble Tea examples.
- pgx and sqlc documentation.
- Go `context`, `os/exec`, `net/http`, `log/slog`, and `crypto` standard library docs.

Do not read everything first. Read just enough before each commit.

---

## 20. Done criteria for the learning project

You are “done enough” when you can explain and demonstrate:

- why there are two binaries.
- why the CLI is thin.
- why the server owns authorization.
- how Gin handlers differ from services.
- where interfaces are useful and where they are noise.
- how context flows through the app.
- how errors move from storage to service to HTTP/CLI.
- how secrets avoid logs and plaintext storage.
- how Postgres access works with migrations and generated queries.
- how to run a subprocess with injected env vars.
- how to add Bubble Tea without mixing TUI state with business logic.

If you can explain those clearly, the project succeeded even if the UI is ugly and the feature set is small.
