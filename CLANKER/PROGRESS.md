# Project Progress

## Current state

Horus has an end-to-end core monitoring flow: manage monitors over HTTP, schedule due checks, enqueue jobs in Redis, execute checks with a bounded worker pool, persist checks and incidents in PostgreSQL, and query check and incident data through the API.

Phase 9A adds a separate frontend in `web/` for the monitor list and its creation, enable/disable, and delete flows. It shows enabled/disabled configuration state only; monitor detail, live health, checks, summary, and incident views are not built.

The code is organized by responsibility: `api`, `check`, `config`, `database`, `incident`, `migrate`, `monitor`, `notification`, `postgres`, `queue`, `scheduler`, and `worker` packages under `internal/`.

## Implemented

### Monitor API and persistence

- Monitor model with UUID IDs and validation for nonblank name, absolute HTTP/HTTPS URL without credentials, positive interval/timeout, and HTTP status range. Creation rejects values too large for the existing integer-seconds columns.
- PostgreSQL monitor repository and migrations for monitor state, including `next_check_at`.
- Create, list, retrieve, delete, enable, and disable monitor endpoints. Create strictly decodes one JSON object and rejects unknown fields or trailing data. Missing or malformed UUIDs return `404` for GET, DELETE, enable, and disable; successful delete/enable/disable return `204`. Repository mutations use affected-row counts to distinguish missing monitors from database failures.

### Checks and history

- HTTP GET checker with monitor timeout, latency measurement, expected-status comparison, and `http`/`network`/`timeout` failure classification.
- PostgreSQL check repository, foreign-key cascade, and monitor/time history index.
- Check history endpoint returns newest checks first, using check ID to break equal timestamp ties. `limit` defaults to 50 (allowed 1–100) and `offset` to 0 (nonnegative); malformed or out-of-range values return `400`. Existing monitors without checks return `200 []`; missing or malformed monitor IDs return `404`.
- Summary endpoint reports total/success/failure counts, average persisted latency truncated to whole milliseconds, latest HTTP status, and check-based uptime. Uptime is successful checks divided by all persisted checks, rounded to two decimals. With no checks, uptime is `null` and the other fields are zero; latest status 0 also represents a newest check without an HTTP response.

### Incident lifecycle

- PostgreSQL incidents are associated with monitors and store outage start, optional resolution, and the initial failure type, status code, and error message.
- At most one open incident per monitor is enforced by a partial unique index. Repeated failures leave the existing incident and its initial failure context unchanged; a successful check resolves it.
- Check results are persisted before incident transitions. If a transition fails, the check remains stored and the check service returns an explicit transition error.
- Migration `004_create_incidents.sql` adds incident storage and indexes. `cmd/migrate` now applies numbered SQL migrations transactionally and records completed versions in `schema_migrations`.
- Incident history uses strict `limit` 1–100 (default 50) and nonnegative `offset` (default 0), orders by start time then ID descending, and returns `200 []` when empty. The current incident endpoint returns only the open incident or an empty `204` when none is open. Both routes return `404` for missing or malformed monitor IDs. Responses include timestamps, open state, initial failure context, and elapsed or fixed resolved duration in milliseconds, clamped to zero if negative.

### Discord notifications

- Optional `HORUS_DISCORD_WEBHOOK_URL` enables DOWN on incident creation and RECOVERED on resolution. Repeated failures and ordinary successes send nothing.
- The webhook variable must be exported into the process environment; `.env` is not automatically loaded. Startup logs enabled/disabled state without the URL. Enabling notifications does not replay existing incidents.
- The webhook message includes monitor name and URL, incident failure context and start time, and recovery time and duration when resolved.
- Requests use the check context and a five-second deadline. A Discord failure leaves the persisted check and incident transition intact and is returned to the worker for logging; there is no delivery retry or durable queue.
- Request errors are safe to log without disclosing the webhook URL/token; HTTP failures retain their response status, and cancellation/timeout remain detectable through error unwrapping.

### Scheduling and workers

- One-second scheduler selects enabled monitors whose `next_check_at` is due, queues jobs in Redis list `horus:checks`, then advances their schedule. A per-monitor enqueue or schedule-update failure is logged without blocking later monitors in the same pass. Failed enqueues leave the monitor due; update errors are reevaluated from PostgreSQL on the next tick.
- Redis client provides ping, enqueue (`LPUSH`), and blocking dequeue (`BRPOP`) with a one-second wait so an idle worker can observe cancellation promptly.
- Application starts the configured worker count (three by default) in the same process; workers load monitors, run checks, and persist results.
- Process handles interrupt/SIGTERM, cancels and joins workers, and gracefully shuts down the HTTP server. Idle Redis dequeues no longer wait indefinitely before worker shutdown can finish.

### Runtime/API

