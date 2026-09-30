# Architecture

## Current system

Horus is a single Go process with an HTTP API, scheduler, and worker pool. PostgreSQL stores monitor configuration and check history. Redis transports check jobs between the scheduler and workers.

```text
HTTP client ──> net/http API ──> PostgreSQL

PostgreSQL monitors ──> 1s scheduler ──> Redis list `horus:checks`
                                                │
                                      3 blocking workers
                                                │
                                      HTTP checker / target
                                                │
                                      check result ──> PostgreSQL
```

Startup and wiring live in `cmd/server/main.go`; environment parsing and validation live in `internal/config`. PostgreSQL, Redis, HTTP listen address, and worker count use `HORUS_*` variables with local Compose-compatible defaults. The process checks PostgreSQL and Redis at startup. `compose.yaml` provides PostgreSQL 17 and Redis 8 for local development.

## Packages

```text
cmd/server/        process startup, dependency wiring, routes, health handler
internal/api/      monitor and check HTTP handlers / JSON contracts
internal/check/    check result types, HTTP checker, check service
internal/database/ PostgreSQL connection pool construction
internal/monitor/  monitor domain model and validation
internal/postgres/ PostgreSQL monitor and check repositories
internal/queue/    Redis list client and check job payload
internal/scheduler/ due-monitor scheduling
internal/worker/   bounded check worker pool
migrations/        SQL schema migrations
```

The domain packages do not import PostgreSQL. Repository implementations in `internal/postgres` depend on the domain packages and `pgxpool`.

## Runtime flow

### Monitor requests

The API decodes and validates monitor creation through `monitor.New`, then calls the monitor repository. The repository owns SQL access. Listing and retrieval map persisted models to API response structs. Enable/disable update state; delete removes a monitor and the checks migration's foreign key cascades to its check history.

### Scheduling and checks

The scheduler wakes every second, lists monitors, skips disabled or not-yet-due monitors, advances each due monitor's `next_check_at` by one interval, and pushes a `MonitorID` job to Redis. Redis uses the `horus:checks` list (`LPUSH`/blocking `BRPOP`). Three workers are started at process startup. Each worker loads the monitor, runs the checker through `CheckService`, and persists the result.

The checker performs a GET with a timeout derived from the monitor, measures elapsed time, and compares the response status to `expected_status`. Request errors are classified as `network` or `timeout`; mismatched HTTP responses are `http`. The check service persists both successful and failed results. There is no retry, backoff, job acknowledgement/dead-letter strategy, duplicate suppression, or incident processing implemented.

### Shutdown

Interrupt and SIGTERM cancel the root context. Workers are cancelled and joined, then the HTTP server receives a five-second graceful-shutdown deadline. PostgreSQL and Redis clients are closed on process exit.

## Persistence

Migrations are plain SQL files and are not applied automatically by startup. Apply `001_create_monitors.sql`, `002_create_checks.sql`, then `003_add_monitor_next_check_at.sql` in order.

`monitors` stores UUID, name, URL, interval/timeout in seconds, expected status, enabled flag, timestamps, and (after migration 003) `next_check_at`. `checks` stores UUID, monitor foreign key with cascade delete, nullable status code, latency in milliseconds, success, failure type, nullable error, and check timestamp. An index supports per-monitor history ordered by recent check time.

## HTTP API

Routes are registered with Go's `net/http` method/path patterns in `cmd/server/main.go`:

```text
GET    /health
POST   /monitors
GET    /monitors
GET    /monitors/{id}
DELETE /monitors/{id}
PATCH  /monitors/{id}/enable
PATCH  /monitors/{id}/disable
GET    /monitors/{id}/checks
GET    /monitors/{id}/summary
```

Handlers use small repository interfaces defined at the API boundary. Check history accepts `limit` (default 50, range 1–100) and `offset` (default 0, nonnegative), and returns rows newest first. The summary reports counts, average latency, and latest HTTP status; no checks yields zero values. Unknown/malformed pagination values that fail integer parsing currently fall back to defaults.

## Boundaries and limitations

- PostgreSQL is the source of truth; Redis is currently a job transport, not a cache or durable business store.
- The scheduler currently shares one process with API and workers; Redis does not imply independently deployed workers.
- Worker count and service settings are configurable through environment variables, but there is no production configuration profile or secret management.
- The checker allows only HTTP/HTTPS URLs without user information, blocks non-public DNS results during dialing, and does not follow redirects. SSRF defenses should still be reviewed and extended as needed.
- There is no authentication, authorization, readiness endpoint, metrics, or incident/notification subsystem.
