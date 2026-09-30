# Code Standards

These conventions describe the current Go code and preferred changes. Security and production features listed as future work should not be mistaken for protections already present.

## Go and packages

- Format Go code with `gofmt`; use idiomatic Go names and conventional acronym casing (`ID`, `URL`, `HTTP`, `API`).
- Keep packages cohesive by responsibility. Current code is organized under `internal/api`, `internal/check`, `internal/database`, `internal/incident`, `internal/monitor`, `internal/postgres`, `internal/queue`, `internal/scheduler`, and `internal/worker`.
- Keep SQL in `internal/postgres`, HTTP transport in API handlers, monitor domain behavior in `internal/monitor`, incident types in `internal/incident`, check execution and lifecycle coordination in `internal/check`, scheduling in `internal/scheduler`, and queue consumption in `internal/worker`.
- Define small interfaces near their consumers where that improves testability. Avoid speculative abstractions and dependencies.
- Pass `context.Context` to I/O operations, propagate cancellation, and handle errors explicitly with useful context where appropriate.
- Keep API request/response types explicit and separate from persistence/domain types. Use JSON field names and external units such as `interval_seconds`, `latency_ms`, and RFC3339 timestamps.

## API conventions

Current routes are:

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

Handlers decode and validate request-level input, call repository/application behavior, then encode responses and status codes. Do not put SQL, scheduling, or checker algorithms in handlers. Preserve existing status behavior unless deliberately changing the API contract: create returns `201`, delete and enable/disable return `204`, and list/retrieve/history/summary return JSON `200` responses.

## Persistence and reliability

- PostgreSQL repositories use parameterized queries and receive request/job context.
- Schema changes belong in ordered SQL files under `migrations/`; startup currently does not apply migrations.
- PostgreSQL holds durable monitor/check state; Redis currently carries jobs on `horus:checks`.
- Keep worker concurrency bounded and respect cancellation. Current process starts three workers.
- Test domain behavior and handler contracts with focused tests. Repository tests connect to a local PostgreSQL instance, so they require the database and applied migrations.

## Security status

Monitor URLs are user-controlled. The checker currently restricts schemes to HTTP/HTTPS, blocks non-public DNS results at connection time, and refuses redirects. Authentication, authorization, request limits, rate limiting, and complete SSRF protection are not implemented. Treat these as known gaps when extending the service; do not document them as existing guarantees.

## Change discipline

Inspect related source/tests first, make a focused change, format changed Go files, run appropriate verification when requested or needed, and review the diff. Do not claim verification that was not run. Keep commits logically scoped and do not rewrite history without explicit direction.
