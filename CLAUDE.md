# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run server (listens :3000)
go run .

# Build
go build -o example-tasks .

# Tests — NOTE: all *_test.go files were deleted (commit 676844c) and the
# Makefile was removed (commit 13ab01a). Re-add tests before relying on these:
go test ./...                                        # all packages
go test ./service/... -run TestGetTaskByID -v -count=1  # single test by name

# DB setup (required before first run)
psql -U postgres -d <db> -c "CREATE SCHEMA IF NOT EXISTS example;"
psql -U postgres -d <db> -f migrations/0001_create_table_tasks.up.sql
```

## Architecture

Handler → Service → Repository, manual DI in `main.go`. PostgreSQL via `database/sql` + `lib/pq`, no ORM.

```
main.go                 # Fiber setup, builds db -> repo -> service -> handler
handler/
  task_handler.go       # /task* CRUD
  health_handler.go     # /live /ready /info
service/
  task_service.go       # TaskService interface + impl, validation, error mapping
  health_service.go     # ping checks
repository/
  task_repository.go    # TaskRepository interface + raw SQL
model/
  task_model.go         # Task, TaskRequest (pointer fields for partial update), PagedResponse
  config.go             # AppConfig, DatabaseConfig, AppInfo
utils/
  errors_response.go    # *AppError, GetAppErrorByCode, HandleError (code -> HTTP status)
  errors.go             # Sentinel errors: ErrTaskNotFound200, ErrTaskAlreadyExists200
  validator.go          # ValidationError type, MsgForTag (validator-tag -> message)
config/
  config.go             # Viper loader, embeds config.yaml, env override
  config.yaml           # Embedded at build (//go:embed). Edits need rebuild.
migrations/
  0001_create_table_tasks.up.sql / .down.sql
```

`TaskRepository` and `TaskService` are interfaces — handler/service depend on the interface, enabling mocks.

## Routes (all wired in main.go)

| Method | Path        | Handler                  |
|--------|-------------|--------------------------|
| GET    | `/`         | inline "Hello, World!"   |
| GET    | `/live`     | healthHandler.Live       |
| GET    | `/ready`    | healthHandler.Ready      |
| GET    | `/info`     | healthHandler.Info       |
| POST   | `/task`     | createTask (201)         |
| GET    | `/tasks`    | list w/ cursor pagination|
| GET    | `/task/:id` | getByID                  |
| PATCH  | `/task/:id` | partial update           |
| DELETE | `/task/:id` | soft delete              |

## Error model — read carefully

`AppError{ErrorCode, ErrorMessage}` in `utils/errors_response.go`. Codes:

| Code | Sentinel             | `HandleError` HTTP |
|------|----------------------|--------------------|
| E001 | ErrNotFound          | 404                |
| E002 | ErrInvalidRequest    | 400                |
| E003 | ErrDuplicateEntry    | **400** (not 409)  |
| E500 | ErrInternalServer    | 500                |
| SUCCESS | Success           | n/a                |

**Inconsistency to know**: only `CreateTask` handler short-circuits duplicates to **409** using `ErrTaskAlreadyExists200` (sentinel from `utils/errors.go`) before reaching `HandleError`. `UpdateTask` lets E003 fall through `HandleError` and returns **400**. Same code, different status by route.

`GetTaskByID` returns `model.Task{}` (ID=0) for not-found, no error — handler converts to 404 inline. Other handlers rely on `HandleError`.

## Database

- Schema `example`, table `example.tasks`. Migration must be applied manually (`psql -f migrations/...up.sql`).
- **Soft delete**: reads filter `deleted_at IS NULL`; delete sets `deleted_at = CURRENT_TIMESTAMP`.
- **Unique title**: partial unique index where `deleted_at IS NULL` — deleted titles can be reused.
- **`status`**: `varchar(20)` CHECK in (`pending`, `doing`, `done`).
- **`priority`**: nullable int, CHECK 1–5.
- **Pagination**: keyset by `id` with `cursor` query param. `next_cursor = 0` ⇔ empty page. `sort_with` ∈ {id, priority, title}, `sort_by` ∈ {asc, desc}.

### `UpdateTask` footguns

`repository.UpdateTask` always writes `deleted_at = NULL` in the SET clause — **PATCH on a soft-deleted row will resurrect it** if matched. WHERE has no `deleted_at IS NULL` guard.

Repo also writes `title` when `task.Title != nil && *task.Title != ""` (contradicts README claim that title is parsed-but-not-updated).

## Config

Viper loads embedded `config/config.yaml`; env vars override using `__` separator (`DATABASE__HOST`, `DATABASE__PORT`, `DATABASE__USER`, `DATABASE__PASSWORD`, `DATABASE__DBNAME`, `DATABASE__SSLMODE`). `.env` auto-loaded in local. `sslmode` from config is **ignored** in `main.go` — DSN hardcodes `sslmode=disable`.

## Conventions (from AGENTS.md)

- Layer boundaries strict: handler never touches DB; repo never returns Fiber types.
- Repo returns `(errorCode string, err error)` pair; service maps via `GetAppErrorByCode`.
- Parameterized queries only. No `SELECT *`.
- Validation: `go-playground/validator/v10` on DTOs in handler; secondary checks in service.
