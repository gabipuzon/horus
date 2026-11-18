# Phase 1: Architecture audit

**Status:** Reviewed (2026-11-18)

## Scope and working tree

This audit describes the current working tree, including unfinished local work. It makes no package moves, behavior fixes, or removals. The committed baseline has 104 tracked files. Before this audit edit, 14 tracked files were modified and 14 files were untracked. None were discarded.

| Local work | Inventory and significance |
| --- | --- |
| Modified tracked files | `README.md`, `cmd/server/main.go`, `internal/check/{checker,model,service}.go`, `internal/config/{config,config_test}.go`, `internal/migrate/migrate_test.go`, `internal/postgres/{check_repository,check_repository_test}.go`, `internal/queue/{check_job,check_job_test}.go`, `internal/scheduler/scheduler.go`, `internal/worker/pool.go`. These add a private-target switch for isolated benchmarks and carry scheduled/start times from job to persisted check. |
| Untracked migration | `migrations/005_add_check_timing.sql` adds nullable `started_at` and `scheduled_at`. Modified migration tests expect version 5. The check repository writes these columns; API history still returns the existing fields. This working tree requires migration 005 before writing checks. |
| Untracked benchmark | Nine files under `bench/`: isolated Compose stack, mock target, seed/report SQL, runner, instructions, and dated results. Results are evidence for one host, not a production capacity guarantee. |
| Untracked local data | `.nami/scans.db` is a 1.2 MB local scan database. Its contents were not inspected. Preserve until its owner decides how to handle it. |
| Untracked planning docs | `docs/phase-1-audit.md`, `docs/phase-2-migration.md`, and `docs/phase-3-fixes.md` define this staged work. |

`CLANKER/` is tracked in the committed baseline. It is useful historical context, but source and tests are more current. For example, `CLANKER/PROGRESS.md` says detail screens remain unimplemented under “Frontend Phase 9A”; `web/src/app/app.tsx` and its later Phase 9/10 sections show they exist. Its architecture snapshot does not include the local timing migration and benchmark. Preserve useful guidance in maintained docs before any Phase 2 removal.

## Current architecture and flow

```text
Browser -> Nginx/Vite /api proxy -> Go ServeMux in cmd/server
                                  -> internal/api handlers
                                  -> internal/postgres repositories -> PostgreSQL

cmd/server -> scheduler -> PostgreSQL monitor list -> Redis horus:checks
           -> worker pool -> PostgreSQL monitor lookup -> check service
              -> checker HTTP GET -> PostgreSQL check insert
              -> incident open/resolve -> optional Discord notification
```

