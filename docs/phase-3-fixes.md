# Phase 3: Audit driven fixes

**Status:** Implemented and locally verified (2026-11-18); CI pending

## Goal

Improve maintainability and address a bounded set of correctness issues identified by the audit, after the structural migration is stable.

## Changes

- Discard queued jobs for monitors that are disabled when a worker loads them.
- Make tests that require PostgreSQL or Redis opt in for local runs while keeping full integration coverage in CI.
- Review the new architecture and development documentation when applying the fixes.

Phase 2 already made integration tests configurable through the existing `HORUS_DB_*` and `HORUS_REDIS_*` variables and fixed HTTP listen failures to exit unsuccessfully. Phase 3 uses `HORUS_TEST_INTEGRATION=1` to opt into service-dependent tests locally; CI sets this flag. Workers discard a queued job when its monitor is disabled at lookup time. Larger queue delivery, scheduling, or check and incident consistency redesigns remain deferred as described in the phase 1 audit.

## Verification

The disabled queued-job test and a default `go test ./...` run passed. The default run skips tests requiring external PostgreSQL or Redis. An isolated PostgreSQL 17 and Redis 8 stack on ports 55432/56379 applied all five migrations; with `HORUS_TEST_INTEGRATION=1`, both `go test ./...` and `go test -race ./...` passed. `go vet ./...`, `go build ./...`, and `git diff --check` also passed. The isolated containers and volumes were removed afterward. CI retains the full integration run through the same flag; its frontend and image checks have not been rerun for this backend-only phase.

## Complete when

Approved fixes have focused tests, the full CI checks pass, and deferred risks are documented with a clear reason for deferral.
