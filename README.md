# Horus

Horus is a lightweight uptime monitoring service written in Go.

It periodically checks configured URLs, records the results in PostgreSQL, and exposes an HTTP API for managing monitors and viewing check history.

## Current Architecture

```text
                    ┌──────────────┐
                    │   Horus API  │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  PostgreSQL  │
                    │   Monitors   │
                    │    Checks    │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Scheduler   │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Worker Pool  │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ HTTP Checker │
                    └──────┬───────┘
                           │
                           ▼
                      Monitored URL
```

## Features

* Create, list, retrieve, enable, disable, and delete monitors
* Configurable check intervals
* Configurable request timeouts
* Expected HTTP status validation
* HTTP, network, and timeout failure classification
* Response latency measurement
* Persistent check history
* Paginated check history
* Check history summaries
* Concurrent check workers
* Graceful application shutdown
* PostgreSQL persistence and forward-only schema migrations
* Docker Compose startup for PostgreSQL, Redis, migrations, and Horus

## Tech Stack

* Go
* PostgreSQL
* pgx
* Docker Compose
* HTTP/JSON API

## Requirements

* Docker and Docker Compose for the preferred quick start
* Go for running the process directly

## Continuous integration

GitHub Actions runs on pushes to `main` and pull requests targeting `main`.
CI starts PostgreSQL and Redis, applies the SQL migrations with `cmd/migrate`,
then runs tests, the race detector, `go vet`, `go build`, and a Docker image build.
The workflow does not need a Discord webhook or repository secrets.

## Quick start with Docker Compose

Copy the safe local defaults, then build and start the stack:

```bash
cp .env.example .env
docker compose up --build
```

Compose starts PostgreSQL and Redis, waits for both to be healthy, applies pending
SQL migrations, then starts Horus. The migration job records completed versions in
`schema_migrations`; later starts skip them. A failed migration prevents Horus from
starting. PostgreSQL data lives in the named `postgres_data` volume and survives
normal restarts. Open `http://localhost:8080` for the API. `GET /health` returns
`200 {"status":"ok"}` for process liveness without checking dependencies.
`GET /ready` gives PostgreSQL and Redis a shared one-second deadline: both
reachable returns `200 {"status":"ready"}`; either unavailable returns
`503 {"status":"not_ready"}` without dependency error details. Readiness
checks connectivity, not scheduler or worker progress.
Databases initialized manually before `schema_migrations` need an explicit
migration baseline; the command does not infer completed versions from tables.

Set `HORUS_DISCORD_WEBHOOK_URL` in `.env` to enable optional incident notifications.
Leave it blank to disable them. `.env` is read by Compose; it is not copied into
the image. Compose sets database and Redis hosts to `postgres` and `redis` inside
containers. The example's `localhost` values are for direct Go runs. Host ports
can be changed with `HORUS_HTTP_PUBLISH_PORT`, `HORUS_DB_PUBLISH_PORT`, and
`HORUS_REDIS_PUBLISH_PORT`.

Stop containers with `Ctrl+C` or `docker compose down`. To intentionally delete
local PostgreSQL and Redis data as well, run `docker compose down -v`.

## Run Go directly during development

Start the dependencies, export local settings, then migrate and start the server:

```bash
docker compose up -d postgres redis
cp .env.example .env
set -a
. ./.env
set +a
go run ./cmd/migrate
go run ./cmd/server
```

`cmd/migrate` is safe to run again: it applies only unrecorded SQL migrations.
Horus itself does not load `.env`; export it as shown or set `HORUS_*` variables
through your process environment. Horus requires PostgreSQL and Redis at
startup and exits if either initial connection check fails. The server does
not run migrations itself; run `cmd/migrate` first when starting Go directly.
Run tests with:

```bash
go test ./...
```

The following settings are read from the process environment. Unset values use
the local development defaults shown here. Blank host, user, database name,
HTTP address, and port values are rejected; ports must be 1–65535. A blank
database password is allowed only if the PostgreSQL server permits it.

