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
normal restarts. Open `http://localhost:8080` for the API. `GET /health` checks
process liveness; `GET /ready` checks PostgreSQL and Redis connectivity.
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
through your process environment. Run tests with:

```bash
go test ./...
```

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
  "latest_status": 200
}
```

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
| `POST`   | `/monitors`              | Create a monitor  |
| `GET`    | `/monitors`              | List monitors     |
| `GET`    | `/monitors/{id}`         | Get a monitor     |
| `DELETE` | `/monitors/{id}`         | Delete a monitor  |
| `PATCH`  | `/monitors/{id}/enable`  | Enable a monitor  |
| `PATCH`  | `/monitors/{id}/disable` | Disable a monitor |
| `GET`    | `/monitors/{id}/checks`  | Get check history |
| `GET`    | `/monitors/{id}/summary` | Get check summary |

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
