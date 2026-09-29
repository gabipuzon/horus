# Project Progress

## Current state

Horus has an end-to-end core monitoring flow: manage monitors over HTTP, schedule due checks, enqueue jobs in Redis, execute checks with a bounded worker pool, persist check results in PostgreSQL, and query check history and summaries through the API.

## Implemented

### Monitor API and persistence

- Monitor model with UUID IDs and validation for required name/URL, positive interval/timeout, and HTTP status range.
- PostgreSQL monitor repository and migrations for monitor state, including `next_check_at`.
- Create, list, retrieve, delete, enable, and disable monitor endpoints.

### Checks and history

- HTTP GET checker with monitor timeout, latency measurement, expected-status comparison, and `http`/`network`/`timeout` failure classification.
- PostgreSQL check repository, foreign-key cascade, and monitor/time history index.
- Check history endpoint with `limit`/`offset` pagination (default limit 50, maximum 100).
- Summary endpoint with total/success/failure counts, average latency, and latest HTTP status.

### Scheduling and workers

- One-second scheduler selects enabled monitors whose `next_check_at` is due, advances their schedule, and queues jobs in Redis list `horus:checks`.
- Redis client provides ping, enqueue (`LPUSH`), and blocking dequeue (`BRPOP`).
- Application starts three workers in the same process; workers load monitors, run checks, and persist results.
- Process handles interrupt/SIGTERM, cancels and joins workers, and gracefully shuts down the HTTP server.

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
```

## Known gaps and limitations

- No incident lifecycle, outage/recovery state, or notifications.
- No user accounts, authentication, authorization, or monitor ownership.
- No SSRF protections, URL scheme restriction, private-network/metadata blocking, or redirect validation.
- No retries/backoff, duplicate-job protection, queue recovery/dead-letter handling, or configurable worker count.
- No readiness endpoint, metrics, structured logging, production configuration, or container image for Horus.
- PostgreSQL/Redis addresses and credentials are hardcoded in `cmd/server/main.go`; migrations must be applied manually.
- Monitor enable/disable and delete handlers do not distinguish a missing ID from a successful update/delete.
- Checker uses `http.DefaultClient`; status mismatch is checked against one exact expected status.

## Verification record

The repository includes unit/API tests and PostgreSQL-backed repository tests. Database tests require a reachable local PostgreSQL service and applied migrations. This documentation update did not run tests or change implementation code; no new test result is claimed.

## Next work

Choose the next implementation step from product priorities. Security around user-controlled target URLs and configurable runtime settings are important gaps before exposing the service beyond a trusted local environment. Incidents and notifications remain future product capabilities.
