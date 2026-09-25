# README-AI.md

## SYSTEM

- Purpose: bootstrap AI agents into the Horus development context.
- Authority: Files 1-5 are read-only context; File 6 is the only writable context file.
- Rule: Read this index first, then load Files 2-6 as required.
- Goal: preserve architectural consistency, development state, and AI behavior across sessions.
- Project: Horus — Go-based monitoring platform / Pingdom clone.

## CONTEXT FILES

| # | File | Access | Role |
|---|---|---|---|
| 1 | `README-AI.md` | READ | System index / bootstrap |
| 2 | `PROJECT-OVERVIEW.md` | READ | Goals, scope, domain, requirements |
| 3 | `ARCHITECTURE.md` | READ | Architecture, components, data flow, tech stack |
| 4 | `CODE-STANDARDS.md` | READ | Go conventions, structure, testing, naming |
| 5 | `AI-WORKFLOW.md` | READ | AI behavior, workflow, decision rules |
| 6 | `PROGRESS.md` | READ/WRITE | Current implementation state; only writable context file |

## LOAD ORDER

1. Read `README-AI.md`.
2. Read `PROJECT-OVERVIEW.md`.
3. Read `ARCHITECTURE.md`.
4. Read `CODE-STANDARDS.md`.
5. Read `AI-WORKFLOW.md`.
6. Read `PROGRESS.md`.
7. Inspect repository code before modifying it.
8. Update `PROGRESS.md` only after confirmed implementation changes.

## FILE CONTRACTS

### `PROJECT-OVERVIEW.md`
- Project purpose
- Product scope
- Functional requirements
- Non-goals
- Domain concepts
- Target deployment/use case

### `ARCHITECTURE.md`
- System architecture
- Component boundaries
- Data flow
- Database model
- Async/worker model
- External dependencies
- Technology choices
- Architectural constraints

### `CODE-STANDARDS.md`
- Go conventions
- Package structure
- Naming
- Error handling
- Interfaces
- Testing
- API conventions
- Database conventions
- Security/reliability rules

### `AI-WORKFLOW.md`
- Implementation workflow
- Task decomposition
- Context-loading rules
- Change boundaries
- Verification requirements
- Git/commit conventions
- Decision-making rules
- Forbidden behaviors
- Progress-update rules

### `PROGRESS.md`
- Completed work
- Current task
- Next task
- Known issues
- Technical debt
- Tests/status
- Git state
- Only file AI may modify within the context system

## GLOBAL RULES

- Preserve existing architecture unless a change is explicitly justified.
- Prefer incremental, testable changes.
- Do not invent project state; inspect the repository.
- Do not rewrite unrelated code.
- Verify changes with appropriate tests before marking work complete.
- Keep context files internally consistent.
- Treat `PROGRESS.md` as the source of implementation-state truth.
- Do not modify Files 1-5.
- When context conflicts with actual repository state, inspect code first and document the discrepancy in `PROGRESS.md`.
