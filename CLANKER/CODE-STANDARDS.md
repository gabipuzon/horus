# CODE-STANDARDS.md

## PURPOSE

- Define implementation conventions for Horus.
- Optimize for correctness, readability, testability, and low cognitive overhead.
- Prefer explicit code over clever abstractions.
- Follow existing repository patterns before introducing new ones.

## GO

### Formatting

- Run `gofmt` on changed Go files.
- Keep imports formatted by `gofmt`.
- Do not manually format Go code against standard tooling.

### Naming

- Use idiomatic Go names.
- `CamelCase` for exported identifiers.
- `camelCase` for unexported identifiers.
- Avoid unnecessary abbreviations.
- Acronyms use conventional Go casing: `ID`, `URL`, `HTTP`, `API`, `DB`.
- Names should describe responsibility, not implementation trivia.

### Functions

- Keep functions focused.
- Prefer small functions with one clear responsibility.
- Avoid premature helper extraction.
- Avoid functions whose primary purpose is merely reducing line count.

### Errors

- Return errors explicitly.
- Handle errors at the appropriate boundary.
- Add context when useful:

```go
return fmt.Errorf("create monitor: %w", err)
````

* Do not silently discard meaningful errors.
* Do not use `panic` for normal application failures.
* HTTP handlers should translate internal errors into appropriate HTTP responses.
* Do not expose sensitive internal errors directly to clients.

### Context

* Functions performing I/O accept `context.Context`.
* Propagate request/job context.
* Do not create unnecessary background contexts inside request paths.
* Respect cancellation and deadlines.

## PACKAGES

* Group code by domain responsibility.
* Keep packages cohesive.
* Avoid circular dependencies.
* Avoid generic dumping-ground packages such as `utils`, `helpers`, or `common` unless a concrete shared responsibility exists.
* Keep infrastructure details behind appropriate boundaries.

Preferred direction:

```text
api → domain/application → repository/infrastructure
```

Avoid:

```text
api → SQL
api → Redis internals
domain → HTTP handlers
domain → API response structs
```

## STRUCTS

* Keep domain models independent from transport formats.
* Use API-specific request/response structs.
* Do not expose internal implementation details unnecessarily.
* Use explicit JSON tags on API structs.
* Avoid using one struct simultaneously as domain model and API contract when their responsibilities differ.

Example:

```go
type createMonitorRequest struct {
    Name           string `json:"name"`
    URL            string `json:"url"`
    Interval       int    `json:"interval_seconds"`
}
```

## INTERFACES

* Define interfaces at the point of consumption when practical.
* Keep interfaces small.
* Depend on behavior, not concrete implementations.
* Do not create interfaces speculatively.

Preferred:

```go
type MonitorRepository interface {
    Create(ctx context.Context, m *monitor.Monitor) error
}
```

Avoid large interfaces containing unrelated operations.

## DEPENDENCY INJECTION

Inject external dependencies when they affect:

* Database access
* HTTP clients
* Redis
* Queues
* Notification providers
* Time-sensitive behavior when deterministic testing requires it

Constructors should make dependencies explicit.

Example:

```go
func NewChecker(client *http.Client) *Checker
```

Avoid hidden global dependencies.

## HTTP API

### Routes

Use descriptive REST-oriented routes.

Current style:

```text
POST   /monitors
GET    /monitors
GET    /monitors/{id}
DELETE /monitors/{id}
PATCH  /monitors/{id}/enable
PATCH  /monitors/{id}/disable
```

### Status Codes

Use conventional HTTP semantics:

```text
200 OK
201 Created
204 No Content
400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
429 Too Many Requests
500 Internal Server Error
```

### Request Handling

Handlers should:

1. Decode input.
2. Validate request-level input.
3. Invoke application/domain behavior.
4. Map results to API responses.
5. Return the correct status.

Handlers should not contain database queries or monitoring algorithms.

### API Contracts

* Use explicit JSON field names.
* Use stable external units.
* Do not expose `time.Duration` directly as JSON.
* Use explicit fields such as `interval_seconds`.
* Keep response structures independent from database schemas.

## DATABASE

### SQL

* SQL belongs in repository/infrastructure code.
* Use parameterized queries.
* Never concatenate user input into SQL.
* Keep queries readable.
* Select explicit columns rather than `SELECT *`.

Preferred:

```sql
SELECT
    id,
    name,
    url,
    enabled