- `GET /health` is process liveness and returns `{"status":"ok"}` without checking dependencies. `GET /ready` pings the existing PostgreSQL pool and Redis client with a shared one-second deadline, returning `200 {"status":"ready"}` when both respond or `503 {"status":"not_ready"}` otherwise. Responses contain no raw dependency errors.
- Local direct-run defaults are PostgreSQL and Redis on localhost, HTTP `:8080`, and three workers. Config rejects blank required connection fields and HTTP address, invalid ports, worker counts outside 1–1000, and nonblank Discord webhook URLs without an HTTP(S) hostname or with embedded credentials. Blank database passwords are allowed for PostgreSQL setups that support them. Startup exits if the initial PostgreSQL or Redis check fails; there is no degraded startup mode.
- Handler-produced monitor, check, and incident errors use `application/json` with one `error` string. Safe request errors remain `400`, missing or malformed monitor IDs remain `404`, and internal repository failures return a generic `500` while logging operation and error type. The readiness `503` status response and Go mux defaults remain separate conventions.
- A multi-stage Dockerfile builds the server and migration binaries; the runtime image uses CA certificates and a non-root user.
- Compose starts PostgreSQL 17 and Redis 8 with healthchecks, runs migrations after PostgreSQL becomes healthy, then starts Horus after Redis is healthy and migrations succeed. Horus container health uses `/ready`.
- Compose keeps PostgreSQL data in the `postgres_data` named volume. It reads local `.env` for variable substitution and sets `postgres`/`redis` service hostnames inside containers; the Go binaries still read only process environment variables.

### Continuous integration

- A single GitHub Actions workflow runs on pushes to `main` and pull requests targeting it, cancelling obsolete runs for the same branch or PR. It grants repository read permission only.
- PostgreSQL 17 and Redis 8 services use healthchecks. CI applies migrations through `cmd/migrate`, then runs tests, race tests, vet, Go build, and Docker build without a Discord webhook or repository secrets.
- Migration tests keep schema work inside a rolled-back transaction. The Redis queue integration test uses DB 14 so a locally running Horus worker on DB 0 cannot consume its jobs.

### Frontend Phase 9A

- `web/` uses Vite, React, TypeScript, Tailwind CSS, local shadcn/ui style primitives backed by Radix Dialog, React Router, and TanStack Query. `UI-CONTEXT.md` is the visual and UX reference; `UI-DESIGN.md` is not present.
- One `/monitors` screen has desktop sidebar and compact mobile navigation, a dense list of monitor configuration, a validated create dialog, direct enable/disable actions, and a named delete confirmation. Loading skeletons, empty-state action, page retry, and local mutation errors are implemented.
- The typed fetch client under `web/src/lib/api.ts` handles Horus JSON errors. Vite proxies same-origin `/api/*` requests to the local Go API with `/api` removed; `VITE_HORUS_API_URL` may select a separate origin when browser CORS permits it. No backend CORS or Compose changes were made.
- The list deliberately shows only enabled/disabled configuration state. Monitor health, latency, uptime, check history, incidents, and detail screens remain unimplemented in the frontend.

## Routes

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

## Known gaps and limitations

- No durable notification delivery, flapping suppression, or time-weighted uptime calculation; the reported percentage counts check outcomes.
- No user accounts, authentication, authorization, or monitor ownership.
- The checker accepts only absolute HTTP/HTTPS URLs without embedded credentials, rejects non-public DNS results at connection time, and does not follow redirects. Other SSRF edge cases should continue to be reviewed as the service is hardened.
- No check execution retries, duplicate-job protection, or queue recovery/dead-letter handling. Scheduling errors are retried on the next tick, and workers back off after dequeue errors. If enqueue succeeds but updating `next_check_at` fails, a later tick can enqueue the same monitor again.
- No metrics, structured logging, or production secret management. `/ready` checks dependency connectivity, not migration state, scheduler progress, worker activity, or queue delivery.
- Legacy schemas created manually before `schema_migrations` need an explicit baseline or fresh volume; migrations are not inferred from existing tables. Horus also requires PostgreSQL and Redis at startup and does not load `.env` automatically.
- Checker uses `http.DefaultClient`; status mismatch is checked against one exact expected status.
- Offset-based check history pages can shift when new checks arrive between requests.
- Offset-based incident history pages can shift when incidents open between requests.
- The frontend is a development app, not served by backend Compose. Production static hosting needs an equivalent `/api` reverse proxy or a separately CORS-enabled API origin. The monitor list does not show live check status.

## Verification record

The repository includes unit/API tests and PostgreSQL/Redis-backed repository and queue tests. Database tests require reachable local services and applied migrations. For the Redis cancellation and scheduler fixes, targeted scheduler/worker/queue tests, `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` passed with local services. A controlled run processed the reported overdue monitor through a failed check and an incident opening; its diagnostic data was then removed. An idle-worker process exited in under one second after Ctrl+C with real Redis. The original scheduling inactivity was not reproduced, so its historical cause remains unconfirmed.

