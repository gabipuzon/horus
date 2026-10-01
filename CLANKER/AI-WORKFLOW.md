# AI Workflow

## Before changing code

1. Read the relevant files in `CLANKER/` and inspect current source/tests.
2. Use implementation and tests as the source of truth when docs disagree.
3. Identify the smallest coherent change and preserve established package boundaries.
4. Explain meaningful changes briefly in terms of behavior and reason.

## Implementation and verification

- Follow existing Go style and avoid unrelated rewrites or speculative dependencies.
- Add or update focused tests for meaningful behavior when implementation work calls for them.
- Run `gofmt` on changed Go files. Run relevant tests when requested or when needed to verify the change; report the exact commands and outcome. Do not claim a test passed unless it was run.
- Database-backed repository tests connect to `postgres://horus:horus@localhost:5432/horus` and require applied SQL migrations. For a fresh local stack, `cp .env.example .env` then `docker compose up --build -d` starts PostgreSQL, Redis, a one-shot migration job, Horus, and web at `http://localhost:3000`. For direct Go runs, start only `postgres redis`, export `.env`, run `go run ./cmd/migrate`, then `go run ./cmd/server`. Go binaries do not load `.env` themselves.
- `.github/workflows/ci.yml` runs against fresh PostgreSQL and Redis services: migrate, Go test/race/vet/build, frontend test/lint/build, and both Docker image builds. Keep automated tests independent of public websites and Discord secrets.
- Review the final diff and keep changes within the requested scope.
- For frontend changes, follow `UI-CONTEXT.md`, keep requests in the typed API client, and run the `web/` npm test, build, and lint scripts. Vite proxies same-origin `/api` requests to local Horus; start Horus before browser smoke testing.
- Detail screens should keep metadata, summary, checks, and incident queries independent so a section failure remains local. Check `204` current-incident and no-check summary states against the backend contract.
- For Overview, verify that `/monitors` counts are configuration counts and that capped current-incident requests explicitly report incomplete coverage. Browser passes should visit all three routes at desktop, tablet, and mobile widths and clean up temporary monitors.
- For release packaging, test direct React route refreshes, same-origin `/api` proxying, migration reruns, named-volume persistence across `down`/`up`, and clean Compose shutdown. Use an isolated Compose project for destructive fresh-volume checks.

## Documentation

- Keep implementation state in `PROGRESS.md`; keep product scope in `PROJECT-OVERVIEW.md`; keep runtime/design details in `ARCHITECTURE.md`.
- Distinguish current behavior from plans and known gaps.
- The user may explicitly ask for changes to any `CLANKER` markdown file; follow that request even when an older context rule says a file is read-only.
- Update progress after code implementation is confirmed. Documentation-only work should update the relevant context without inventing code or test changes.

## Decisions and collaboration

- Prefer the simplest design that satisfies current requirements and matches current boundaries.
- Ask for input only when requirements are materially ambiguous, an irreversible architectural choice is needed, or necessary credentials are unavailable. Otherwise make a reasonable, clearly stated choice and continue.
- Do not send external messages, rewrite Git history, or claim deployment/operational state without authorization/evidence.

## Completion

Report the implemented or documented outcome, important files changed, and verification performed. State material limitations plainly. Do not jump into unrelated future work.
