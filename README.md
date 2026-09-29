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
* PostgreSQL persistence

## Tech Stack

* Go
* PostgreSQL
* pgx
* Docker Compose
* HTTP/JSON API

## Requirements

* Go
* Docker
* Docker Compose

## Running Locally

Start PostgreSQL:

```bash
docker compose up -d
```

Run the tests:

```bash
go test ./...
```

Start Horus:

```bash
go run ./cmd/server
```

The API will be available at:

```text
http://localhost:8080
```

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
| `GET`    | `/health`                | Health check      |
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
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   ├── check/
│   ├── database/
│   ├── monitor/
│   ├── postgres/
│   ├── queue/
│   ├── scheduler/
│   └── worker/
├── migrations/
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
