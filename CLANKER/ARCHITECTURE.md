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

Startup and wiring live in `cmd/server/main.go`; environment parsing and validation live in `internal/config`. PostgreSQL, Redis, HTTP listen address, and worker count use `HORUS_*` variables with direct-local defaults (localhost dependencies, `:8080`, three workers). Required non-password connection fields and HTTP address cannot be blank; ports must be 1–65535 and workers 1–1000. Empty database passwords are accepted when PostgreSQL permits passwordless authentication. The process exits if its initial PostgreSQL pool ping or Redis ping fails; the server does not start degraded. `compose.yaml` provides PostgreSQL 17, Redis 8, a one-shot migration job, and the Horus server. Compose waits for PostgreSQL and Redis health and successful migrations before starting Horus. Its Horus healthcheck calls `/ready`.

## Packages

```text
cmd/server/        process startup, dependency wiring, routes, liveness and readiness handlers
cmd/migrate/       apply pending PostgreSQL schema migrations, then exit
internal/api/      monitor, check, and incident HTTP handlers / JSON contracts
internal/check/    check result types, HTTP checker, check service and incident transitions
internal/incident/ incident domain model and failure context
internal/database/ PostgreSQL connection pool construction
internal/monitor/  monitor domain model and validation
internal/migrate/  ordered, transactional migration runner
internal/notification/ optional Discord webhook transport
internal/postgres/ PostgreSQL monitor, check, and incident repositories
internal/queue/    Redis list client and check job payload
internal/scheduler/ due-monitor scheduling
internal/worker/   bounded check worker pool
migrations/        SQL schema migrations embedded in the migration binary
```

The domain packages do not import PostgreSQL. Repository implementations in `internal/postgres` depend on the domain packages and `pgxpool`.

## Runtime flow

### Monitor requests

The API strictly decodes one monitor-creation JSON value, rejects unknown fields and trailing data, and validates name, URL syntax, positive interval/timeout, stored-seconds range, and expected status through the request and `monitor.New`. URL DNS/IP safety remains a checker-time concern. The repository owns SQL access. Listing and retrieval map persisted models to API response structs. GET, DELETE, enable, and disable return `404` for missing or malformed UUIDs; other database failures return safe `500` responses. The repository maps no-row lookups and zero affected mutation rows to `monitor.ErrNotFound`, without a pre-mutation lookup. Enable/disable update state and return `204`; delete returns `204` and removes a monitor, with check history cascading through the existing foreign key.

### Scheduling and checks

The scheduler wakes every second, lists monitors, skips disabled or not-yet-due monitors, pushes a `MonitorID` job to Redis, then advances the due monitor's `next_check_at` by one interval. An enqueue or schedule-update failure is logged and does not prevent later monitors in the same pass from being attempted. An enqueue failure leaves the monitor due; after an update error, the next tick reads its persisted schedule again. Redis uses the `horus:checks` list (`LPUSH`/`BRPOP`). Each blocking dequeue waits at most one second before checking for cancellation and blocking again; this avoids a busy loop. Three workers are started at process startup. Each worker loads the monitor, runs the checker through `CheckService`, and persists the result.

The checker performs a GET with a timeout derived from the monitor, measures elapsed time, and compares the response status to `expected_status`. Request errors are classified as `network` or `timeout`; mismatched HTTP responses are `http`. The check service persists every result first, then opens an incident for failures or resolves the open incident after success. PostgreSQL returns the incident only when an insert or resolution actually changes a row; repeated failures and ordinary successes do not produce transitions. On each transition, the service sends one DOWN or RECOVERED notification through the optional notifier. Messages include monitor details, initial failure context, and incident times; recovery includes outage duration. If the incident transition fails, the check remains persisted and the service returns an explicit error. If Discord fails after a transition, check and incident state remain persisted and the service returns an error for the worker to log. Workers back off after dequeue errors. There is no check execution retry, job acknowledgement/dead-letter strategy, or duplicate-job suppression.

`HORUS_DISCORD_WEBHOOK_URL` enables Discord notifications. An unset or blank value disables them without affecting startup; nonblank values require an absolute HTTP(S) URL with a hostname and no embedded credentials. The Discord client uses a context-aware HTTP request with a five-second deadline and treats non-2xx responses as delivery failures. Notification delivery is synchronous with check processing and has no durable queue or retry.

Configuration uses `os.LookupEnv`; the application does not read `.env`. Local shell settings must be exported to the process before startup. The server logs `Discord notifications enabled` or `Discord notifications disabled` when constructing the notifier. Request errors expose only safe failure categories (including cancellation and timeout), while non-2xx errors include the HTTP status; neither includes the webhook URL. Existing open incidents do not generate a DOWN notification when configuration is enabled later.

### Shutdown

Interrupt and SIGTERM cancel the root context. Workers are cancelled and joined, then the HTTP server receives a five-second graceful-shutdown deadline. An idle Redis dequeue notices cancellation after its current blocking wait, normally within one second; in-flight operations still depend on their own cancellation behavior. PostgreSQL and Redis clients are closed on process exit.

