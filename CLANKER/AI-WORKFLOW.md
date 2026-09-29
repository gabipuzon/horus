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
- Database-backed repository tests connect to `postgres://horus:horus@localhost:5432/horus` and require the SQL migrations to be applied. `docker compose up -d` starts local PostgreSQL and Redis; the app itself does not apply migrations.
- Review the final diff and keep changes within the requested scope.

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
