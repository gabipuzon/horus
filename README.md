# Horus

Horus is a self-hosted uptime monitoring service written in Go. It checks HTTP
and HTTPS URLs, stores results and incidents in PostgreSQL, and exposes an
HTTP/JSON API. The backend is one process with separate API, scheduler, and
worker packages.

## Features

- Create, list, retrieve, enable, disable, and delete monitors.
- Schedule periodic checks through a Redis list and a bounded worker pool.
- Compare HTTP responses with an expected status; classify HTTP, network, and
  timeout failures; persist check history and latency.
- Open one incident for an outage and resolve it on recovery. Optionally send
  Discord DOWN and RECOVERED webhook notifications on those transitions.
- Query check history, a check-based uptime summary, incident history, and the
  current incident.
- Run with Docker Compose, forward-only PostgreSQL migrations, process
  liveness/readiness endpoints, and GitHub Actions CI.
- Use the separate React monitor dashboard to list, create, enable, disable,
  and delete monitors.

## Architecture

```text
HTTP API ──> PostgreSQL monitors, checks, incidents
                   │
             1s scheduler
                   │
             Redis job list
                   │
          bounded worker pool
                   │
             HTTP checker
                   │
          persisted check result
                   │
          incident transition ──> optional Discord webhook
```

The scheduler enqueues due monitors before advancing `next_check_at`. Workers
consume jobs, perform checks, then persist each result before attempting the
incident transition. An incident transition failure leaves the check stored.
A webhook failure leaves both the check and incident state stored.

## Quick Start

Requires Docker and Docker Compose. The preferred local start is:

```bash
cp .env.example .env
docker compose up --build -d
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

The expected responses are `{"status":"ok"}` and `{"status":"ready"}`, both
with HTTP 200. Compose waits for healthy PostgreSQL and Redis, applies
migrations in a one-shot job, then starts Horus. Inside containers, Horus uses
the `postgres` and `redis` service names. If the migration job fails, Horus
does not start.

```bash
docker compose down       # Stop; keep PostgreSQL data
docker compose down -v    # Intentionally delete local volumes and data
```

PostgreSQL uses the `postgres_data` named volume. The second command deletes
that data; use it only when a fresh local database is intended.

## Configuration

All settings have local development defaults. Go reads the **process
environment**, never `.env` automatically. Docker Compose reads `.env` for
substitution; it supplies container-specific hosts and ports. Blank database
host/user/name, Redis host, HTTP address, and DB/Redis ports are rejected.
Ports must be 1–65535. An empty database password is permitted if PostgreSQL
allows passwordless authentication.

| Variable | Default | Purpose |
|---|---|---|
| `HORUS_DB_HOST` | `localhost` | PostgreSQL host |
| `HORUS_DB_PORT` | `5432` | PostgreSQL port |
| `HORUS_DB_USER` | `horus` | PostgreSQL user |
| `HORUS_DB_PASSWORD` | `horus` | PostgreSQL password |
| `HORUS_DB_NAME` | `horus` | PostgreSQL database |
| `HORUS_REDIS_HOST` | `localhost` | Redis host |
| `HORUS_REDIS_PORT` | `6379` | Redis port |
| `HORUS_HTTP_ADDR` | `:8080` | HTTP listen address |
| `HORUS_WORKER_COUNT` | `3` | Worker count, allowed 1–1000 |
| `HORUS_DISCORD_WEBHOOK_URL` | empty | Optional absolute HTTP(S) URL; empty disables notifications |

The safe examples in `.env.example` use `localhost` for direct Go runs.
Compose sets `HORUS_DB_HOST=postgres`, `HORUS_REDIS_HOST=redis`, internal ports,
and `HORUS_HTTP_ADDR=:8080`. `HORUS_DB_PUBLISH_PORT`,
`HORUS_REDIS_PUBLISH_PORT`, and `HORUS_HTTP_PUBLISH_PORT` only change the
host-side Compose ports; they are not Horus process settings. Change example
credentials before using Horus outside local development.

## Migrations

`cmd/migrate` embeds the numbered SQL files in `migrations/` and applies
pending versions in numeric order. `schema_migrations` records completed
versions. Each migration and its version record run in one transaction, so a
failed migration rolls back and exits nonzero. A fresh database applies every
version; an already-current database applies zero and exits successfully.
Compose runs this command before Horus. For direct Go runs, invoke it
separately; the server does not migrate at startup.

A database migrated manually before `schema_migrations` existed may need an
intentional baseline or a fresh volume. The runner does not infer completed
versions from existing tables and has no rollback command.

## API

All paths are relative to `http://localhost:8080` with the default host port.

