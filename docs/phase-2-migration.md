# Phase 2: Server and domain migration

**Status:** Complete (2026-11-18)

## Goal

Make startup flow through `internal/server` and organize behavior by domain, following the reviewed phase 1 audit.

## Intended structure

- Keep the command entry point small: it calls `server.Run`, which owns configuration, dependency wiring, lifecycle, HTTP server startup, and shutdown. Use the local `gowatch/internal/server` example as the pattern. Return HTTP listen failures so the command exits unsuccessfully.
- Put PostgreSQL and Redis connection setup in `internal/database`. Keep queue operations in `internal/queue`.
- Give `internal/monitor`, `internal/check`, and `internal/incident` clear model, repository, and handler ownership. The check workflow has an application service; monitor and incident handlers use their repositories directly because they have no separate workflow to coordinate. Use `internal/probe` for the HTTP URL checking engine so `probe` and `check` have distinct names.
- Move routes and shared HTTP concerns out of the current central `internal/api` package, and move domain repositories out of the current central `internal/postgres` package.
- Remove tracked `CLANKER/` documentation after its useful information has been captured in maintained docs.

## Compatibility

Preserve existing routes and JSON, environment variables, Redis job payloads, and PostgreSQL schema unless a reviewed audit finding calls for an explicit change. Include the current uncommitted and untracked work in the migration baseline; do not discard it.

## Migration order

1. Preserve the current working tree and freeze existing contracts in tests, including the timing fields and Redis job format. Apply `migrations/005_add_check_timing.sql` in integration environments before testing check writes.
2. Add `internal/server` and move wiring, routes, health endpoints, and lifecycle out of `cmd/server`. Make `server.Run` return HTTP listen failures and verify the process exits unsuccessfully. Keep `cmd/migrate` independent.
3. Extract the HTTP probing engine and address policy into `internal/probe`, with types that avoid a dependency cycle with `internal/check`.
4. Move monitor, check, and incident behavior into their domain packages one at a time, including handlers and repositories. Keep the check workflow in its service, shared HTTP response behavior small, and consumer-side interfaces where needed.
5. Move Redis connection construction into `internal/database` while keeping queue encoding and operations in `internal/queue`. Preserve the queue key and job JSON. Remove superseded `internal/api` and `internal/postgres` packages after their callers and tests move.
6. Update maintained documentation, capture useful guidance from `CLANKER/`, then remove its tracked files. Run focused checks after each slice and the full Go, frontend, and image checks with configured services at the end.

## Complete when

The server starts and shuts down through `internal/server`, HTTP listen failures produce an unsuccessful process exit, each domain has a clear home, obsolete package boundaries are gone, and relevant Go, frontend, and build checks pass with required services available.

## Implementation and verification

- `cmd/server` now delegates to `server.Run`; startup, routes, health/readiness, and lifecycle live in `internal/server`. Binding happens before workers start. A controlled occupied-port run returned nonzero; normal startup served `/health` and `/ready` and exited zero on SIGTERM. The referenced local `gowatch/internal/server` example was unavailable in this workspace, so the implementation followed the ownership and lifecycle requirements directly.
- The HTTP checker moved to `internal/probe`. Monitor, check, and incident handlers and PostgreSQL repositories moved to their domain packages. Incident transitions now live with the incident repository; `check.Service` receives separate check and incident interfaces. Shared HTTP response behavior lives in `internal/httpresponse`.
- Redis client construction moved to `internal/database`; `internal/queue` retains job JSON and Redis list operations. The queue key, existing job field, routes, JSON, and schema were preserved. Migration 005 and the benchmark timing path were included in the baseline.
- Maintained architecture, development, and UI guidance now live in `docs/`; the tracked `CLANKER/` folder was removed after capture. Existing local work in `bench/` and `.nami/` was preserved.
- An isolated PostgreSQL 17 and Redis 8 stack on ports 55432/56379 applied all five migrations. `go test ./...`, the backend race suite, `go vet`, `go build`, `git diff --check`, 31 frontend tests, frontend lint/build, and `docker compose build` passed. Frontend checks ran in Node 22 Alpine because the existing host `node_modules` contains Alpine Rollup binaries while the host is Linux glibc/Node 26.

Phase 3 still owns disabled queued jobs and opt-in local integration tests. Larger queue and incident consistency changes remain deferred as described in the Phase 1 audit.
