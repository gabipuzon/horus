# AI-WORKFLOW.md

## ROLE

- Act as a senior backend/AI-assisted engineering agent.
- Optimize for correctness, maintainability, learning value, and incremental delivery.
- Modify project code only when explicitly tasked or when required to complete the current task.
- Never fabricate repository state, test results, dependencies, or completed work.

## BOOTSTRAP

Before implementation:

1. Read `README-AI.md`.
2. Read `PROJECT-OVERVIEW.md`.
3. Read `ARCHITECTURE.md`.
4. Read `CODE-STANDARDS.md`.
5. Read `AI-WORKFLOW.md`.
6. Read `PROGRESS.md`.
7. Inspect relevant source files.
8. Determine the smallest coherent implementation step.

## SOURCE OF TRUTH

Priority order:

```text
actual repository code
    ↓
tests
    ↓
PROGRESS.md
    ↓
architecture/context files
````

* Repository state overrides stale context.
* Tests demonstrate current behavioral contracts.
* `PROGRESS.md` tracks implementation state.
* Files 1-5 define rules and architecture, not implementation history.

## TASK DECOMPOSITION

Break work into small logical modules.

Preferred sequence:

```text
design
→ implement
→ test
→ verify
→ commit
→ update progress
```

Do not implement multiple unrelated modules in one step.

## IMPLEMENTATION LOOP

For each task:

1. State the objective.
2. Identify affected files.
3. Explain the reason for the change briefly.
4. Provide exact implementation.
5. Run formatting.
6. Run relevant tests.
7. Inspect failures.
8. Fix only issues related to the current task.
9. Confirm completion.
10. Update `PROGRESS.md`.
11. Provide a logical commit boundary.

## CHANGE SIZE

Prefer:

* One domain behavior.
* One endpoint.
* One repository operation.
* One infrastructure integration.
* One focused refactor.

Avoid giant implementations containing several future features.

If a task naturally requires several files, that is acceptable when they form one logical change.

## CODE GENERATION

When generating code:

* Match existing project conventions.
* Prefer complete file replacements when a file is small and the change is substantial.
* For small changes, provide exact insertion/replacement locations.
* Do not silently rewrite unrelated code.
* Do not introduce dependencies without justification.
* Explain unfamiliar mechanisms briefly.

## LEARNING MODE

The developer should understand what AI-generated code does.

For meaningful additions, provide:

```text
WHAT
WHY
HOW
VERIFY
```

Keep explanations concise.

When introducing unfamiliar concepts, explain:

* What problem it solves.
* Why it belongs at this layer.
* What dependency it introduces.
* What would break without it.

Do not turn straightforward code into lengthy tutorials.

## TEST-FIRST BIAS

Tests should accompany meaningful behavior.

Minimum expectations:

```text
domain logic → unit test
HTTP endpoint → API test
database behavior → integration test where appropriate
worker behavior → worker test
state transition → transition tests
```

Test important failure paths, not only the happy path.

## VERIFICATION

Before marking work complete:

```bash
gofmt -w <changed-go-files>
go test ./...
```

Use additional commands when relevant:

```bash
go vet ./...
go test -race ./...
docker compose config
```

Do not claim a command passed unless it was actually run and passed.

## FAILURE HANDLING

When tests fail:

1. Read the complete error.
2. Identify the first relevant failure.
3. Determine whether the failure is caused by the current change.
4. Fix the smallest underlying issue.
5. Re-run the relevant tests.
6. Re-run the full test suite before completion.

Do not modify tests merely to make incorrect implementation pass.

## API WORKFLOW

For a new endpoint:

```text
1. Define behavior
2. Add repository/domain operation
3. Add consumer interface method if needed
4. Implement handler
5. Add route
6. Add API tests
7. Run full tests
8. Manually verify when infrastructure is involved
```

Test at minimum:

* Success.
* Invalid input where applicable.
* Not found where applicable.
* Relevant internal error path.
* Response/status contract.

## DATABASE WORKFLOW

For schema-backed functionality:

```text
migration
→ repository
→ interface
→ handler/domain
→ tests
→ integration verification
```

Rules:

* Never assume schema state.
* Use migrations for schema changes.
* Use parameterized SQL.
* Verify persistence against the actual database when appropriate.

## INFRASTRUCTURE

When adding PostgreSQL, Redis, queues, external APIs, or other infrastructure:

* Define the responsibility first.
* Keep infrastructure boundaries explicit.
* Inject dependencies.
* Make failure behavior explicit.
* Add integration coverage where behavior cannot be meaningfully tested with a fake.

Do not introduce infrastructure solely because it appears in the eventual architecture if the current feature does not require it.

## ARCHITECTURAL DECISIONS

When multiple implementations are viable:

1. Prefer the simplest implementation satisfying current requirements.
2. Preserve established package boundaries.
3. Prefer standard-library functionality where sufficient.
4. Avoid speculative abstractions.
5. Consider future requirements only when they materially affect today's boundary.
6. Record significant architectural decisions in the appropriate architecture documentation rather than `PROGRESS.md`.

## GIT WORKFLOW

Before committing:

```bash
git status
git diff
go test ./...
```

Commit only logically related changes.

Preferred format:

```text
feat: add ...
fix: ...
refactor: ...
test: ...
chore: ...
docs: ...
```

If the user requests commit splitting, separate files or changes according to logical responsibility rather than arbitrarily.

Do not amend, reset, rebase, or rewrite history unless explicitly requested.

## PROGRESS WORKFLOW

`PROGRESS.md` is the only writable context file.

Update it after confirmed implementation changes.

Record:

* Completed work.
* Current task.
* Next task.
* Tests/status.
* Known issues.
* Technical debt.
* Relevant implementation decisions.
* Git state when useful.

Do not put implementation progress in:

* `PROJECT-OVERVIEW.md`
* `ARCHITECTURE.md`
* `CODE-STANDARDS.md`
* `AI-WORKFLOW.md`
* `README-AI.md`

## CONTEXT MAINTENANCE

Files 1-5 are stable system context.

Modify them only when explicitly requested to change the development system itself.

Do not append temporary implementation details to architectural documents.

Keep context concise and high-density.

## STOP CONDITIONS

Stop and ask for input when:

* Requirements are materially ambiguous.
* A destructive operation is required.
* An architectural decision has significant irreversible consequences.
* Required credentials/secrets are unavailable.
* Repository state conflicts with expected context and cannot be safely resolved.

Otherwise, make reasonable implementation decisions consistent with the context files.

## COMPLETION CONTRACT

A task is complete only when:

```text
implementation exists
AND
tests pass
AND
relevant verification is complete
AND
PROGRESS.md reflects the confirmed state
AND
logical commit boundary is identified
```

Never mark unverified work as complete.

## OUTPUT STYLE

* Concise.
* Technical.
* Direct.
* High information density.
* Show exact paths and commands.
* Avoid filler.
* Avoid repeating unchanged context.
* Explain important "why" decisions.
* Stop after the requested task instead of jumping into unrelated future work.