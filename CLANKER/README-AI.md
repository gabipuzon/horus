# AI Context for Horus

Use these files as a concise guide to the current repository. Source code and tests define behavior; update documentation when implementation changes. The user may explicitly request edits to any file in this folder.

| File | Purpose |
|---|---|
| `PROJECT-OVERVIEW.md` | Product purpose, implemented capabilities, and future scope |
| `ARCHITECTURE.md` | Implemented components, data flow, persistence, and runtime wiring |
| `CODE-STANDARDS.md` | Existing Go and API conventions |
| `AI-WORKFLOW.md` | Repository workflow and agent behavior |
| `PROGRESS.md` | Snapshot of implemented work and known gaps |
| `UI-CONTEXT.md` | Authoritative frontend visual and UX guidance |

## Working with this context

1. Read the relevant context files and inspect source/tests before making code changes.
2. Treat repository code and tests as authoritative when context is stale.
3. Keep documentation aligned with observable behavior; distinguish implemented features from planned work.
4. Do not claim tests, migrations, services, or operational checks were run unless they were.
5. Preserve existing package boundaries and avoid speculative infrastructure.

## Current implementation at a glance

Horus is a Go HTTP API backed by PostgreSQL. A one-second scheduler enqueues due monitor checks in Redis, keeps running after scheduling errors, and continues to later monitors when one enqueue or schedule update fails. Three workers by default consume jobs, execute HTTP checks, persist results, and update incident state in PostgreSQL. Redis dequeue uses a bounded blocking wait so idle workers can exit after cancellation. Incident openings and resolutions can send Discord webhook notifications when `HORUS_DISCORD_WEBHOOK_URL` is configured. The API exposes monitor management, check history and summaries, incident history and current state. `GET /health` reports HTTP process liveness; `GET /ready` checks PostgreSQL and Redis connectivity with a one-second deadline. A multi-stage image runs the server as a non-root user. Compose starts PostgreSQL, Redis, a one-shot migration job, and Horus in dependency order, with PostgreSQL data on a named volume.

The check summary includes a check-based uptime percentage: successful persisted checks divided by all persisted checks, rounded to two decimals, or `null` when no checks exist. Discord notifications are optional and sent once per persisted incident transition; delivery failures leave check and incident state intact and are reported to the worker. Durable notification delivery, users/authentication, metrics, and complete SSRF protections are not implemented. The checker includes baseline URL restrictions and the application supports environment-based runtime configuration. See `PROGRESS.md` for the current detail.

Configuration comes from the process environment; the Go binaries do not load `.env` automatically. Compose reads `.env` for substitution and supplies container-specific hostnames. For direct Go runs, export local settings before launching and run `go run ./cmd/migrate` to apply pending embedded SQL migrations (see the root README). `schema_migrations` records completed versions. Startup logs whether Discord notifications are enabled without printing the webhook URL. Notification request errors use safe messages, and enabling notifications does not replay existing incidents.

Runtime configuration defaults to localhost PostgreSQL/Redis, `:8080`, and three workers for direct development. Required non-password connection fields and the HTTP address cannot be blank; ports must be 1–65535 and worker count 1–1000. An empty Discord URL disables notifications; a nonempty URL requires an HTTP(S) scheme, hostname, and no embedded credentials. The server exits if its initial PostgreSQL or Redis check fails. Shutdown cancels scheduler/workers, joins workers, then gives HTTP shutdown five seconds. `/health` remains independent of dependencies, while `/ready` uses a shared one-second PostgreSQL/Redis deadline.

One GitHub Actions CI workflow runs on pushes to `main` and pull requests targeting it. It starts PostgreSQL and Redis services, applies migrations with `cmd/migrate`, then runs tests (including the race detector), vet, Go build, and Docker build. CI uses local credentials and no Discord webhook or repository secrets.

Monitor creation now rejects malformed or unknown JSON fields, trailing data, blank names, invalid HTTP/HTTPS URL forms, nonpositive intervals/timeouts, values too large for stored seconds, and invalid expected status codes. Monitor GET, DELETE, enable, and disable return `404` for missing or malformed UUIDs. Successful create remains `201`, reads return `200`, and delete/enable/disable return `204`.

Check history uses strict `limit` (default 50, range 1–100) and `offset` (default 0, nonnegative) parsing; invalid values return `400`. Existing monitors with no checks return `[]`. Check history and summary return `404` for missing or malformed UUIDs. History is newest first, with check ID breaking equal timestamps. Summary averages persisted latency in whole milliseconds and reports the newest check's HTTP status (0 if none); with no checks it returns zero fields and null uptime.

Incident history uses the same strict pagination policy, returns `[]` for an existing monitor without incidents, and orders by `started_at DESC, id DESC`. The current route returns only the open incident or an empty `204` when none exists. Both incident routes return `404` for missing or malformed monitor IDs. Open `duration_ms` measures elapsed time at response; resolved duration is fixed from start to resolution, and negative values clamp to zero. The incident schema allows a null resolution time; initial failure type, status code, and message are nonnullable, with status 0 and empty message defaults.

Handler-produced API errors use JSON `{"error":"..."}` with `Content-Type: application/json`: request validation is `400`, missing or malformed monitor IDs are `404`, and internal failures are generic `500` responses. Internal failure logs identify the operation and error type without copying error text that could contain secrets. `/ready` keeps its operational `503 {"status":"not_ready"}` response; unsupported paths and methods use the standard Go mux behavior.

Backend v1 documentation now lives in the root README. Local tests, race tests, vet, Go build, Docker build, and an isolated Compose API smoke flow passed for the Phase 8F review. No blocking backend defect was found; limitations such as check-based uptime, non-durable Redis jobs and Discord delivery, missing authentication, and connectivity-only readiness are documented. The backend contract is ready to freeze.

Phase 9 adds `web/`: Vite, React, TypeScript, Tailwind CSS, local shadcn/ui style primitives with Radix Dialog, React Router, and TanStack Query. `/` redirects to `/overview`; `/monitors` manages configuration; `/monitors/:id` loads metadata, summary, first 50 checks, current incident, first 50 incidents, and a response-time chart using independent queries. Overview reads `/monitors` for honest configuration counts and a compact list. It checks current incidents for at most the 12 newest monitors with concurrency three; partial coverage is labeled and no global incident count is claimed. Enabled is configuration, while an open incident indicates DOWN. No active incident does not imply healthy. `web/src/lib/api.ts` owns fetch and safe API errors; Vite proxies same-origin `/api/*` to Horus with the prefix removed. `VITE_HORUS_API_URL` optionally selects a separate API origin when browser CORS permits it. `UI-CONTEXT.md` is the visual reference. A global Incidents screen is not implemented.
