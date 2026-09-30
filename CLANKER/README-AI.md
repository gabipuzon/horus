# AI Context for Horus

Use these files as a concise guide to the current repository. Source code and tests define behavior; update documentation when implementation changes. The user may explicitly request edits to any file in this folder.

| File | Purpose |
|---|---|
| `PROJECT-OVERVIEW.md` | Product purpose, implemented capabilities, and future scope |
| `ARCHITECTURE.md` | Implemented components, data flow, persistence, and runtime wiring |
| `CODE-STANDARDS.md` | Existing Go and API conventions |
| `AI-WORKFLOW.md` | Repository workflow and agent behavior |
| `PROGRESS.md` | Snapshot of implemented work and known gaps |

## Working with this context

1. Read the relevant context files and inspect source/tests before making code changes.
2. Treat repository code and tests as authoritative when context is stale.
3. Keep documentation aligned with observable behavior; distinguish implemented features from planned work.
4. Do not claim tests, migrations, services, or operational checks were run unless they were.
5. Preserve existing package boundaries and avoid speculative infrastructure.

## Current implementation at a glance

Horus is a Go HTTP API backed by PostgreSQL. A one-second scheduler enqueues due monitor checks in Redis, keeps running after scheduling errors, and continues to later monitors when one enqueue or schedule update fails. Three workers consume jobs, execute HTTP checks, persist results, and update incident state in PostgreSQL. Redis dequeue uses a bounded blocking wait so idle workers can exit after cancellation. Incident openings and resolutions can send Discord webhook notifications when `HORUS_DISCORD_WEBHOOK_URL` is configured. The API exposes monitor management, check history and summaries, incident history and current state. `GET /health` reports HTTP process liveness; `GET /ready` checks PostgreSQL and Redis connectivity with a one-second deadline. A multi-stage image runs the server as a non-root user. Compose starts PostgreSQL, Redis, a one-shot migration job, and Horus in dependency order, with PostgreSQL data on a named volume.

The check summary includes a check-based uptime percentage: successful persisted checks divided by all persisted checks, rounded to two decimals, or `null` when no checks exist. Discord notifications are optional and sent once per persisted incident transition; delivery failures leave check and incident state intact and are reported to the worker. Durable notification delivery, users/authentication, metrics, and complete SSRF protections are not implemented. The checker includes baseline URL restrictions and the application supports environment-based runtime configuration. See `PROGRESS.md` for the current detail.

Configuration comes from the process environment; the Go binaries do not load `.env` automatically. Compose reads `.env` for substitution and supplies container-specific hostnames. For direct Go runs, export local settings before launching and run `go run ./cmd/migrate` to apply pending embedded SQL migrations (see the root README). `schema_migrations` records completed versions. Startup logs whether Discord notifications are enabled without printing the webhook URL. Notification request errors use safe messages, and enabling notifications does not replay existing incidents.

One GitHub Actions CI workflow runs on pushes to `main` and pull requests targeting it. It starts PostgreSQL and Redis services, applies migrations with `cmd/migrate`, then runs tests (including the race detector), vet, Go build, and Docker build. CI uses local credentials and no Discord webhook or repository secrets.

Check history uses strict `limit` (default 50, range 1–100) and `offset` (default 0, nonnegative) parsing; invalid values return `400`. Existing monitors with no checks return `[]`. Check history and summary return `404` for missing or malformed UUIDs. History is newest first, with check ID breaking equal timestamps. Summary averages persisted latency in whole milliseconds and reports the newest check's HTTP status (0 if none); with no checks it returns zero fields and null uptime.