| Variable | Default | Purpose |
|---|---|---|
| `HORUS_DB_HOST` | `localhost` | PostgreSQL host |
| `HORUS_DB_PORT` | `5432` | PostgreSQL port |
| `HORUS_DB_USER` | `horus` | PostgreSQL user |
| `HORUS_DB_PASSWORD` | `horus` | PostgreSQL password |
| `HORUS_DB_NAME` | `horus` | PostgreSQL database |
| `HORUS_REDIS_HOST` | `localhost` | Redis host |
| `HORUS_REDIS_PORT` | `6379` | Redis port |
| `HORUS_HTTP_ADDR` | `:8080` | Horus listen address |
| `HORUS_WORKER_COUNT` | `3` | Worker count, 1–1000 |
| `HORUS_DISCORD_WEBHOOK_URL` | empty | Optional absolute HTTP(S) webhook URL; empty disables notifications |

Compose uses the same database credentials but sets the container hosts to
`postgres` and `redis` and their internal ports to `5432` and `6379`. The
`HORUS_*_PUBLISH_PORT` values in `.env.example` change host-side Compose ports,
not the process configuration inside containers.

SIGINT or SIGTERM cancels the scheduler and workers. Idle Redis dequeues wake
within their one-second blocking interval to observe cancellation. Horus joins
the workers, then gives the HTTP server up to five seconds to shut down.

### Optional Discord notifications

For a direct Go run, Horus reads the process environment; it does not
automatically load `.env`. After setting `HORUS_DISCORD_WEBHOOK_URL` in `.env`,
export its values before starting Horus:

```bash
(
  set -a
  . ./.env
  set +a
  go run ./cmd/server
)
```

Startup logs `Discord notifications enabled` or `Discord notifications disabled`
without exposing the webhook URL. An unset or blank value disables notifications.
Production deployments should provide the variable through their process environment.

DOWN is sent only when a new incident opens; RECOVERED is sent when that incident
resolves. For a DOWN test, create a fresh monitor for
`https://httpbin.org/status/500` expecting HTTP 200. Repeated failures during an
existing incident do not send more messages. Enabling notifications or restarting
Horus does not replay an incident that opened while notifications were disabled.
Webhook failures are logged without the webhook URL and do not undo persisted
checks or incidents; delivery is not retried.

## Creating a Monitor

Create a monitor for a public HTTP endpoint:

```bash
curl -X POST http://localhost:8080/monitors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "HTTPBin",
    "url": "https://httpbin.org/status/200",
    "interval_seconds": 10,
    "timeout_seconds": 5,
    "expected_status": 200
  }'
```

The response contains the monitor ID:

```json
{
  "id": "monitor-id",
  "name": "HTTPBin",
  "url": "https://httpbin.org/status/200",
  "interval_seconds": 10,
  "timeout_seconds": 5,
  "expected_status": 200,
  "enabled": true
}
```

Creation returns `201 Created`. The request must be one JSON object with only the
documented fields; malformed JSON, unknown fields, and trailing data return `400`.
Name must not be blank. URL must be an absolute HTTP or HTTPS URL without embedded
credentials. Interval and timeout must be positive whole seconds that fit the
PostgreSQL integer columns; expected status must be 100–599. URL DNS/IP checks
still happen when a check runs, not during creation.

Horus will then check the URL every 10 seconds.

## Viewing Check History

Replace `MONITOR_ID` with the ID returned when creating the monitor:

```bash
curl http://localhost:8080/monitors/MONITOR_ID/checks
```

Pagination is supported:

```bash
curl "http://localhost:8080/monitors/MONITOR_ID/checks?limit=20&offset=0"
```

Checks are returned newest first by `checked_at` (with check ID breaking timestamp
ties). `limit` defaults to 50 and must be 1–100; `offset` defaults to 0 and must
be nonnegative. Invalid or empty pagination values return `400`. An existing
monitor with no checks returns `200` and `[]`; a missing or malformed monitor ID
returns `404`. Each check includes its ID, monitor ID, HTTP status (0 if no
response was received), latency in milliseconds, success, failure type, and
check time. A stored error message is included when present.

## Viewing a Summary

```bash
curl http://localhost:8080/monitors/MONITOR_ID/summary
```

Example:

```json
{
  "total_checks": 10,
  "successful_checks": 9,
  "failed_checks": 1,
  "average_latency_ms": 42,
  "latest_status": 200,
  "uptime_percentage": 90
}
```

`uptime_percentage` is successful persisted checks divided by all persisted
checks, multiplied by 100 and rounded to two decimal places. It measures check
outcomes, not elapsed uptime. `average_latency_ms` is the average of persisted
latencies, truncated to whole milliseconds. `latest_status` comes from the
newest check; 0 means that check received no HTTP response. For an existing
monitor with no checks, all count, latency, and status fields are 0 and
`uptime_percentage` is `null`. A missing or malformed monitor ID returns `404`.

