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

Horus is a Go HTTP API backed by PostgreSQL. A one-second scheduler enqueues due monitor checks in Redis; three workers consume jobs, execute HTTP checks, and persist results to PostgreSQL. The API exposes monitor management, check history, summaries, and `/health`.

Incidents, notifications, users/authentication, metrics, and SSRF protections are not implemented yet. See `PROGRESS.md` for the current detail.
