# Architecture

Horus is one Go process containing the HTTP API, a one-second scheduler, and a bounded worker pool. PostgreSQL stores monitors, checks, and incidents. Redis carries check jobs on `horus:checks`; it is not an acknowledged durable queue. The React app in `web/` calls the API through `/api` in both Vite development and the production Nginx image.

```text
Browser -> Nginx/Vite -> Go API -> PostgreSQL
                            |
                      scheduler (1s)
                            |
                      Redis job list
                            |
                       worker pool -> HTTP target
                            |
                      check + incident writes -> optional Discord webhook
```

## Package map

| Path | Responsibility |
| --- | --- |
| `cmd/server` | Small executable entry point; logs and exits nonzero if `server.Run` fails. |
| `internal/server` | Configuration, dependency wiring, routes, liveness/readiness, signal handling, HTTP listener, and shutdown. |
| `internal/config` | `HORUS_*` environment parsing and validation. The Go binaries do not load `.env`. |
| `internal/database` | PostgreSQL pool and Redis client construction. |
| `internal/monitor` | Monitor model/validation, API handler, and PostgreSQL repository. |
| `internal/probe` | HTTP GET engine, timeout and failure classification, redirect policy, IP dialing policy. |
| `internal/check` | Check model, application service, API handler, and PostgreSQL repository. The service coordinates probing, persistence, incident transitions, and notifications. |
| `internal/incident` | Incident model, API handler, and PostgreSQL read/transition repository. |
| `internal/httpresponse` | Shared JSON error and response-write helpers; monitor existence guard for check/incident routes. |
| `internal/queue` | Job payload and Redis LPUSH/BRPOP transport. |
| `internal/scheduler` | Due-monitor selection and queue submission. |
| `internal/worker` | Bounded queue consumers and dequeue-error backoff. |
| `internal/notification` | Optional Discord webhook transport. |
| `cmd/migrate`, `internal/migrate`, `migrations` | Separate transactional migration command and embedded SQL. |
| `web/src/features/monitors`, `web/src/lib/api.ts` | Typed frontend requests/query keys and shared fetch/error handling. |

Handlers use small interfaces near their consumers. Repository implementations live with their domains; `check.Service` receives separate check and incident repositories. `probe` depends on monitor input but does not depend on the check package. `internal/server` is the composition root. Monitor and incident handlers currently use their repositories directly because they have no additional application workflow; the check workflow has an explicit service.

## Runtime flow

`server.Run` validates configuration, connects to PostgreSQL and Redis, binds the HTTP address, then creates domain handlers and workers. It starts the scheduler and serves the mux. A bind or serve failure returns an error to `cmd/server`, which exits unsuccessfully. SIGINT/SIGTERM cancel the root context; workers and the scheduler join before the HTTP server receives its five-second graceful-shutdown deadline. The HTTP serve goroutine is also joined. Clients close on process exit. `/health` responds without dependency I/O. `/ready` pings PostgreSQL and Redis with a shared one-second deadline and returns a generic `503` on failure.

The scheduler lists monitors every second, skips disabled and not-yet-due entries, enqueues one `monitor_id` job with optional `scheduled_at`, then advances `next_check_at` by one interval. Enqueue and schedule-update errors are logged per monitor and do not stop later monitors in that pass. Redis uses LPUSH and a BRPOP with a bounded one-second wait. Workers load the monitor and discard the job if it is disabled at that point. Enabled monitors proceed through the probe, check insert, and incident transition. The check service sends Discord DOWN/RECOVERED only for new transitions when configured. A notification failure leaves persisted state intact and is logged by the worker.

The probe accepts only absolute HTTP/HTTPS URLs without credentials, does not follow redirects, and by default refuses non-public resolved IPs when dialing. `HORUS_ALLOW_PRIVATE_TARGETS=true` permits private targets for isolated trusted benchmarks. Check results retain `scheduled_at` and `started_at` through the job, probe, and PostgreSQL insert. Migration 005 adds those nullable columns. They support benchmark lag reports; API check history still returns the established JSON fields.

Monitor creation strictly decodes one JSON object, rejects unknown fields/trailing data, and validates stored-second range, URL, interval, timeout and expected status. Domain handlers preserve the documented API routes and status codes in the [README](../README.md#api). Check and incident histories use strict `limit`/`offset` validation and stable timestamp/ID ordering. An open incident is unique per monitor. Handler errors use `{"error":"..."}`, with generic `500` text and logs that omit underlying error contents.

## Persistence and deployment

Migrations 001-005 are embedded in `cmd/migrate` and applied in numeric order. Each migration and its version record commit in one PostgreSQL transaction. The server never migrates on startup. Compose waits for PostgreSQL and Redis health and migration success before starting Horus; web waits for Horus readiness. PostgreSQL and Redis use named volumes, and published ports bind to loopback by default. Nginx serves built React assets with SPA fallback and proxies `/api/*` to Horus. CI starts fresh PostgreSQL and Redis, migrates, runs Go tests/race/vet/build and frontend tests/lint/build, then builds both images.

The frontend routes are `/overview`, `/monitors`, and `/monitors/:id`. Overview derives configuration counts from `/monitors` and checks current incidents for at most 12 recent monitors with concurrency three. It labels incomplete coverage and shows an aggregate only when coverage is complete. Detail loads monitor metadata, summary, checks, current incident, and incident history independently. `204` current incident becomes `null` in the frontend. The list shows enabled/disabled configuration, while detail uses an open incident to indicate DOWN; absence of an open incident does not prove health. [UI guidance](ui-context.md) records the product's visual and status conventions.

## Reliability limits

Redis dequeue removes a job before result persistence, so a worker crash can lose it. Enqueue and schedule update are separate, so an update failure can enqueue a duplicate on a later tick; multiple Horus replicas also run multiple schedulers. Check insert and incident transition are separate operations, and notifications have no durable retry. Uptime is based on persisted check outcomes rather than elapsed availability. Readiness checks connectivity, not migration state or scheduler/worker progress. Authentication, rate limits, full SSRF protection, metrics, and a global incidents API are not implemented. These remain known limits for a future redesign.