| Method | Path | Success |
|---|---|---|
| `GET` | `/health` | `200` process liveness |
| `GET` | `/ready` | `200` dependency readiness |
| `POST` | `/monitors` | `201` created monitor |
| `GET` | `/monitors` | `200` monitor array |
| `GET` | `/monitors/{id}` | `200` monitor |
| `DELETE` | `/monitors/{id}` | `204`, empty body |
| `PATCH` | `/monitors/{id}/enable` | `204`, empty body |
| `PATCH` | `/monitors/{id}/disable` | `204`, empty body |
| `GET` | `/monitors/{id}/checks` | `200` check array |
| `GET` | `/monitors/{id}/summary` | `200` summary |
| `GET` | `/monitors/{id}/incidents` | `200` incident array |
| `GET` | `/monitors/{id}/incidents/current` | `200` open incident or `204`, empty body |

### Monitors

Create a monitor with one JSON object:

```bash
curl -i -X POST http://localhost:8080/monitors \
  -H 'Content-Type: application/json' \
  -d '{"name":"Example","url":"https://example.com","interval_seconds":30,"timeout_seconds":5,"expected_status":200}'
```

The response includes `id`, `name`, `url`, `interval_seconds`,
`timeout_seconds`, `expected_status`, and `enabled`. Name must be nonblank;
URL must be absolute HTTP(S) without embedded credentials; interval and
timeout must be positive whole seconds that fit the PostgreSQL integer
columns; expected status must be 100–599. Malformed JSON, unknown fields, and
trailing JSON are rejected with `400`. DNS and IP safety checks occur when a
check runs. Missing or malformed UUIDs return `404` for monitor reads,
deletes, and enable/disable operations.

### Checks and summary

`GET /monitors/{id}/checks` returns newest checks first, breaking equal
`checked_at` timestamps by ID descending. `limit` defaults to 50 and must be
1–100; `offset` defaults to 0 and must be nonnegative. Invalid, empty, or
out-of-range values return `400`. An existing monitor without checks returns
`200 []`. Check rows include `id`, `monitor_id`, `status_code`,
`latency_ms`, `success`, `failure_type`, `checked_at`, and `error` when
present. Status 0 means no HTTP response.

`GET /monitors/{id}/summary` returns `total_checks`,
`successful_checks`, `failed_checks`, `average_latency_ms`,
`latest_status`, and `uptime_percentage`. Uptime is **successful persisted
checks ÷ all persisted checks × 100**, rounded to two decimals. It is the
share of successful checks, not time-weighted availability. Average latency
is truncated to whole milliseconds. Latest status comes from the newest
check; 0 means that check had no HTTP response. With no checks, the counts,
average latency, and latest status are 0, while uptime is `null`.

### Incidents

`GET /monitors/{id}/incidents` uses the same `limit` and `offset` rules as
check history. It orders newest `started_at` first, then incident ID
descending; an existing monitor without incidents returns `200 []`.
`GET /monitors/{id}/incidents/current` returns only the open incident, or
`204` with no body when none is open.

Incidents expose `id`, `monitor_id`, `started_at`, `resolved_at`,
`is_open`, `duration_ms`, `failure_type`, `status_code`, and
`failure_message` when present. The failure context is captured when the
incident opens; repeated failed checks keep it unchanged. For an open
incident, `resolved_at` is `null` and duration grows from start to response
time. For a resolved incident, duration is fixed from start to
`resolved_at`. Negative durations clamp to zero. Status 0 means no initial
HTTP response. Missing or malformed monitor IDs return `404` on both check
and incident routes.

### Errors

Horus handler errors use `Content-Type: application/json` with a single
field, for example `{"error":"monitor not found"}`. `400` means invalid
input, `404` means a missing or malformed monitor ID, and unexpected
repository failures return `500 {"error":"internal server error"}` without
internal details. Readiness is the deliberate exception: dependency failure
returns `503 {"status":"not_ready"}`. Unsupported paths and methods use
Go `http.ServeMux` defaults, which may not use this JSON shape.