## Viewing Incidents

```bash
curl "http://localhost:8080/monitors/MONITOR_ID/incidents?limit=20&offset=0"
curl http://localhost:8080/monitors/MONITOR_ID/incidents/current
```

History returns the newest incident first by `started_at`, with incident ID
breaking timestamp ties. `limit` defaults to 50 and must be 1–100; `offset`
defaults to 0 and must be nonnegative. Invalid or empty pagination values
return `400`. An existing monitor with no incidents returns `200` and `[]`.
The current route returns the open incident with `200`, or `204 No Content`
with an empty body if none is open. Both routes return `404` for a missing or
malformed monitor ID.

Incident responses include `id`, `monitor_id`, `started_at`, `resolved_at`,
`is_open`, `duration_ms`, `failure_type`, and `status_code`. The initial
`failure_message` appears when present. An open incident has `resolved_at: null`
and `duration_ms` measures elapsed time from its start to the response. A
resolved incident's duration is fixed at `resolved_at - started_at`. Negative
durations are reported as 0. Status 0 means no HTTP response was recorded for
the initial failure. Repeated failed checks keep the same incident and its
initial failure context.

## Testing Failure Detection

Horus considers a check successful when the returned HTTP status matches the configured expected status.

For example, this endpoint always returns HTTP 500:

```text
https://httpbin.org/status/500
```

Create a monitor expecting HTTP 200:

```bash
curl -X POST http://localhost:8080/monitors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "HTTPBin Failure",
    "url": "https://httpbin.org/status/500",
    "interval_seconds": 10,
    "timeout_seconds": 5,
    "expected_status": 200
  }'
```

Horus records the result as an HTTP failure:

```json
{
  "status_code": 500,
  "success": false,
  "failure_type": "http"
}
```

## API

| Method   | Endpoint                 | Description       |
| -------- | ------------------------ | ----------------- |
| `GET`    | `/health`                | Process liveness check |
| `GET`    | `/ready`                 | PostgreSQL and Redis readiness check |
| `POST`   | `/monitors`              | Create a monitor (`201`; invalid request `400`) |
| `GET`    | `/monitors`              | List monitors (`200`) |
| `GET`    | `/monitors/{id}`         | Get a monitor (`200`; missing or malformed ID `404`) |
| `DELETE` | `/monitors/{id}`         | Delete a monitor (`204`; missing or malformed ID `404`) |
| `PATCH`  | `/monitors/{id}/enable`  | Enable a monitor (`204`; missing or malformed ID `404`) |
| `PATCH`  | `/monitors/{id}/disable` | Disable a monitor (`204`; missing or malformed ID `404`) |
| `GET`    | `/monitors/{id}/checks`  | Get check history |
| `GET`    | `/monitors/{id}/summary` | Get check summary |

Errors produced by Horus API handlers use `Content-Type: application/json` and a single field, for example `{"error":"monitor not found"}`. Invalid JSON, monitor values, or pagination return `400` with a useful message. Missing monitors and malformed monitor IDs return `404`. Unexpected repository failures return `500` with `{"error":"internal server error"}`; internal details are not sent to clients. `/ready` is an operational exception: dependency failure returns `503` with `{"status":"not_ready"}`. Go's default `ServeMux` handles unsupported paths and methods.

## Project Structure

```text
horus/
├── cmd/
│   ├── migrate/
│   └── server/
├── internal/
│   ├── api/
│   ├── check/
│   ├── config/
│   ├── database/
│   ├── incident/
│   ├── migrate/
│   ├── monitor/
│   ├── notification/
│   ├── postgres/
│   ├── queue/
│   ├── scheduler/
│   └── worker/
├── migrations/
├── Dockerfile
├── compose.yaml
├── go.mod
└── README.md
```

## Project Status

Horus currently has a working end-to-end monitoring pipeline:

```text
API
 ↓
PostgreSQL
 ↓
Scheduler
 ↓
Redis Queue
↓
Worker Pool
 ↓
HTTP Checker
 ↓
External URL
 ↓
Check Result
 ↓
PostgreSQL
 ↓
API
```

The project is currently focused on building the core monitoring infrastructure before adding additional functionality.
