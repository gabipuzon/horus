# Project Overview

## Identity and purpose

Horus is a self-hosted uptime monitoring service written in Go. It stores HTTP monitor configurations, schedules checks, records their outcomes, and serves monitoring data through an HTTP/JSON API. The repository currently implements the core monitoring path; it is not yet a full Pingdom-style product.

## Implemented capabilities

### Monitor management

- Create, list, retrieve, enable, disable, and delete monitors.
- Configure URL, check interval, request timeout, and one expected HTTP status code.
- Persist monitor state and scheduling time in PostgreSQL.

### Health checks and history

- Schedule due enabled monitors and enqueue check jobs in Redis.
- Run checks using a bounded pool of three workers.
- Record status code, latency, success, failure category, optional error text, and timestamp in PostgreSQL.
- Classify unsuccessful results as `http`, `network`, or `timeout`; successful results have an empty failure category.
- List checks newest first with limit/offset pagination and return aggregate count/latency/latest-status summaries.

### Runtime

- Expose a basic `/health` response.
- Handle interrupt/SIGTERM and shut down workers and the HTTP server.

## API

| Method | Path | Behavior |
|---|---|---|
| `GET` | `/health` | Returns `{"status":"ok"}` |
| `POST` | `/monitors` | Creates a monitor; returns `201` and its representation |
| `GET` | `/monitors` | Lists monitors |
| `GET` | `/monitors/{id}` | Retrieves a monitor |
| `DELETE` | `/monitors/{id}` | Deletes a monitor; returns `204` |
| `PATCH` | `/monitors/{id}/enable` | Enables a monitor; returns `204` |
| `PATCH` | `/monitors/{id}/disable` | Disables a monitor; returns `204` |
| `GET` | `/monitors/{id}/checks` | Lists checks; `limit` defaults to 50 and must be 1–100, `offset` defaults to 0 and must be nonnegative |
| `GET` | `/monitors/{id}/summary` | Returns total/success/failure counts, average latency in milliseconds, and latest status code |

Monitor JSON fields are `id`, `name`, `url`, `interval_seconds`, `timeout_seconds`, `expected_status`, and `enabled`. Check rows include `id`, `monitor_id`, `status_code`, `latency_ms`, `success`, `failure_type`, optional `error`, and RFC3339 `checked_at`. Summary fields are `total_checks`, `successful_checks`, `failed_checks`, `average_latency_ms`, and `latest_status` (zero when there are no checks).

## Not implemented yet

Incidents and recovery tracking, notifications, authentication/ownership, uptime percentages, retries/backoff, configurable worker count, readiness checks, metrics, structured logging, production configuration, and protections against SSRF are future work. The monitor constructor checks that name/URL are nonempty, interval/timeout are positive, and expected status is in the HTTP status range; it does not enforce URL schemes or protect private networks.

## Principles and non-goals

- Keep the service a modular monolith while that meets current needs.
- PostgreSQL is durable application state; Redis currently carries check jobs.
- Keep concurrency bounded and propagate contexts through I/O.
- Prefer small, testable changes that match the existing package layout.
- Do not expand into browser automation, an enterprise monitoring suite, or microservices without a demonstrated need.
