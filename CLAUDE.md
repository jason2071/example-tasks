# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

A `Makefile` wraps the common tasks (`make help` lists all). Raw commands:

```bash
# Run server (listens :3000)
go run .            # or: make run
make dev            # hot-reload via air (.air.toml builds ./tmp/api, kills :3000 first)

# Build
go build -o example-tasks .   # or: make build  (-> ./tmp/api)

# Tests — NOTE: all *_test.go files were deleted (commit 676844c). Re-add before these work:
go test ./...                                                  # all packages (make test)
go test ./internal/service/... -run TestGetTaskByID -v -count=1  # single test (make test-one PKG=./internal/service NAME=TestGetTaskByID)

# DB setup (required before first run; make db-schema / migrate-up wrap these, override DB= PGUSER=)
psql -U postgres -d <db> -c "CREATE SCHEMA IF NOT EXISTS example;"
psql -U postgres -d <db> -f migrations/0001_create_table_tasks.up.sql
```

Local dev connects via `.env` (Viper env override; e.g. `DATABASE__PORT=5433`, `DATABASE__DBNAME=...`). `.env`, `/example-tasks` binary, `tmp/`, and `graphify-out/` are gitignored.

## Architecture

Handler → Service → Repository, manual DI in `main.go`. PostgreSQL via `database/sql` + `lib/pq`, no ORM. All app packages live under `internal/` (not importable outside the module); module path is `example-tasks`, so imports are `example-tasks/internal/<pkg>`.

```
main.go                 # bootstrap only: LoadConfig -> newDB -> DI wiring -> router.Setup -> Listen(:3000)
internal/
  router/
    router.go           # router.Setup(app, taskHandler, healthHandler) — all routes wired here
  handler/
    task_handler.go     # /task* CRUD
    health_handler.go   # /live /ready /info
  service/
    task_service.go     # TaskService interface + impl, validation, error mapping
    health_service.go   # ping checks
  repository/
    task_repository.go  # TaskRepository interface + raw SQL
  model/
    task_model.go       # Task, TaskRequest (pointer fields for partial update), PagedResponse
    config.go           # AppConfig, DatabaseConfig, AppInfo
  utils/
    errors_response.go  # *AppError, GetAppErrorByCode, HandleError (code -> HTTP status)
    errors.go           # Sentinel errors: ErrTaskNotFound200, ErrTaskAlreadyExists200
    validator.go        # ValidationError type, MsgForTag (validator-tag -> message)
  config/
    config.go           # Viper loader, embeds config.yaml, env override
    config.yaml         # Embedded at build (//go:embed). Edits need rebuild.
migrations/
  0001_create_table_tasks.up.sql / .down.sql
```

`TaskRepository` and `TaskService` are interfaces — handler/service depend on the interface, enabling mocks.

## Routes (all wired in `internal/router/router.go`)

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

`GetTaskByID` returns `model.Task{}` (ID=0) for not-found, no error — handler converts to 404 inline. Other handlers rely on `HandleError`. The handler also ignores the id-parse error (`id, _ := strconv.ParseInt(...)`), so a non-numeric `:id` becomes `0` → 404, not 400.

**`GET /tasks` response shape**: service returns `*model.PagedResponse` (`{data, pagination{next_cursor, page_size}}`), but the handler assigns it to a var named `tasks` and wraps it under the `"tasks"` key. Actual body is `{"tasks": {"data": [...], "pagination": {...}}}` — not a flat array. No total-count field; `repository.taskTotalRecords` exists but is dead code (never called).

## Database

- Schema `example`, table `example.tasks`. Migration must be applied manually (`psql -f migrations/...up.sql`).
- **Soft delete**: reads filter `deleted_at IS NULL`; delete sets `deleted_at = CURRENT_TIMESTAMP`.
- **Unique title**: partial unique index where `deleted_at IS NULL` — deleted titles can be reused.
- **`status`**: `varchar(20)` CHECK in (`pending`, `doing`, `done`).
- **`priority`**: nullable int, CHECK 1–5.
- **Pagination**: keyset by `id` with `cursor` query param. `next_cursor` = last row's `id` (or `0` on empty page); there is no has-more flag, so a full last page still returns a cursor — clients detect the end only by getting an empty next page. `sort_with` ∈ {id, priority, title}, `sort_by` ∈ {asc, desc}.

### `UpdateTask` footguns

`repository.UpdateTask` always writes `deleted_at = NULL` in the SET clause — **PATCH on a soft-deleted row will resurrect it** if matched. WHERE has no `deleted_at IS NULL` guard.

Repo also writes `title` when `task.Title != nil && *task.Title != ""` (contradicts README claim that title is parsed-but-not-updated).

## Config

Viper loads embedded `config/config.yaml`; env vars override using `__` separator (`DATABASE__HOST`, `DATABASE__PORT`, `DATABASE__USER`, `DATABASE__PASSWORD`, `DATABASE__DBNAME`, `DATABASE__SSLMODE`). `.env` auto-loaded in local. `sslmode` from config is **ignored** in `main.go` — DSN hardcodes `sslmode=disable`.

## Conventions

(Originally from `AGENTS.md`, now deleted — these rules live here.)


- Layer boundaries strict: handler never touches DB; repo never returns Fiber types.
- Repo returns `(errorCode string, err error)` pair; service maps via `GetAppErrorByCode`.
- Parameterized queries only. No `SELECT *`.
- Validation: `go-playground/validator/v10` on DTOs in handler; secondary checks in service.

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