| Area | Current owner and dependency path |
| --- | --- |
| Startup and lifecycle | `cmd/server/main.go` loads `internal/config`, opens the `internal/database` PostgreSQL pool and `internal/queue` Redis client, wires repositories, scheduler, worker pool, notifier and routes, then starts HTTP. SIGINT/SIGTERM cancel the context; workers are joined and HTTP gets a five-second shutdown deadline. Scheduler stops on cancellation but is not explicitly joined. `/health` is process liveness; `/ready` pings both dependencies with a shared one-second deadline. |
| HTTP and monitor management | `internal/api/{monitor,check,incident}_handler.go` owns request decoding, response types and errors. Monitor validation/model lives in `internal/monitor/monitor.go`; SQL create/list/get/delete/enable/schedule updates live in `internal/postgres/monitor_repository.go`. `cmd/server/main.go:156-174` registers 12 routes. |
| Scheduling and queue | `internal/scheduler/scheduler.go:65-113` scans all monitors every second, skips disabled monitors, enqueues one due job per monitor, then advances `next_check_at` by one interval. `internal/queue/redis.go` serializes jobs to Redis list `horus:checks` using LPUSH/BRPOP. The local job format adds optional `scheduled_at` while preserving `monitor_id`. |
| Probing and checks | `internal/worker/pool.go:80-94` loads each job's monitor and calls `check.Service.CheckScheduled`. `internal/check/checker.go` performs bounded HTTP GET, expected-status comparison, failure classification, redirect refusal and public-IP dialing by default. The local private-target option uses `HORUS_ALLOW_PRIVATE_TARGETS`. `internal/postgres/check_repository.go` writes outcomes and nullable timing fields, and reads history and summary. |
| Incidents and notification | `internal/check/service.go:48-91` inserts a check, then opens/resolves an incident through `postgres.CheckRepository`; `internal/incident/incident.go` defines the type and `postgres.IncidentRepository` reads history/current state. `internal/notification/discord.go` optionally sends transition notifications afterward. These are separate operations. |
| Frontend | `web/src/app/app.tsx` owns `/overview`, `/monitors` and `/monitors/:id`. `web/src/features/monitors/api.ts` defines typed requests and query keys; `web/src/lib/api.ts` performs fetch/error handling. Vite and Nginx proxy `/api` to Go. TanStack Query owns server data. |
| Build and operations | `internal/migrate` applies embedded numbered SQL transactionally; `cmd/migrate` is separate from server startup. `compose.yaml` starts PostgreSQL, Redis, migration job, Horus and web in dependency order. `Dockerfile` builds Go binaries; `web/Dockerfile` builds/serves the frontend. `.github/workflows/ci.yml` migrates a fresh database, then runs Go tests/race/vet/build, frontend tests/lint/build and image builds. |

### Package ownership today

`internal/config` and `internal/database` own settings and PostgreSQL connection setup. `internal/queue` owns Redis connection setup and job transport. `internal/api` combines handlers for three domains; `internal/postgres` combines their SQL. `CheckRepository` writes incidents while `IncidentRepository` reads them. `internal/check` combines HTTP probing with check/incident coordination. `internal/monitor` and `internal/incident` mostly contain models. `internal/scheduler`, `internal/worker`, `internal/notification` and `internal/migrate` have narrower roles. There is no `internal/server` or `internal/probe` yet.

## Prioritized findings

| Priority | Kind | Finding and evidence | Decision |
| --- | --- | --- | --- |
| P1 | Demonstrated code-path defect | An HTTP listen failure is logged in `cmd/server/main.go:184-189`, calls `stop()`, and the normal path returns from `main` without nonzero exit. A port conflict can look successful to a supervisor. No port-conflict integration run was performed. | Phase 2: return the listen error from `server.Run` and test exit behavior. |
| P1 | Demonstrated code-path defect | Scheduler checks `m.Enabled` before enqueue (`internal/scheduler/scheduler.go:74-76`); worker loads the monitor and calls `CheckScheduled` without rechecking (`internal/worker/pool.go:80-94`). A queued job can execute after the monitor is disabled. | Phase 3: skip disabled jobs and test it. |
| P1 | Structural | `cmd/server/main.go:77-205` owns dependency wiring, routes, scheduler start and shutdown. Lifecycle is hard to test apart from the command. | Phase 2: introduce `internal/server.Run` and keep command thin. |
| P2 | Structural | `internal/api` groups all domain handlers, `internal/postgres` groups all SQL, and `internal/check` combines probe engine and application coordination. Incident writes and reads are split between PostgreSQL repository types. | Phase 2: give monitor/check/incident clear ownership; extract `internal/probe`. |
| P2 | Test environment | `GOCACHE=/tmp/horus-go-cache go test ./...` failed because PostgreSQL at localhost:5432 rejected `horus` credentials and Redis at localhost:6379 refused connections. Fixed local endpoints appear in `internal/database/database_test.go`, `internal/migrate/migrate_test.go`, `internal/postgres/*_test.go`, `internal/queue/redis_test.go` and `cmd/server/main_test.go`. Seven dependency-free package suites passed. This does not demonstrate a product regression. CI provides and migrates the services. | Phase 3: make local integration tests opt in or configurable without losing CI coverage. |
| P2 | Reliability limit | Enqueue and schedule update are separate (`internal/scheduler/scheduler.go:87-113`), so a successful push followed by failed update may duplicate a job. Each replica runs a scheduler. Short benchmark replica probes found no duplicates; that does not prove race freedom. | Defer a coordination redesign pending a separate decision. |
| P2 | Reliability limit | Redis BRPOP removes a job before result persistence (`internal/queue/redis.go`, `internal/worker/pool.go`), so a worker failure can lose it. Check insert and incident transition are separate (`internal/check/service.go:52-80`); notification delivery is not retried. | Document and defer durable delivery/transaction redesign. |
| P3 | Scaling limit | Scheduler reads all monitors every second and advances one interval per due monitor (`internal/scheduler/scheduler.go:65-113`). `bench/results-2026-10-05.md` reports rising queue/lag at 500 monitors with three workers and 1,000 with eight on one host. | Preserve behavior during Phase 2; use benchmark as a regression reference. |
| P3 | Documentation/build hygiene | `CLANKER/PROGRESS.md` has stale phase statements. `go test ./...` found an ignored Go package under `web/node_modules` here, so installed frontend dependencies can affect Go package discovery. `.nami/scans.db` and `bench/` are untracked. | Phase 2: capture useful Clanker content. Separately decide ignore/build-context and test-scope changes. |

