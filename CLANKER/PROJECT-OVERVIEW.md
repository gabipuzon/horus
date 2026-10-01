# Project Overview

## Identity and purpose

Horus is a self-hosted uptime monitoring service written in Go. It stores HTTP monitor configurations, schedules checks, records their outcomes, and serves monitoring data through an HTTP/JSON API. The repository currently implements the core monitoring path; it is not yet a full Pingdom-style product.

## Implemented capabilities

### Frontend foundation

- `web/` contains a separate Vite/React/TypeScript monitor dashboard styled with Tailwind and local shadcn/ui style components, following `UI-CONTEXT.md`.
- React Router exposes the monitor list and `/monitors/:id`; TanStack Query and a typed fetch client load and mutate monitors. The list supports creation, enable/disable, confirmed deletion, and loading/empty/error feedback.
- The list only reports enabled/disabled configuration. Detail shows configuration, check-based summary, a recent-latency chart, first-page checks, current incident, and first-page incident history. An open incident is DOWN; absence of an open incident is not presented as proven health.

### Monitor management

- Create, list, retrieve, enable, disable, and delete monitors.
- Configure URL, check interval, request timeout, and one expected HTTP status code.
- Persist monitor state and scheduling time in PostgreSQL.
- Reject invalid monitor creation JSON and URL forms; distinguish missing monitor IDs from successful GET, delete, enable, and disable operations.

### Health checks and history

- Schedule due enabled monitors and enqueue check jobs in Redis.
- Run checks using a bounded worker pool (three by default).
- Record status code, latency, success, failure category, optional error text, and timestamp in PostgreSQL.
- Classify unsuccessful results as `http`, `network`, or `timeout`; successful results have an empty failure category.
- List checks newest first with strict limit/offset pagination, returning `[]` for an existing monitor without checks. Return aggregate count/latency/latest-status summaries with check-based uptime percentage; missing or malformed monitor IDs return `404` on both routes.

### Incidents

- Open one incident when a monitor fails, keep it open through repeated failures, and resolve it after a successful check.
- Store outage start, optional resolution, and initial failure context in PostgreSQL. Expose incident history newest first with stable ID tie-breaking and strict limit/offset pagination; an empty history is `[]`. The current route returns only an open incident, or an empty `204`. Durations are elapsed for open incidents and fixed for resolved incidents.
- Optionally send DOWN and RECOVERED Discord webhook messages when incidents open or resolve. Repeated failed checks and ordinary successful checks do not send notifications.

### Runtime

- Expose `/health` for process liveness and `/ready` for PostgreSQL and Redis connectivity readiness.
- Keep `/health` independent of dependencies; `/ready` uses a shared one-second deadline and returns `503` when either dependency is unavailable. Initial PostgreSQL and Redis checks are required before Horus starts serving.
- Handle interrupt/SIGTERM and shut down workers and the HTTP server; idle Redis dequeues use a bounded wait so cancellation can complete promptly.
- Configure optional Discord notifications with `HORUS_DISCORD_WEBHOOK_URL`; Horus starts normally when it is unset.
- Build a multi-stage, non-root Horus image with CA certificates. Compose starts PostgreSQL, Redis, migrations, then Horus; PostgreSQL data persists in a named volume.
- Run `cmd/migrate` to apply pending numbered SQL migrations transactionally; successful versions are recorded in `schema_migrations`.
- Use GitHub Actions CI to apply migrations against fresh PostgreSQL and Redis services, then run Go tests, race tests, vet, Go build, and Docker build on pushes and pull requests.
- Export settings into the process environment before starting Go directly; the binaries do not load `.env` automatically. Compose reads `.env` for substitution. Startup logs whether Discord is enabled, without printing its webhook URL. Only new incident transitions notify; existing incidents are not replayed after enabling notifications.
- Direct-run defaults use local PostgreSQL/Redis, HTTP `:8080`, and three workers. Config validates required connection fields, ports, worker count (1–1000), and nonblank Discord URLs; a blank webhook disables notifications.

