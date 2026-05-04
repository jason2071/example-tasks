# example-tasks

> Task management REST API — Go + Fiber v2 + PostgreSQL, no ORM.

## What it does

CRUD API for tasks with cursor-based pagination, priority filter, soft deletes, and basic health endpoints. Reference implementation of the Handler → Service → Repository pattern with explicit dependency injection.

## Requirements

- Go 1.24+
- PostgreSQL 15+

## Quick Start

```bash
git clone <repo-url>
cd example-tasks

# 1. Create the schema (migration assumes it exists)
psql -U postgres -d <your_db> -c "CREATE SCHEMA IF NOT EXISTS example;"

# 2. Apply the migration
psql -U postgres -d <your_db> -f migrations/0001_create_table_tasks.up.sql

# 3. Set DB connection
export DATABASE__HOST=localhost
export DATABASE__USER=postgres
export DATABASE__PASSWORD=secret
export DATABASE__DBNAME=plms

# 4. Run
go run .
# Fiber listens on :3000
```

Verify:

```bash
curl http://localhost:3000/         # Hello, World!
curl http://localhost:3000/live     # {"status":"ok"}
curl http://localhost:3000/ready    # {"status":"ok"}  (also pings DB)
curl http://localhost:3000/info     # service metadata from config.yaml
```

## Configuration

Defaults are embedded from `config/config.yaml` at build time (`//go:embed`). Override any key with environment variables using `__` (double underscore) as the key separator. A `.env` file in the project root is loaded automatically (via `gotenv`) for local dev.

| Env variable | config.yaml key | Default | Notes |
|---|---|---|---|
| `DATABASE__HOST` | `database.host` | _(empty)_ | Required |
| `DATABASE__PORT` | `database.port` | `5432` | |
| `DATABASE__USER` | `database.user` | _(empty)_ | Required |
| `DATABASE__PASSWORD` | `database.password` | _(empty)_ | |
| `DATABASE__DBNAME` | `database.dbname` | _(empty)_ | Required |
| `DATABASE__SSLMODE` | `database.sslmode` | `disable` | **Currently ignored** — `main.go` hardcodes `sslmode=disable` in the DSN. |

App metadata under `app:` (`name`, `version`, `description`, `environment`) is surfaced by `GET /info`.

Example `.env`:

```
DATABASE__HOST=localhost
DATABASE__USER=postgres
DATABASE__PASSWORD=secret
DATABASE__DBNAME=plms
```

## Project Structure

```
main.go                          # Fiber setup, DI: db -> repo -> service -> handler
handler/
├── task_handler.go              # /task* CRUD
└── health_handler.go            # /live, /ready, /info
service/
├── task_service.go              # TaskService interface + impl
└── health_service.go            # DB ping
repository/
└── task_repository.go           # TaskRepository interface + raw SQL
model/
├── task_model.go                # Task, TaskRequest (pointer fields), Pagination, PagedResponse
└── config.go                    # AppConfig, AppInfo, DatabaseConfig
utils/
├── errors_response.go           # *AppError, GetAppErrorByCode, HandleError
├── errors.go                    # Sentinels: ErrTaskNotFound200, ErrTaskAlreadyExists200
└── validator.go                 # ValidationError, MsgForTag
config/
├── config.go                    # Viper loader, embedded YAML + env override
└── config.yaml                  # Default config (embedded at build time)
migrations/
├── 0001_create_table_tasks.up.sql
└── 0001_create_table_tasks.down.sql
```

> **Note:** there are no `*_test.go` files in the repo as of commit `676844c` (test files were removed). The `make test-*` targets and `test/e2e/` directory referenced in `Makefile` are stubs — re-add tests before relying on them.

## Common Tasks

```bash
go run .                        # Start server on :3000

make test-unit                  # No tests currently — exits with "no test files"
make test-e2e                   # ./test/e2e directory does not yet exist
make test-all                   # both above

# Single test by name (once tests exist)
go test ./service/... -run TestCreateTask -v -count=1
```

E2E tests (when added) connect using `E2E_DB_*` overrides in `Makefile`/test setup, falling back to `DATABASE__*`:

