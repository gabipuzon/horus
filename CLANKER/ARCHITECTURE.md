# Architecture

## Current system

Horus is a single Go process with an HTTP API, scheduler, and worker pool. PostgreSQL stores monitor configuration, check history, and incidents. Redis transports check jobs between the scheduler and workers.

```text
HTTP client ──> net/http API ──> PostgreSQL

PostgreSQL monitors ──> 1s scheduler ──> Redis list `horus:checks`
                                                │
                                      3 blocking workers
                                                │
                                      HTTP checker / target
                                                │
                                      check result ──> PostgreSQL
                                                │
                                      incident transition
                                                │
                                      optional Discord webhook
```

Startup and wiring live in `cmd/server/main.go`; environment parsing and validation live in `internal/config`. PostgreSQL, Redis, HTTP listen address, and worker count use `HORUS_*` variables with local Compose-compatible defaults. The process checks PostgreSQL and Redis at startup. `compose.yaml` provides PostgreSQL 17 and Redis 8 for local development.

## Packages

```text
cmd/server/        process startup, dependency wiring, routes, health handler
internal/api/      monitor, check, and incident HTTP handlers / JSON contracts
internal/check/    check result types, HTTP checker, check service and incident transitions
internal/incident/ incident domain model and failure context
internal/database/ PostgreSQL connection pool construction
internal/monitor/  monitor domain model and validation
internal/notification/ optional Discord webhook transport
internal/postgres/ PostgreSQL monitor, check, and incident repositories
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

The scheduler wakes every second, lists monitors, skips disabled or not-yet-due monitors, pushes a `MonitorID` job to Redis, then advances the due monitor's `next_check_at` by one interval. An enqueue or schedule-update failure is logged and does not prevent later monitors in the same pass from being attempted. An enqueue failure leaves the monitor due; after an update error, the next tick reads its persisted schedule again. Redis uses the `horus:checks` list (`LPUSH`/`BRPOP`). Each blocking dequeue waits at most one second before checking for cancellation and blocking again; this avoids a busy loop. Three workers are started at process startup. Each worker loads the monitor, runs the checker through `CheckService`, and persists the result.

The checker performs a GET with a timeout derived from the monitor, measures elapsed time, and compares the response status to `expected_status`. Request errors are classified as `network` or `timeout`; mismatched HTTP responses are `http`. The check service persists every result first, then opens an incident for failures or resolves the open incident after success. PostgreSQL returns the incident only when an insert or resolution actually changes a row; repeated failures and ordinary successes do not produce transitions. On each transition, the service sends one DOWN or RECOVERED notification through the optional notifier. Messages include monitor details, initial failure context, and incident times; recovery includes outage duration. If the incident transition fails, the check remains persisted and the service returns an explicit error. If Discord fails after a transition, check and incident state remain persisted and the service returns an error for the worker to log. Workers back off after dequeue errors. There is no check execution retry, job acknowledgement/dead-letter strategy, or duplicate-job suppression.

`HORUS_DISCORD_WEBHOOK_URL` enables Discord notifications. An unset or blank value disables them without affecting startup. The Discord client uses a context-aware HTTP request with a five-second deadline and treats non-2xx responses as delivery failures. Notification delivery is synchronous with check processing and has no durable queue or retry.

### Shutdown

Interrupt and SIGTERM cancel the root context. Workers are cancelled and joined, then the HTTP server receives a five-second graceful-shutdown deadline. An idle Redis dequeue notices cancellation after its current blocking wait, normally within one second; in-flight operations still depend on their own cancellation behavior. PostgreSQL and Redis clients are closed on process exit.

## Persistence

Migrations are plain SQL files and are not applied automatically by startup. Apply `001_create_monitors.sql`, `002_create_checks.sql`, `003_add_monitor_next_check_at.sql`, then `004_create_incidents.sql` in order.

`monitors` stores UUID, name, URL, interval/timeout in seconds, expected status, enabled flag, timestamps, and (after migration 003) `next_check_at`. `checks` stores UUID, monitor foreign key with cascade delete, nullable status code, latency in milliseconds, success, failure type, nullable error, and check timestamp. An index supports per-monitor history ordered by recent check time. `incidents` stores outage start, optional resolution, and initial failure context. A partial unique index permits only one unresolved incident per monitor.

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
GET    /monitors/{id}/incidents
GET    /monitors/{id}/incidents/current
```

Handlers use small repository interfaces defined at the API boundary. Check and incident history accept `limit` (default 50, range 1–100) and `offset` (default 0, nonnegative), and return rows newest first. Incident history rejects malformed pagination with `400`; check history retains its existing parsing behavior, which falls back to defaults for noninteger values. Incident history returns `[]` when empty. The current incident route returns the open incident or `204` when none is open. Both routes return `404` for a missing monitor. Incident responses include start and optional resolution timestamps, open state, initial failure context, and duration in milliseconds: elapsed time at response for an open incident and start-to-resolution time for a resolved one.

The summary reports counts, average latency, latest HTTP status, and `uptime_percentage`. Uptime is `successful_checks / total_checks × 100`, rounded to two decimals. It counts persisted check outcomes and is not time weighted; no checks yields `null` uptime and zero values for the other summary fields. Check history and summary also return `404` for a missing monitor.

## Boundaries and limitations

- PostgreSQL is the source of truth; Redis is currently a job transport, not a cache or durable business store.
- The scheduler currently shares one process with API and workers; Redis does not imply independently deployed workers.
- Worker count and service settings are configurable through environment variables, but there is no production configuration profile or secret management.
- The checker allows only HTTP/HTTPS URLs without user information, blocks non-public DNS results during dialing, and does not follow redirects. SSRF defenses should still be reviewed and extended as needed.
- There is no authentication, authorization, readiness endpoint, metrics, or durable notification delivery. Uptime reflects the proportion of successful checks rather than elapsed availability. Check persistence and incident transitions are separate operations rather than one transaction, so a transition failure can leave incident state that does not reflect the latest persisted check until a later check transitions it. A Discord delivery failure is reported but not retried, so that transition's notification may be missed.