Discord configuration investigation confirmed that the running process lacked `HORUS_DISCORD_WEBHOOK_URL` even though `.env` contained it. Configuration and startup-wiring tests now cover this boundary, and notification tests cover secret-safe request errors. Focused config/check/notification tests, `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` passed. A manual run with a temporary PostgreSQL database and isolated Redis reproduced disabled delivery, then verified one DOWN across three failed HTTP 500 checks and one RECOVERED across two successful HTTP 200 checks. A local observation relay forwarded the configured notifications to Discord and recorded HTTP 204 for each. Temporary services/data were removed. The application still requires settings to be exported before startup.

For liveness and readiness, focused server and Redis tests, `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` passed with local PostgreSQL and Redis. Tests cover all dependency success/failure combinations, safe 503 responses, liveness after Redis failure, a one-second handler deadline, and a Redis client ping against a stalled TCP listener.

For containerization and migrations, integration tests covered fresh, repeated, partial, and failing migrations. `go test ./...`, `go vet ./...`, `go build ./...`, `git diff --check`, and `docker compose build` passed. An isolated fresh Compose project applied four migrations, reached healthy PostgreSQL/Redis/Horus states, and persisted successful HTTPS checks for a created monitor. `docker compose stop horus` exited cleanly with code 0. After `docker compose down` and `up`, the migration job applied zero versions and the monitor remained available.

For CI, the local migration command applied four versions on an isolated fresh PostgreSQL service and zero on a repeated run. `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `git diff --check`, and `docker build .` passed with local services. The Redis queue race test passed three consecutive runs after moving its jobs to DB 14. The GitHub-hosted workflow has not yet been run.

For Phase 8A, monitor API and domain tests cover strict creation input, URL and numeric validation, safe missing and database-error responses, and all monitor mutation status codes. PostgreSQL integration tests verify affected-row behavior and distinguish missing monitors from database failures. `go test ./internal/api/...`, `go test ./internal/postgres/...`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` passed with local services.

For Phase 8B, API tests cover strict pagination, empty history and summary responses, missing and malformed monitor IDs, and safe repository errors. PostgreSQL integration tests cover empty, successful, failed, and mixed check sets; pagination and timestamp ties; nullable failure fields; latest status; and integer-millisecond average latency. `go test ./internal/api/...`, `go test ./internal/postgres/...`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` passed with local services.

For Phase 8C, API tests cover pagination defaults and invalid values, empty history and current responses, missing and malformed monitor IDs, safe dependency errors, and open/resolved duration behavior. PostgreSQL integration tests cover timestamp ties, stable pages, open-only current reads, nullable resolution, default failure context, and preservation of initial context. The existing incident handlers and SQL already satisfied these semantics, so no production code or schema changes were needed. `go test ./internal/api/...`, `go test ./internal/postgres/...`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` passed with local services.

For Phase 8D, focused handler tests cover JSON `400`, `404`, and generic `500` responses, content type, preservation of useful validation messages, and omission of sensitive repository text from both client responses and diagnostic logs. Existing readiness tests cover the deliberate `503` status response. `go test ./internal/api/...`, `go test ./cmd/server/...`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...` passed with local services. The initial sandboxed server test attempt could not open local sockets; the same test passed outside the sandbox.

For Phase 8E, config tests cover every default, invalid worker counts and ports, blank required fields, optional/invalid Discord URLs, and the rule that Go does not load `.env`. Server tests cover dependency-independent liveness, all readiness states, both stalled dependency positions, and safe Discord startup logs. Queue tests exercise real Redis cancellation; worker tests cover idle pool shutdown. Focused config, queue, worker, and server tests, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...` passed with local services. Direct server runs exited nonzero when either PostgreSQL or Redis used a closed local port. Compose returned `200` for both `/health` and `/ready`; `docker compose stop horus` produced exit code 0, and Horus was restarted.

For Phase 8F, the README was consolidated into a complete backend v1 entrypoint and checked against routes, configuration, migrations, Docker/Compose, CI, and tests. A source search found no completed TODO/FIXME comments to remove. `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `git diff --check`, and `docker build .` passed locally. An isolated `docker compose up --build -d` stack reached healthy PostgreSQL, Redis, and Horus states; its migration job exited 0. `/health` and `/ready` returned `200`. A temporary monitor returned `201` on creation and `200` on retrieval, persisted a successful check visible in history and summary, returned `200 []` for incident history and `204` for current incident, and returned `204` on deletion. The isolated containers and test volumes were removed. No blocking backend defect was found; the documented v1 limitations remain. The GitHub-hosted CI workflow has not been observed in this review.

For Phase 9A, frontend tests, TypeScript/production build, and ESLint passed in a Node 22 container. The final test run covered 15 tests across form validation and monitor-page interactions. The live Vite server served the `/monitors` page as HTML and proxied `/api/monitors` to the running Horus API. Headless Firefox screenshots were inspected at desktop and narrow widths; they captured the initial loading state. A manual click-through against the live backend is still pending.

## Next work

Phases 8A–8F and 9A are complete. Phase 9B should add a monitor detail screen using persisted check, summary, and incident APIs, with honest loading and empty states. Authentication/authorization, queue recovery, telemetry, and durable notification delivery remain future work.