### Liveness and readiness

`GET /health` returns `{"status":"ok"}` whenever the HTTP process serves the request; it performs no dependency I/O. `GET /ready` pings the existing PostgreSQL pool and Redis client with the request context and a shared one-second deadline. The Redis client has context timeouts enabled so the deadline also bounds its socket I/O. Readiness returns `200 {"status":"ready"}` only when both pings succeed before the deadline; otherwise it returns `503 {"status":"not_ready"}`. Responses never include raw dependency errors or connection details. Startup still requires successful initial PostgreSQL and Redis connections.

## Persistence

Migrations remain numbered SQL files under `migrations/`, embedded in `cmd/migrate`. The executable creates `schema_migrations`, applies pending versions in numeric order, and records each version in the same PostgreSQL transaction as its SQL. A failed migration rolls back and exits nonzero. The server does not apply migrations itself; Compose runs the migration job after PostgreSQL is healthy and gates server startup on its successful exit. Repeated runs skip recorded versions. PostgreSQL's `postgres_data` named volume survives `docker compose down` and later starts. A preexisting manually migrated database without `schema_migrations` needs an explicit baseline or a fresh volume; the runner does not infer past versions.

`monitors` stores UUID, name, URL, interval/timeout in seconds, expected status, enabled flag, timestamps, and (after migration 003) `next_check_at`. `checks` stores UUID, monitor foreign key with cascade delete, nullable status code, latency in milliseconds, success, failure type, nullable error, and check timestamp. An index supports per-monitor history ordered by recent check time. `incidents` stores outage start, nullable resolution, and nonnullable initial failure type, status code (default 0), and message (default empty). A partial unique index permits only one unresolved incident per monitor.

## CI

`.github/workflows/ci.yml` checks pushes to `main` and pull requests targeting `main`. One Ubuntu job starts PostgreSQL 17 and Redis 8 service containers with healthchecks, runs `cmd/migrate` against a fresh database, then executes Go tests, race tests, vet, build, and a Docker image build. The job uses local test credentials and a blank Discord webhook. Migration integration tests create an isolated schema inside a transaction. The Redis queue integration test uses DB 14, separate from the application's DB 0 and the cancellation test's DB 15.

## HTTP API

Routes are registered with Go's `net/http` method/path patterns in `cmd/server/main.go`:

```text
GET    /health
GET    /ready
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

Handlers use small repository interfaces defined at the API boundary. Check and incident history accept `limit` (default 50, range 1–100) and `offset` (default 0, nonnegative), and return rows newest first. Check history rejects empty, noninteger, or out-of-range pagination with `400`, returns `[]` for an existing monitor without checks, and orders equal timestamps by check ID descending. The summary's latest status uses the same ordering. Nullable stored check status and error values appear as status 0 and an omitted error field. Incident history rejects malformed pagination with `400`, returns `[]` when empty, and orders by `started_at DESC, id DESC`. The current incident query selects only `resolved_at IS NULL`; it returns the open incident or `204` with an empty body when none is open. Both incident routes return `404` for a missing monitor or malformed UUID. Incident responses include start and optional resolution timestamps, open state, initial failure context, and duration in milliseconds: elapsed time at response for an open incident and fixed start-to-resolution time for a resolved one. Negative durations clamp to zero.

The summary reports counts, average latency in whole milliseconds (truncated from the aggregate), latest HTTP status, and `uptime_percentage`. Uptime is `successful_checks / total_checks × 100`, rounded to two decimals. It counts persisted check outcomes and is not time weighted; no checks yields `null` uptime and zero values for the other summary fields. A latest status of 0 means the newest check had no HTTP response, or that no checks exist. Check history and summary return `404` for a missing monitor or malformed UUID.

Monitor, check, and incident handlers share a small JSON error writer. Handler-produced `400` and `404` responses retain useful safe messages in `{"error":"..."}`; unexpected repository errors log the operation and error type and return `500 {"error":"internal server error"}`. The helper sets `Content-Type: application/json`. Successful responses are unchanged, including empty `204` responses. `/ready` deliberately retains `503 {"status":"not_ready"}` for dependency failure. The standard Go mux handles unsupported routes and methods.

## Boundaries and limitations

- PostgreSQL is the source of truth; Redis is currently a job transport, not a cache or durable business store.
- The scheduler currently shares one process with API and workers; Redis does not imply independently deployed workers.
- Worker count and service settings are configurable through environment variables, but there is no production configuration profile or secret management.
- The checker allows only HTTP/HTTPS URLs without user information, blocks non-public DNS results during dialing, and does not follow redirects. SSRF defenses should still be reviewed and extended as needed.
- There is no authentication, authorization, metrics, or durable notification delivery. Readiness checks connectivity only; it does not confirm migrations, scheduler progress, worker activity, or delivery of queued checks. Uptime reflects the proportion of successful checks rather than elapsed availability. Check persistence and incident transitions are separate operations rather than one transaction, so a transition failure can leave incident state that does not reflect the latest persisted check until a later check transitions it. A Discord delivery failure is reported but not retried, so that transition's notification may be missed.
