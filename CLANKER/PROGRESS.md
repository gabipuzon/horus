# Project Progress

## Current state

Horus has an end-to-end core monitoring flow: manage monitors over HTTP, schedule due checks, enqueue jobs in Redis, execute checks with a bounded worker pool, persist checks and incidents in PostgreSQL, and query check and incident data through the API.

The code is organized by responsibility: `api`, `check`, `config`, `database`, `incident`, `monitor`, `notification`, `postgres`, `queue`, `scheduler`, and `worker` packages under `internal/`.

## Implemented

### Monitor API and persistence

- Monitor model with UUID IDs and validation for required name/URL, positive interval/timeout, and HTTP status range.
- PostgreSQL monitor repository and migrations for monitor state, including `next_check_at`.
- Create, list, retrieve, delete, enable, and disable monitor endpoints.

### Checks and history

- HTTP GET checker with monitor timeout, latency measurement, expected-status comparison, and `http`/`network`/`timeout` failure classification.
- PostgreSQL check repository, foreign-key cascade, and monitor/time history index.
- Check history endpoint with `limit`/`offset` pagination (default limit 50, maximum 100).
- Summary endpoint with total/success/failure counts, average latency, latest HTTP status, and check-based uptime percentage. Uptime is successful checks divided by all persisted checks, rounded to two decimals; it is `null` when there are no checks.

### Incident lifecycle

- PostgreSQL incidents are associated with monitors and store outage start, optional resolution, and the initial failure type, status code, and error message.
- At most one open incident per monitor is enforced by a partial unique index. Repeated failures leave the existing incident and its initial failure context unchanged; a successful check resolves it.
- Check results are persisted before incident transitions. If a transition fails, the check remains stored and the check service returns an explicit transition error.
- Migration `004_create_incidents.sql` adds incident storage and indexes. Migrations remain manual.
- Incident history is paginated newest first. The current incident endpoint returns the open incident or `204` when none is open. Responses include timestamps, open state, initial failure context, and elapsed or resolved duration in milliseconds.

### Discord notifications

- Optional `HORUS_DISCORD_WEBHOOK_URL` enables DOWN on incident creation and RECOVERED on resolution. Repeated failures and ordinary successes send nothing.
- The webhook message includes monitor name and URL, incident failure context and start time, and recovery time and duration when resolved.
- Requests use the check context and a five-second deadline. A Discord failure leaves the persisted check and incident transition intact and is returned to the worker for logging; there is no delivery retry or durable queue.

### Scheduling and workers

- One-second scheduler selects enabled monitors whose `next_check_at` is due, queues jobs in Redis list `horus:checks`, then advances their schedule. A per-monitor enqueue or schedule-update failure is logged without blocking later monitors in the same pass. Failed enqueues leave the monitor due; update errors are reevaluated from PostgreSQL on the next tick.
- Redis client provides ping, enqueue (`LPUSH`), and blocking dequeue (`BRPOP`) with a one-second wait so an idle worker can observe cancellation promptly.
- Application starts three workers in the same process; workers load monitors, run checks, and persist results.
- Process handles interrupt/SIGTERM, cancels and joins workers, and gracefully shuts down the HTTP server. Idle Redis dequeues no longer wait indefinitely before worker shutdown can finish.

### Runtime/API

- `GET /health` returns a basic `{"status":"ok"}` response.
- PostgreSQL 17 and Redis 8 are available through `compose.yaml`.

## Routes

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
GET    /monitors/{id}/incidents
GET    /monitors/{id}/incidents/current
```

## Known gaps and limitations

- No durable notification delivery or time-weighted uptime calculation; the reported percentage counts check outcomes.
- No user accounts, authentication, authorization, or monitor ownership.
- The checker accepts only absolute HTTP/HTTPS URLs without embedded credentials, rejects non-public DNS results at connection time, and does not follow redirects. Other SSRF edge cases should continue to be reviewed as the service is hardened.
- No check execution retries, duplicate-job protection, or queue recovery/dead-letter handling. Scheduling errors are retried on the next tick, and workers back off after dequeue errors. If enqueue succeeds but updating `next_check_at` fails, a later tick can enqueue the same monitor again.
- No readiness endpoint, metrics, structured logging, production configuration, or container image for Horus. `/health` reports that the HTTP process is serving requests; it does not verify that scheduling, PostgreSQL, or Redis are healthy.
- PostgreSQL/Redis addresses and credentials, HTTP listen address, worker count, and optional Discord webhook are configurable through `HORUS_DB_HOST`, `HORUS_DB_PORT`, `HORUS_DB_USER`, `HORUS_DB_PASSWORD`, `HORUS_DB_NAME`, `HORUS_REDIS_HOST`, `HORUS_REDIS_PORT`, `HORUS_HTTP_ADDR`, `HORUS_WORKER_COUNT`, and `HORUS_DISCORD_WEBHOOK_URL`. Local Compose-compatible defaults are used when unset; migrations must still be applied manually.
- Monitor enable/disable and delete handlers do not distinguish a missing ID from a successful update/delete.
- Checker uses `http.DefaultClient`; status mismatch is checked against one exact expected status.

## Verification record

The repository includes unit/API tests and PostgreSQL/Redis-backed repository and queue tests. Database tests require reachable local services and applied migrations. For the Redis cancellation and scheduler fixes, targeted scheduler/worker/queue tests, `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` passed with local services. A controlled run processed the reported overdue monitor through a failed check and an incident opening; its diagnostic data was then removed. An idle-worker process exited in under one second after Ctrl+C with real Redis. The original scheduling inactivity was not reproduced, so its historical cause remains unconfirmed.

## Next work

Continue production hardening with authentication/authorization, queue recovery, readiness and operational telemetry. Durable notification delivery remains future work.