## Notifications

Set `HORUS_DISCORD_WEBHOOK_URL` to an absolute HTTP(S) webhook URL without
embedded credentials to enable Discord notifications. An unset or blank value
disables them. Startup logs enabled/disabled state without logging the URL.

| Check transition | Discord message |
|---|---|
| Healthy → failed; incident opens | One DOWN |
| Failed → failed; same incident remains open | None |
| Open incident → successful; incident resolves | One RECOVERED |
| Healthy → successful | None |

Messages include monitor and incident context, with recovery time and outage
duration on recovery. A failed webhook delivery does not undo the persisted
check or incident transition; the worker logs the error. There is no retry or
durable notification queue. Enabling notifications later does not replay
transitions that already occurred.

## Health and Readiness

`GET /health` is pure process liveness: `200 {"status":"ok"}` even when
dependencies are unavailable. `GET /ready` pings PostgreSQL and Redis using
one shared one-second deadline; it returns `200 {"status":"ready"}` only
when both respond, otherwise `503 {"status":"not_ready"}`. Responses do not
expose dependency errors. PostgreSQL and Redis must both be available at
startup; Horus exits if either initial check fails.

SIGINT and SIGTERM cancel the scheduler and workers. An idle Redis dequeue
uses a finite one-second blocking wait so workers can exit promptly. After
joining workers, Horus gives HTTP shutdown up to five seconds.

## Development

Go and local PostgreSQL/Redis are needed for a direct process run. One
convenient path uses Compose for dependencies only:

```bash
cp .env.example .env
docker compose up -d postgres redis
set -a
. ./.env
set +a
go run ./cmd/migrate
go run ./cmd/server
```

The exported `localhost` settings connect to the published dependency
ports. Horus itself does not load `.env`. Stop the direct server with
Ctrl+C; stop the dependencies with `docker compose down`.

### Frontend development

The Phase 9A frontend lives in `web/` and requires Node.js 20.19+ and npm.
Start Horus first using the Compose or direct Go instructions above, then:

```bash
cd web
cp .env.example .env
npm ci
npm run dev
```

Open `http://localhost:5173/monitors`. The frontend defaults to same-origin
`/api` requests; Vite proxies those requests to `http://localhost:8080` and
removes `/api` before forwarding. Set `VITE_HORUS_API_URL` to an absolute API
origin only when that origin explicitly allows the browser's frontend origin.
For a production static host, proxy `/api/*` to the Horus API with the same
prefix removal. The backend Compose stack does not serve or deploy the
frontend. The monitor list shows configuration state (enabled/disabled), not
live health, uptime, or latency.

Frontend checks:

```bash
cd web
npm test
npm run build
npm run lint
```

## Testing and CI

With local PostgreSQL and Redis running and migrations applied:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
docker build .
```

Repository and queue integration tests use real local services. GitHub
Actions runs on pushes to `main` and pull requests targeting `main`. Its
single workflow starts healthy PostgreSQL and Redis services, applies
migrations with `cmd/migrate`, then runs tests, race tests, vet, Go build,
and Docker build. It does not deploy or publish an image.

## Known Limitations

- Uptime counts check outcomes; it is not time-weighted. Offset pagination
  can shift when checks or incidents are added between page requests.
- There are no users, authentication, or monitor ownership; expose the API
  only in an environment you control. There is no flapping suppression.
- Redis list jobs have no acknowledgement or recovery protocol. An enqueue
  followed by a failed `next_check_at` update can create a duplicate job.
- Check persistence and incident transitions are separate operations. A
  transition failure can leave an incident temporarily out of sync with a
  persisted check. Failed Discord delivery is not retried.
- Readiness checks dependency connectivity, not schema state, scheduler or
  worker progress, or queued-job delivery. Startup requires both dependencies.
- Go does not load `.env` automatically. Databases migrated manually before
  `schema_migrations` may need a baseline or reset. URL safety checks are
  partial and should be reviewed before accepting untrusted monitor URLs.

## Project Status

The backend v1 flow is implemented and covered by Go tests and local Compose
verification. The Phase 9A monitor frontend is implemented as a separate Vite
development app. Monitor detail, checks, summary, incident screens, frontend
deployment, authentication, and metrics are not implemented.
