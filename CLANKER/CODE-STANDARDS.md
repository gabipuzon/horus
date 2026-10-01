# Code Standards

## Frontend conventions

- Keep the frontend in `web/`, separate from Go. Follow `UI-CONTEXT.md` for status language, density, colors, and accessible interaction.
- Use React Router for implemented routes, TanStack Query for server state, and local React state for dialogs/forms. Avoid a global state store.
- Keep fetch and JSON error handling in `web/src/lib/api.ts`; put monitor-specific types, requests, validation, and components under `web/src/features/monitors/`.
- Keep detail queries keyed by monitor ID and resource. Use `null` as the frontend query result for `204` current incident; TanStack Query data cannot be `undefined`. Keep time formatting in `web/src/lib/time.ts`.
- Keep Overview counts derived from `/monitors`. Bound any per-monitor current-incident scan (currently 12 monitors, concurrency three), label partial coverage, and show an active-incident count only when every monitor was checked successfully. Do not infer health from the enabled flag or add a global incident endpoint for the frontend.
- Only render data the API actually supplies. Enabled/disabled is configuration, not an uptime or health result. Use labeled inputs, focus-visible styles, and Radix-backed dialogs.
- Validate with `npm test`, `npm run build` (including TypeScript), and `npm run lint` from `web/`.
- Production frontend changes belong in `web/Dockerfile` and `web/nginx.conf`: build assets in Node, serve through Nginx, preserve SPA fallback, and strip `/api` when proxying to Horus. Keep normal Compose browser traffic same-origin; do not require CORS. Compose host ports bind to loopback by default.

These conventions describe the current Go code and preferred changes. Security and production features listed as future work should not be mistaken for protections already present.

## Go and packages

- Format Go code with `gofmt`; use idiomatic Go names and conventional acronym casing (`ID`, `URL`, `HTTP`, `API`).
- Keep packages cohesive by responsibility. Current code is organized under `internal/api`, `internal/check`, `internal/config`, `internal/database`, `internal/incident`, `internal/migrate`, `internal/monitor`, `internal/notification`, `internal/postgres`, `internal/queue`, `internal/scheduler`, and `internal/worker`.
- Keep SQL in `internal/postgres`, HTTP transport in API handlers, monitor domain behavior in `internal/monitor`, incident types in `internal/incident`, Discord HTTP transport in `internal/notification`, check execution and lifecycle coordination in `internal/check`, scheduling in `internal/scheduler`, and queue consumption in `internal/worker`.
- Define small interfaces near their consumers where that improves testability. Avoid speculative abstractions and dependencies.
- Pass `context.Context` to I/O operations, propagate cancellation, and handle errors explicitly with useful context where appropriate.
- Keep API request/response types explicit and separate from persistence/domain types. Use JSON field names and external units such as `interval_seconds`, `latency_ms`, and RFC3339 timestamps.

## API conventions

Current routes are:

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

Handlers decode and validate request-level input, call repository/application behavior, then encode responses and status codes. Do not put SQL, scheduling, or checker algorithms in handlers. Preserve existing status behavior unless deliberately changing the API contract: create returns `201`, delete and enable/disable return `204`, and list/retrieve/history/summary return JSON `200` responses.
Monitor creation uses strict one-value JSON decoding. Monitor GET, delete, and enable/disable map `monitor.ErrNotFound` to `404`; unexpected repository errors remain safe `500` responses. PostgreSQL monitor mutations use `RowsAffected` rather than a separate existence query.
Check history pagination uses `limit` 1–100 (default 50) and nonnegative `offset` (default 0), rejecting malformed values with `400`. Check history and summary return `404` for missing or malformed monitor IDs; an existing monitor with no checks returns `[]` for history and zero summary fields with null uptime.
Incident history follows the same strict limit/offset rules and returns `[]` for an existing monitor without incidents. The current incident route returns an empty `204` when no open incident exists. Incident history orders equal start times by ID descending; open duration is elapsed at response time and resolved duration is fixed.
Handler-produced errors use the shared `writeError` helper: `application/json` with one `error` string. Preserve useful safe `400` messages and the `404` monitor message; return `500 {"error":"internal server error"}` for unexpected failures. Log the operation and error type without copying possibly sensitive repository error text. `/ready` keeps its separate `503 {"status":"not_ready"}` contract, and unsupported routes/methods use Go mux defaults. If response writing fails after headers are sent, log the failure rather than appending a second error body.

## Persistence and reliability

- Read runtime settings from `HORUS_*` process environment variables in `internal/config`; Go binaries do not load `.env`. Keep direct-run defaults local, validate blank required fields, ports, worker count, and configured Discord URLs without echoing secret values in errors or startup state logs. The server requires PostgreSQL and Redis during startup. Keep `/health` free of dependency checks and `/ready` bounded by its shared one-second context deadline.
- PostgreSQL repositories use parameterized queries and receive request/job context.
- Schema changes belong in ordered SQL files under `migrations/`. `cmd/migrate` embeds and applies pending versions transactionally; Compose runs it before the server. The server does not migrate at startup.
- PostgreSQL holds durable monitor, check, and incident state; Redis currently carries jobs on `horus:checks`. Discord notifications are optional and sent after incident transitions. Delivery errors must not undo persisted state.
- Keep worker concurrency bounded and respect cancellation. Current process starts three workers by default.
- Test domain behavior and handler contracts with focused tests. Repository tests connect to a local PostgreSQL instance, so they require the database and applied migrations.
- Keep integration tests isolated from live application data. Migration tests use their own rolled-back schema; Redis queue tests use a separate test DB. CI starts real PostgreSQL and Redis services and runs migrations before tests.

## Security status

Monitor URLs are user-controlled. The checker currently restricts schemes to HTTP/HTTPS, blocks non-public DNS results at connection time, and refuses redirects. Authentication, authorization, request limits, rate limiting, and complete SSRF protection are not implemented. Treat these as known gaps when extending the service; do not document them as existing guarantees.

## Change discipline

Inspect related source/tests first, make a focused change, format changed Go files, run appropriate verification when requested or needed, and review the diff. Do not claim verification that was not run. Keep commits logically scoped and do not rewrite history without explicit direction.
