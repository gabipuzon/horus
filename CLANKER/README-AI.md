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

Horus is a Go HTTP API backed by PostgreSQL. A one-second scheduler enqueues due monitor checks in Redis, keeps running after scheduling errors, and continues to later monitors when one enqueue or schedule update fails. Three workers consume jobs, execute HTTP checks, persist results, and update incident state in PostgreSQL. Redis dequeue uses a bounded blocking wait so idle workers can exit after cancellation. Incident openings and resolutions can send Discord webhook notifications when `HORUS_DISCORD_WEBHOOK_URL` is configured. The API exposes monitor management, check history and summaries, incident history and current state, and `/health`.

The check summary includes a check-based uptime percentage: successful persisted checks divided by all persisted checks, rounded to two decimals, or `null` when no checks exist. Discord notifications are optional and sent once per persisted incident transition; delivery failures leave check and incident state intact and are reported to the worker. Durable notification delivery, users/authentication, metrics, and complete SSRF protections are not implemented. The checker includes baseline URL restrictions and the application supports environment-based runtime configuration. See `PROGRESS.md` for the current detail.
