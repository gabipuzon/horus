# ARCHITECTURE.md

## SYSTEM

- Pattern: modular monolith initially; distributed workers introduced where justified.
- Language: Go.
- API: HTTP/JSON.
- Database: PostgreSQL.
- Queue/cache: Redis.
- Deployment: Docker-compatible.
- Principle: keep synchronous core simple; introduce async processing for workload isolation and scalability.

## TARGET ARCHITECTURE

```text
                    ┌──────────────┐
                    │    Client    │
                    └──────┬───────┘
                           │ HTTP
                           ▼
                    ┌──────────────┐
                    │  Horus API   │
                    └──────┬───────┘
                           │
             ┌─────────────┴─────────────┐
             ▼                           ▼
      ┌─────────────┐             ┌─────────────┐
      │ PostgreSQL  │             │    Redis    │
      └─────────────┘             └──────┬──────┘
                                         │ jobs
                                         ▼
                                  ┌─────────────┐
                                  │   Workers   │
                                  └──────┬──────┘
                                         │ HTTP
                                         ▼
                                  ┌─────────────┐
                                  │   Targets   │
                                  └─────────────┘
                                         │
                                         ▼
                                  CheckResult
                                         │
                           ┌─────────────┴─────────────┐
                           ▼                           ▼
                     Check History                Incident
                                                       │
                                                       ▼
                                                 Notifications
````

## PACKAGE STRUCTURE

Expected high-level structure:

```text
horus/
├── cmd/
│   └── server/
├── internal/
│   ├── api/
│   ├── database/
│   ├── monitor/
│   ├── check/
│   ├── incident/
│   ├── notification/
│   ├── scheduler/
│   ├── worker/
│   └── ...
├── migrations/
├── compose.yaml
├── go.mod
└── README.md
```

Package boundaries should reflect domain responsibility rather than arbitrary technical layers.

## DOMAIN FLOW

### Monitor Configuration

```text
API
 ↓
Monitor validation
 ↓
Monitor domain
 ↓
Monitor repository
 ↓
PostgreSQL
```

### Scheduled Check

```text
Scheduler
 ↓
Select enabled monitors
 ↓
Create check job
 ↓
Redis
 ↓
Worker
 ↓
HTTP Checker
 ↓
CheckResult
 ↓
Persistence
```

### Incident Detection

```text
CheckResult
 ↓
Incident evaluator
 ├── success
 │     └── recover existing incident
 │
 └── failure
       └── open/update incident
```

### Notification

```text
Incident state change
 ↓
Notification job
 ↓
Redis
 ↓
Notification worker
 ↓
Provider
```

## CURRENT TECHNOLOGY

### Go

* Standard library preferred where sufficient.
* `net/http` for HTTP server.
* `context.Context` for request/job cancellation.
* `time.Duration` for internal timing.
* Explicit dependency injection for external resources.

### PostgreSQL

* Primary durable datastore.
* Monitor configuration stored here.
* Check history stored here.
* Incident history stored here.
* User/authentication data stored here.
* Migrations are versioned SQL files.

### pgx

* PostgreSQL driver/client.
* `pgxpool.Pool` for connection pooling.
* Repository layer owns SQL access.
* Domain packages should not directly depend on SQL where avoidable.

### Redis

Reserved for asynchronous coordination:

* Check jobs
* Notification jobs
* Retries
* Queue state where required

Redis should not become the authoritative store for durable business state.

## API ARCHITECTURE

Handlers:

* Decode HTTP input.
* Validate request-level constraints.
* Invoke domain/repository operations.
* Map domain results to API responses.
* Return appropriate HTTP status codes.

Handlers should not contain:

* Raw SQL.
* Scheduling logic.
* HTTP monitoring logic.
* Incident state machines.
* Notification delivery logic.

## REPOSITORY ARCHITECTURE

Repositories abstract persistence operations.

Example:

```go
type MonitorRepository interface {
    Create(ctx context.Context, m *Monitor) error
    List(ctx context.Context) ([]*Monitor, error)
    GetByID(ctx context.Context, id string) (*Monitor, error)
    Delete(ctx context.Context, id string) error
    SetEnabled(ctx context.Context, id string, enabled bool) error
}
```

Rules:

* Accept `context.Context`.
* Return explicit errors.
* Keep SQL inside repository implementations.
* Do not expose database-specific types unnecessarily.
* Interfaces belong near the consumer when practical.

## CHECKER ARCHITECTURE

The checker executes one monitor request.

Responsibilities:

* Build HTTP request.
* Apply timeout.
* Execute request.
* Measure latency.
* Validate expected status.
* Classify failures.
* Return `CheckResult`.

Checker does not:

* Persist results.
* Create incidents.
* Send notifications.
* Schedule itself.

## SCHEDULER

Responsibilities:

* Determine which enabled monitors require execution.
* Respect monitor intervals.
* Submit check jobs.
* Prevent duplicate/overlapping work.

Scheduler does not perform HTTP checks directly once worker architecture is introduced.

## WORKERS

Workers:

* Consume jobs.
* Execute bounded concurrent work.
* Respect context cancellation.
* Persist results.
* Trigger downstream processing.
* Handle retry policy.

Worker count and concurrency should be configurable.

## INCIDENT MODEL

Incident state should be deterministic from check outcomes.

Minimum conceptual states:

```text
open
resolved
```

Rules:

* One active incident per monitor.
* Repeated failures update the existing incident.
* A successful check can resolve an active incident.
* Recovery should not create a second incident.

## DATABASE PRINCIPLES

* PostgreSQL is source of truth.
* Use UUID primary keys for externally meaningful entities.
* Store timestamps as `TIMESTAMPTZ`.
* Store internal durations using explicit units.
* API units must be explicit, e.g. seconds.
* Use indexes based on actual query patterns.
* Foreign keys enforce relationships.
* Schema changes require migrations.

## RELIABILITY

Required architectural concerns:

* Request timeouts.
* Context cancellation.
* Bounded concurrency.
* Retry/backoff where appropriate.
* Graceful shutdown.
* Database connection pooling.
* Queue failure handling.
* Duplicate-job protection.
* SSRF protection for user-controlled target URLs.

## OBSERVABILITY

Expose:

* Request metrics.
* Check success/failure metrics.
* Check latency.
* Worker/job metrics.
* Queue metrics.
* Database metrics.
* Application health.
* Readiness state.

Logs should contain enough structured context to trace:

```text
request → monitor → check → incident → notification
```

## SECURITY

Required protections:

* Authentication before protected resources.
* Authorization by resource ownership.
* Password hashing.
* Request size limits.
* Rate limiting where appropriate.
* SSRF prevention.
* Safe URL validation.
* Security-conscious HTTP client configuration.
* Secrets supplied through configuration/environment, not source code.

## ARCHITECTURAL CONSTRAINTS

* Do not introduce microservices merely for organizational appearance.
* Do not use Redis as a replacement for PostgreSQL.
* Do not couple handlers directly to infrastructure.
* Do not put business state transitions inside SQL-only logic.
* Do not allow unbounded worker concurrency.
* Do not allow user-controlled monitoring targets to bypass SSRF protections.
* Prefer measurable requirements over speculative abstractions.

## EVOLUTION

Architecture may evolve when implementation demonstrates a real need.

Decision order:

```text
simple implementation
      ↓
measure / identify constraint
      ↓
introduce abstraction or infrastructure
      ↓
test behavior
      ↓
document architectural reason
```

Implementation state and completed work belong exclusively in `PROGRESS.md`.