FROM monitors
WHERE id = $1
```

### Transactions

Use transactions when multiple database operations must succeed or fail together.

Do not introduce transactions around isolated operations without a concrete consistency requirement.

### Migrations

* One logical schema change per migration where practical.
* Number migrations sequentially.
* Use descriptive filenames.

Example:

```text
001_create_monitors.sql
002_create_checks.sql
003_create_incidents.sql
```

### Timestamps

* Store timestamps as `TIMESTAMPTZ`.
* Use UTC-compatible application behavior.
* Keep `CreatedAt` immutable.
* Update `UpdatedAt` when mutable state changes.

### IDs

* Use UUIDs for externally meaningful entities.
* Generate IDs in the application/domain layer unless database generation is explicitly chosen by architecture.

## TIME AND DURATIONS

* Use `time.Duration` internally.
* Convert to explicit units at persistence/API boundaries.
* Avoid raw numeric duration values without documented units.

Example:

```go
time.Duration(seconds) * time.Second
```

## TESTING

### General

* Every meaningful behavior should have automated coverage.
* Tests should verify behavior rather than implementation details.
* Run:

```bash
go test ./...
```

before declaring a change complete.

### Unit Tests

Use for:

* Domain validation
* Failure classification
* Pure logic
* Small state transitions

### API Tests

Use `httptest` for handlers.

Test:

* Successful requests
* Invalid input
* Not-found behavior
* Relevant error paths
* Response structure
* Status codes

### Fakes

Use small fakes for interfaces.

Example:

```go
type fakeMonitorRepository struct {
    created *monitor.Monitor
}
```

Avoid excessive mocking frameworks when simple fakes are sufficient.

### Integration Tests

Use real PostgreSQL/Redis where behavior depends on infrastructure.

Do not claim an integration test exists when only a fake was used.

## HTTP CHECKING

* Always apply request timeouts.
* Propagate context.
* Measure latency around the actual request.
* Close response bodies.
* Classify failures explicitly.
* Do not treat every request failure as an HTTP-status failure.

Current failure categories:

```text
http
network
timeout
```

## CONCURRENCY

* Prefer bounded concurrency.
* Never spawn unbounded goroutines based directly on external workload.
* Use context cancellation.
* Define worker ownership clearly.
* Avoid shared mutable state unless synchronization is explicit.

## LOGGING

* Use structured logging once the logging layer is introduced.
* Include useful identifiers:

  * monitor ID
  * check ID
  * incident ID
  * job ID
* Never log secrets or credentials.
* Avoid noisy logs for expected control flow.

## CONFIGURATION

* Environment/configuration values belong outside source code.
* Do not hardcode production credentials.
* Configuration should have explicit defaults where appropriate.
* Validate required configuration at startup.

## SECURITY

### User-Controlled URLs

Monitoring targets are untrusted input.

Required protections include:

* URL parsing/validation.
* Scheme restrictions.
* Private/internal network protection.
* Loopback protection.
* Cloud metadata endpoint protection.
* DNS/rebinding considerations.
* Redirect validation.

### Secrets

* Never commit credentials.
* Never log credentials.
* Pass secrets through environment/configuration or secret-management infrastructure.

## GIT

* Make commits small and logically coherent.
* One logical change per commit.
* Use conventional-style commit messages.

Examples:

```text
feat: add monitor listing repository
feat: add monitor lookup handler
test: add monitor deletion API tests
fix: classify request deadline failures
refactor: simplify monitor validation
```

* Do not mix unrelated refactors with feature changes.
* Run tests before committing.

## CHANGE DISCIPLINE

When modifying existing code:

1. Inspect current implementation.
2. Identify the smallest required change.
3. Preserve unrelated behavior.
4. Update tests.
5. Format code.
6. Run tests.
7. Review the diff.
8. Commit the logical change.

## FORBIDDEN PATTERNS

Avoid:

* Global mutable state.
* Hidden dependencies.
* Unbounded goroutines.
* Raw SQL in handlers.
* Business logic inside HTTP routing.
* Database models used directly as public API contracts.
* Premature abstraction.
* Large unrelated refactors.
* Ignoring returned errors.
* `panic` for recoverable application errors.
* Hardcoded production secrets.
* Unvalidated user-controlled monitoring URLs.