| Env | Fallback | Default |
|---|---|---|
| `E2E_DB_HOST` | `DATABASE__HOST` | `localhost` |
| `E2E_DB_PORT` | `DATABASE__PORT` | `5432` |
| `E2E_DB_USER` | `DATABASE__USER` | `postgres` |
| `E2E_DB_PASSWORD` | `DATABASE__PASSWORD` | _(empty)_ |
| `E2E_DB_NAME` | `DATABASE__DBNAME` | `plms` |
| `E2E_DB_SSLMODE` | `DATABASE__SSLMODE` | `disable` |

Each e2e test should truncate and re-seed `example.tasks` for isolation.

## Database

- Schema: `example`, table: `example.tasks`. The migration does **not** create the schema — run `CREATE SCHEMA IF NOT EXISTS example;` first.
- **Soft deletes:** all read queries filter `WHERE deleted_at IS NULL`. `DELETE` sets `deleted_at = CURRENT_TIMESTAMP`.
- **Unique title:** enforced by partial index `WHERE deleted_at IS NULL`, so a deleted title can be reused.
- **`status`:** `varchar(20)`, `DEFAULT 'pending'`, CHECK in (`pending`, `doing`, `done`). **Not validated at the struct level** — invalid values surface as DB errors, not 400 validation responses.
- **`priority`:** nullable `int`, CHECK 1–5.

To roll back the migration:

```bash
psql -U postgres -d <your_db> -f migrations/0001_create_table_tasks.down.sql
```

---

## API Reference

Base URL: `http://localhost:3000`. All responses JSON.

### Error format

```json
{
  "error": "resource not found",
  "code": "E001",
  "path": "/task/99"
}
```

| Code | HTTP status (via `HandleError`) | Meaning |
|---|---|---|
| `E001` | 404 | Resource not found |
| `E002` | 400 | Invalid request data |
| `E003` | 400 | Duplicate title — **except** `POST /task`, which short-circuits to **409** before `HandleError` runs |
| `E500` | 500 | Internal server error |

Validation errors on `POST /task` return 400 with a `details` array instead of `code`:

```json
{
  "error": "Validation failed",
  "details": [
    { "field": "Title", "message": "This field is required" },
    { "field": "Priority", "message": "Should be at least 1 characters" }
  ],
  "path": "/task"
}
```

Validator tag → message map lives in `utils/validator.go` (`MsgForTag`). Only `required`, `email`, `min`, `max`, `oneof` are mapped — anything else returns `"Invalid value"`.

---

### Health & Info

#### `GET /live`

Liveness probe. Always returns 200 if the process is up.

```json
{ "status": "ok" }
```

#### `GET /ready`

Readiness probe. Pings the database; returns 500 (`E500`) if the ping fails.

```json
{ "status": "ok" }
```

#### `GET /info`

App metadata sourced from `config.yaml` `app:` section.

```json
{
  "service": "example-tasks",
  "version": "1.0.0",
  "description": "Go + Fiber v2 + PostgreSQL REST API ...",
  "environment": "development",
  "status": "running"
}
```

---

### POST /task

Create a new task.

```http
POST /task HTTP/1.1
Content-Type: application/json

{
  "title": "Write unit tests",
  "status": "pending",
  "priority": 2
}
```

| Field | Type | Required | Validation | Constraints |
|---|---|---|---|---|
| `title` | string | yes | `required` | Unique among active (non-deleted) tasks |
| `status` | string | yes | `required` (enum enforced by DB CHECK only) | `pending` \| `doing` \| `done` |
| `priority` | integer | no | `omitempty,min=1,max=5` | 1–5 |

**201 Created**

```json
{ "message": "Task created successfully" }
```

**400 Bad Request** — validation failure (see error format above).

**409 Conflict** — title already in use by an active task

```json
{
  "error": "task with the same title already exists",
  "code": "E003",
  "path": "/task"
}
```

---

### GET /tasks

List tasks with cursor-based pagination. Only returns rows where `deleted_at IS NULL`.

```http
GET /tasks?size=2&cursor=0&sort_with=id&sort_by=asc HTTP/1.1
```

| Query param | Type | Default | Constraints |
|---|---|---|---|
| `cursor` | integer | `0` | ID of last item on previous page; `0` = first page |
| `size` | integer | `20` | Min 1 |
| `priority` | integer | `0` | `0` = no filter; `1`–`5` = exact match |
| `sort_with` | string | `id` | `id` \| `priority` \| `title` |
| `sort_by` | string | `asc` | `asc` \| `desc` |

Cursor pagination: `pagination.next_cursor` is the ID of the last item returned. Pass it as `cursor` on the next request. `next_cursor = 0` means the page was empty.