## API

| Method | Path | Behavior |
|---|---|---|
| `GET` | `/health` | Returns `{"status":"ok"}` |
| `GET` | `/ready` | Returns `{"status":"ready"}` when PostgreSQL and Redis respond within one second, otherwise `503` with `{"status":"not_ready"}` |
| `POST` | `/monitors` | Creates a monitor; returns `201` and its representation |
| `GET` | `/monitors` | Lists monitors |
| `GET` | `/monitors/{id}` | Retrieves a monitor |
| `DELETE` | `/monitors/{id}` | Deletes a monitor; returns `204` |
| `PATCH` | `/monitors/{id}/enable` | Enables a monitor; returns `204` |
| `PATCH` | `/monitors/{id}/disable` | Disables a monitor; returns `204` |
| `GET` | `/monitors/{id}/checks` | Lists newest checks; `limit` defaults to 50 and must be 1–100, `offset` defaults to 0 and must be nonnegative; invalid values return `400` |
| `GET` | `/monitors/{id}/summary` | Returns check counts, average latency in whole milliseconds, latest status code, and check-based uptime percentage |
| `GET` | `/monitors/{id}/incidents` | Lists incidents newest first; `limit` defaults to 50 and must be 1–100, `offset` defaults to 0 and must be nonnegative; invalid values return `400` |
| `GET` | `/monitors/{id}/incidents/current` | Returns only the open incident, or an empty `204` when none is open |

Monitor JSON fields are `id`, `name`, `url`, `interval_seconds`, `timeout_seconds`, `expected_status`, and `enabled`. Check rows include `id`, `monitor_id`, `status_code`, `latency_ms`, `success`, `failure_type`, optional `error`, and RFC3339 `checked_at`; 0 status means no HTTP response. Summary fields are `total_checks`, `successful_checks`, `failed_checks`, `average_latency_ms`, `latest_status`, and `uptime_percentage`. The percentage is `successful_checks / total_checks × 100`, rounded to two decimals; it is `null` with no checks and is not time weighted. The average latency is truncated to whole milliseconds, and latest status 0 means no HTTP response on the newest check or no checks. Incident rows include start and optional resolution timestamps, `is_open`, `duration_ms`, and initial failure context. Open duration is elapsed at response time; resolved duration ends at `resolved_at`. Missing or malformed monitor IDs return `404` on check and incident data routes.

Horus handler errors use JSON `{"error":"..."}`: invalid requests return `400`, missing or malformed monitor IDs return `404`, and unexpected internal failures return a generic `500`. `/ready` retains its `503 {"status":"not_ready"}` dependency response. Unsupported routes and methods use Go's default mux responses.

## Not implemented yet

Durable notification delivery and retries, authentication/ownership, time-weighted uptime, check execution retries, metrics, structured logging, production configuration, and complete SSRF defenses are future work. `/ready` checks connection health, not migration state or actual scheduler and worker progress. A Discord failure does not undo a persisted check or incident transition; the error is reported to the worker and that notification may be missed. The scheduler retries scheduling errors on its next tick, and workers back off after dequeue errors. Runtime connection/listen settings and worker count can be set through environment variables. The monitor constructor checks that name/URL are nonempty, interval/timeout are positive, and expected status is in the HTTP status range; URL safety checks run in the checker before dialing.

The backend v1 contract has been documented and locally verified for freeze readiness. Phases 9A–9B establish the frontend monitor list and detail. Overview and global incident screens remain future frontend work.

## Principles and non-goals

- Keep the service a modular monolith while that meets current needs.
- PostgreSQL is durable application state; Redis currently carries check jobs.
- Keep concurrency bounded and propagate contexts through I/O.
- Prefer small, testable changes that match the existing package layout.
- Do not expand into browser automation, an enterprise monitoring suite, or microservices without a demonstrated need.