No new API contract defect was established. The private-target switch materially changes the URL safety policy when enabled; keep its explicit opt-in and isolated-benchmark purpose. Timing fields are currently used for benchmark reports, not frontend responses.

## Proposed boundaries and migration order

1. Freeze current working-tree contracts in tests: routes, JSON, environment names, Redis job compatibility, migrations 001-005 and benchmark timing. Apply migration 005 in integration environments. Preserve all modified and untracked work.
2. Add `internal/server` for wiring, routes, readiness/liveness and lifecycle; keep `cmd/server` thin and `cmd/migrate` independent. Return HTTP listen failures from `server.Run` so the command exits unsuccessfully, and test that behavior.
3. Extract HTTP probe and address policy from `internal/check/checker.go` into `internal/probe`. Give it input/result types that avoid a probe-to-check dependency cycle.
4. Move monitor handler/SQL into `internal/monitor`, then check handler/SQL into `internal/check`, then incident handler/read/write behavior into `internal/incident`. Move one slice at a time and run focused tests. Keep shared HTTP response behavior small; use consumer-side interfaces to avoid cycles.
5. Put Redis connection construction in `internal/database` while queue encoding/operations remain in `internal/queue`. Preserve queue key and JSON. Remove obsolete `internal/api` and `internal/postgres` after callers/tests move. Update maintained docs, then retire tracked `CLANKER/` content after capturing useful guidance.
6. Do bounded Phase 3 fixes: disabled queued jobs and local integration-test gating. Review larger reliability items separately. Run full Go, frontend and image checks with configured services before completing later phases.

## Commands and results

| Command | Result |
| --- | --- |
| `git ls-files`, `git status --short --untracked-files=all`, `rg --files`, `git diff --stat` | Inventoried 104 tracked, 14 modified tracked and 14 untracked files; inspected source, tests, config, migrations, Compose, CI, frontend and Clanker docs. |
| `git diff --check` | Passed before this audit edit. |
| `GOCACHE=/tmp/horus-go-cache go test ./...` | Failed only in packages needing local PostgreSQL/Redis: PostgreSQL password authentication failed; Redis connection refused. Dependency-free tests passed. Package scan also found `web/node_modules/flatted/golang/pkg/flatted`. |
| `GOCACHE=/tmp/horus-go-cache go test ./internal/api ./internal/check ./internal/config ./internal/monitor ./internal/notification ./internal/scheduler ./internal/worker` | Passed all seven packages. |

The audit has been reviewed. Phase 2 can proceed from these findings and the decisions in `docs/phase-2-migration.md`.