**200 OK**

```json
{
  "tasks": {
    "data": [
      {
        "id": 1,
        "title": "Write unit tests",
        "status": "pending",
        "created_at": "2026-04-23T10:00:00Z",
        "updated_at": "2026-04-23T10:00:00Z",
        "priority": 2
      }
    ],
    "pagination": {
      "next_cursor": 1,
      "page_size": 2
    }
  }
}
```

**400 Bad Request** — invalid query parameters

```json
{ "error": "Invalid sort parameters", "path": "/tasks" }
```

---

### GET /task/:id

Get a single task by ID.

**200 OK**

```json
{
  "task": {
    "id": 1,
    "title": "Write unit tests",
    "status": "pending",
    "created_at": "2026-04-23T10:00:00Z",
    "updated_at": "2026-04-23T10:00:00Z",
    "priority": 2
  }
}
```

**404 Not Found** — task does not exist or is soft-deleted

```json
{ "error": "resource not found", "code": "E001" }
```

---

### PATCH /task/:id

Partially update a task. `updated_at` is always refreshed.

```http
PATCH /task/1 HTTP/1.1
Content-Type: application/json

{ "status": "done", "priority": 5 }
```

| Field | Type | Required | Constraints |
|---|---|---|---|
| `title` | string | no | Written only if non-empty; subject to active-title unique index |
| `status` | string | no | DB CHECK: `pending` \| `doing` \| `done` |
| `priority` | integer | no | DB CHECK: 1–5 |

> ⚠️ **Side effect**: the repository's UPDATE statement always sets `deleted_at = NULL`, and the `WHERE` clause does **not** filter `deleted_at IS NULL`. PATCHing a soft-deleted task will **resurrect** it.

**200 OK**

```json
{ "message": "Task updated successfully" }
```

**400 Bad Request** — invalid `:id`, malformed body, or duplicate title (E003 falls through `HandleError` here, unlike POST).

**404 Not Found**

```json
{ "error": "resource not found", "code": "E001", "path": "/task/99999" }
```

---

### DELETE /task/:id

Soft-delete a task. Sets `deleted_at = CURRENT_TIMESTAMP`; the row remains. A deleted task's title can be reused.

**200 OK**

```json
{ "message": "Task deleted successfully" }
```

**400 Bad Request** — invalid `:id`.

**404 Not Found** — task does not exist or was already deleted.

---

## Architecture

```
HTTP request
     │
     ▼
  handler/       Parse + validate (go-playground/validator), call service,
                 write JSON response
     │
     ▼
  service/       Business logic, secondary validation, map repo error codes
                 → *AppError or sentinel error
     │
     ▼
  repository/    Raw SQL (database/sql + lib/pq), returns (errorCode string, error)
     │
     ▼
  PostgreSQL     example.tasks
```

Dependency injection is wired manually in `main.go`:

```go
taskRepo    := repository.NewTaskRepository(db)
taskService := service.NewTaskService(taskRepo)
taskHandler := handler.NewTaskHandler(taskService)

healthSvc     := service.NewHealthService(db)
healthHandler := handler.NewHealthHandler(healthSvc, appConfig.AppInfo)
```

Both `TaskRepository` and `TaskService` are interfaces, so unit tests can swap in mocks.

### Error propagation

1. Repository returns `(errorCode, err)` — e.g. `("E001", nil)`, `("E003", nil)`, `("E500", err)`.
2. Service calls `utils.GetAppErrorByCode(code)` to map to an `*AppError`.
   - `CreateTask` further translates `ErrDuplicateEntry` → sentinel `ErrTaskAlreadyExists200` so the handler can return **409**.
3. Handler calls `utils.HandleError(c, err)`, which maps `AppError.ErrorCode` → HTTP status and writes the JSON body.

### Known inconsistencies

- `POST /task` returns **409** for duplicate title; `PATCH /task/:id` returns **400** for the same condition because `HandleError` maps `E003` → 400.
- `GET /task/:id` does not return an `AppError` for "not found" — service returns an empty `Task{}`, handler converts ID=0 to 404 inline.
- `database.sslmode` is in the config but `main.go` hardcodes `sslmode=disable` in the DSN, so the value is unused.
- `repository.UpdateTask` always sets `deleted_at = NULL` and lacks a `deleted_at IS NULL` WHERE filter — soft-deleted tasks can be resurrected via PATCH.
