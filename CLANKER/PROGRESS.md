# PROGRESS.md

## STATE

- Status: active development
- Source of truth: confirmed repository implementation + tests
- Update policy: update only after verified changes
- Writable context file: yes
- Other AI context files: read-only

## COMPLETED

### Foundation

- [x] Initialize Go project
- [x] Create HTTP server
- [x] Create Monitor domain
- [x] Add monitor validation
- [x] Generate monitor UUIDs
- [x] Add configurable HTTP checker
- [x] Add request timeout handling
- [x] Add failure classification
- [x] Add `POST /monitors`
- [x] Define API response model
- [x] Add monitor creation API tests

### PostgreSQL

- [x] Add PostgreSQL Docker Compose environment
- [x] Add PostgreSQL connection module
- [x] Add monitors migration
- [x] Add monitor repository
- [x] Persist monitors through API
- [x] Load monitors from PostgreSQL
- [x] Add `GET /monitors`
- [x] Add `GET /monitors/{id}`
- [x] Add `DELETE /monitors/{id}`
- [x] Add monitor enable/disable persistence
- [x] Add monitor enable endpoint
- [x] Add monitor disable endpoint
- [x] Add API tests for monitor CRUD/control operations

## CURRENT DOMAIN

### Monitor API

```text
POST   /monitors
GET    /monitors
GET    /monitors/{id}
DELETE /monitors/{id}
PATCH  /monitors/{id}/enable
PATCH  /monitors/{id}/disable
````

### Monitor Persistence

Current monitor fields:

```text
id
name
url
interval_seconds
timeout_seconds
expected_status
enabled
created_at
updated_at
```

### Monitor Checker

Current failure classifications:

```text
http
network
timeout
```

Current checker responsibilities:

* HTTP request execution
* Context timeout
* Latency measurement
* Expected status validation
* Failure classification

## VERIFIED

* `go test ./...` passes after completed implementation steps.
* PostgreSQL development container is operational.
* PostgreSQL connection has been verified.
* Monitor migration has been applied.
* Monitor persistence has been manually verified.
* API handlers use repository interfaces for testability.
* Monitor API behavior has automated tests.

## CURRENT NEXT AREA

### Check History

Implement persistent health-check results.

Planned sequence:

```text
1. Design checks table
2. Create checks migration
3. Create check repository
4. Persist CheckResult
5. Add GET /monitors/{id}/checks
6. Add pagination
7. Add latency/status history
```

## CHECK DATA

Expected conceptual fields:

```text
id
monitor_id
checked_at
status_code
latency
success
failure_type
error
```

Exact schema should be designed before implementation.

## FUTURE AREAS

### Monitoring Engine

```text
scheduler
→ monitor selection
→ job creation
→ worker pool
→ HTTP checker
→ CheckResult
→ persistence
```

### Incidents

```text
CheckResult
→ failure detection
→ incident creation/update
→ recovery detection
→ incident resolution
```

### Async Infrastructure

* Redis
* Check queue
* Workers
* Retry/backoff
* Notification queue

### Notifications

* Notification domain
* Email
* Webhooks
* Discord/Slack
* Outage alerts
* Recovery alerts
* Cooldowns

### Authentication

* Users
* Password hashing
* Login
* Authentication middleware
* Monitor ownership
* Authorization

### Observability

* Structured logging
* Request logging
* Prometheus metrics
* Worker metrics
* Database metrics
* Health/readiness endpoints

### Production

* Configuration management
* Environment variables
* Graceful shutdown
* Rate limiting
* Request limits
* SSRF protection
* Security headers
* Dockerfile
* Production deployment

### Portfolio

* Architecture diagram
* API documentation
* Setup documentation
* Architecture decisions
* Benchmarks
* Load testing
* Dashboard/screenshots
* CV project description
* GitHub cleanup

## KNOWN CONSTRAINTS

* PostgreSQL is the durable source of truth.
* Redis is for asynchronous coordination, not primary business state.
* HTTP handlers must not contain SQL.
* Repository interfaces should remain small.
* External monitoring targets are untrusted input.
* Worker concurrency must be bounded.
* Context cancellation must propagate through I/O.
* Do not introduce infrastructure without a concrete requirement.

## GIT

* Use one logical change per commit.
* Prefer conventional commit prefixes:

  * `feat:`
  * `fix:`
  * `test:`
  * `refactor:`
  * `chore:`
  * `docs:`
* Do not rewrite history unless explicitly requested.

## UPDATE RULE

After each verified implementation step:

1. Mark completed work.
2. Record relevant verification.
3. Update current/next area if changed.
4. Record important known issues.
5. Do not record speculative completion.