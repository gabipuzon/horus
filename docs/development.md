# Development guide

The [README](../README.md) is the operator entry point. [Architecture](architecture.md) describes current package ownership and runtime flow; [UI guidance](ui-context.md) records frontend conventions.

## Go changes

Keep the command entry points small. Wire dependencies in `internal/server`; place monitor, check, and incident models, HTTP handlers, and PostgreSQL repositories in their domain packages. Put HTTP target logic in `internal/probe`, Redis transport in `internal/queue`, and connection setup in `internal/database`. Define small interfaces near consumers. Pass contexts through I/O and return explicit errors. Format changed Go files with `gofmt`.

Preserve the API contract unless a change is deliberate: monitor create returns `201`; delete/enable/disable and absent current incident return `204`; missing or malformed monitor IDs return `404`; invalid input returns `400`; unexpected failures return generic JSON `500`. Do not put raw repository error contents into client responses or logs. Keep persistence migrations numbered and transactional. The server does not apply them. Preserve the `horus:checks` key and existing `monitor_id` job field when extending queue payloads.

The Go binaries read only process environment. For a direct run, export settings, apply migrations with `go run ./cmd/migrate`, then run `go run ./cmd/server`. `HORUS_ALLOW_PRIVATE_TARGETS` changes the probe's IP policy and is intended for isolated trusted tests only.

`go test ./...` runs local unit and HTTP tests without external PostgreSQL or Redis. Tests that need those services skip unless `HORUS_TEST_INTEGRATION=1` is set. For a full local run, start both services, apply all migrations, then run `HORUS_TEST_INTEGRATION=1 go test ./...`. Integration tests use localhost ports 5432/6379 and `horus:horus` credentials by default; `HORUS_DB_HOST`, `HORUS_DB_PORT`, `HORUS_DB_USER`, `HORUS_DB_PASSWORD`, `HORUS_DB_NAME`, `HORUS_REDIS_HOST`, and `HORUS_REDIS_PORT` select an isolated stack. Migration tests use rolled-back schemas; queue tests use Redis DBs 14 and 15. CI sets the integration flag and provides fresh services. Run focused tests after a domain change, then Go tests, race tests, vet, build, frontend checks, and image build before declaring a structural migration complete.

## Frontend changes

The frontend uses React Router, TanStack Query, a typed API client at `web/src/lib/api.ts`, and monitor-specific types/actions under `web/src/features/monitors`. Keep server state in TanStack Query and dialog/form state local. Detail queries remain independent so one failed section does not hide others. A `204` current incident maps to `null`. Overview configuration counts come from `/monitors`; its bounded incident scan must label partial coverage. Enabled is configuration, not a health result. Use [UI guidance](ui-context.md) for status language, density, contrast, accessibility, and responsive behavior.

From `web/`, run `npm test`, `npm run lint`, and `npm run build`. The production Nginx image uses same-origin `/api` proxying and SPA fallback. Keep browser flows tied to implemented API behavior.
