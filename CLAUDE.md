# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run server
go run .

# Run all tests
make test-all

# Run unit tests only (excludes e2e)
make test-unit

# Run e2e tests (requires live DB)
make test-e2e

# Run single test by name
go test ./service/... -run TestGetTaskByID -v -count=1

# Run specific e2e test
make test-e2e-get-task-by-id   # or test-e2e-update-task, test-e2e-delete-task
```

## Architecture

Handler → Service → Repository layered architecture with explicit DI wired in `main.go`.

```
main.go          # Fiber setup, DI wiring: repo → service → handler
handler/         # HTTP layer: parse request, call service, return JSON
service/         # Business logic, validation, error mapping
repository/      # Raw SQL via database/sql + lib/pq (PostgreSQL)
model/           # Request DTOs, Task struct, pagination types
utils/           # AppError type, error codes, HandleError(), validator mapping
config/          # Viper-based YAML config with env override support
migrations/      # SQL DDL (example.tasks table)
test/e2e/        # End-to-end tests against live server
```

**Interfaces**: `TaskRepository` and `TaskService` defined for DI; handler depends on service interface, service depends on repository interface.

## Error Handling

Custom `AppError` in `utils/errors_response.go` with standardized codes:
- `E001` = not found → 404
- `E002` = invalid request → 400
- `E003` = duplicate → 409
- `E500` = server error → 500

`utils.HandleError(ctx, err)` maps `AppError` → HTTP status automatically. All layers return typed errors; handlers never construct error responses directly.

## Database

- PostgreSQL via `database/sql` + `lib/pq`. No ORM.
- Soft deletes: all queries filter `deleted_at IS NULL`.
- Keyset pagination by task ID (cursor-based), supports `sort_with`/`sort_by` query params.
- Config via `config/config.yaml` (embedded); env overrides use double-underscore separator (e.g. `DATABASE__HOST`).

## API

5 REST endpoints on Fiber v2:
- `POST /task` — create (201)
- `GET /tasks` — list with cursor pagination
- `GET /task/:id` — get by ID
- `PATCH /task/:id` — partial update
- `DELETE /task/:id` — soft delete

Priority field: integer 1–5.

## Agents

Use specialist agents for complex tasks. Invoke via `Agent` tool with `subagent_type`.

| Task | Agent |
|------|-------|
| New endpoint / service / repository | `backend-developer` |
| Unit or table-driven tests | `test-writer` |
| PR / diff / function review | `code-reviewer` |
| Schema design or migration | `database-designer` |
| Slow query / EXPLAIN analysis | `sql-optimizer` |
| Refactor, decouple, reduce duplication | `refactor-specialist` |
| Full feature (handler + service + repo + test) | `tech-lead` (orchestrates sub-agents) |
| Explore unknown files or search codebase | `Explore` |

**When to parallelize**: run independent agents in single message (e.g., `test-writer` + `code-reviewer` simultaneously after implementing a feature).
