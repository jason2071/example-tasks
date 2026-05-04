# example-tasks

> Task management REST API — Go + Fiber v2 + PostgreSQL, no ORM.

## What it does

Provides a CRUD API for tasks with cursor-based pagination, priority filtering, and soft deletes. Built as a reference implementation of the Handler → Service → Repository pattern with explicit dependency injection.

## Requirements

- Go 1.24+
- PostgreSQL 15+ (schema `example` created by migration)

## Quick Start

```bash
git clone <repo-url>
cd example-tasks

# 1. Apply the migration
psql -U postgres -d <your_db> -f migrations/0001_create_table_tasks.up.sql

# 2. Set DB connection
export DATABASE__HOST=localhost
export DATABASE__USER=postgres
export DATABASE__PASSWORD=secret
export DATABASE__DBNAME=plms

# 3. Run
go run .
# Fiber listens on :3000
```

Verify:

```bash
curl http://localhost:3000/
# Hello, World!
```

## Configuration

Config is embedded from `config/config.yaml` at build time. Override any key with environment variables using `__` (double underscore) as the key separator. A `.env` file in the project root is loaded automatically in local environments.

| Env variable | config.yaml key | Default | Description |
|---|---|---|---|
| `DATABASE__HOST` | `database.host` | _(empty)_ | PostgreSQL host |
| `DATABASE__PORT` | `database.port` | `5432` | PostgreSQL port |
| `DATABASE__USER` | `database.user` | _(empty)_ | PostgreSQL user |
| `DATABASE__PASSWORD` | `database.password` | _(empty)_ | PostgreSQL password |
| `DATABASE__DBNAME` | `database.dbname` | _(empty)_ | Database name |
| `DATABASE__SSLMODE` | `database.sslmode` | `disable` | `disable` / `require` / `verify-full` |

Example `.env`:

```
DATABASE__HOST=localhost
DATABASE__USER=postgres
DATABASE__PASSWORD=secret
DATABASE__DBNAME=plms
```

## Project Structure

```
main.go                      # Fiber setup, DI wiring: repo → service → handler
handler/
└── task_handler.go          # HTTP layer: parse, validate, call service, return JSON
service/
└── task_service.go          # Business logic, secondary validation, error mapping
repository/
└── task_repository.go       # Raw SQL via database/sql + lib/pq
model/
├── task_model.go            # Task struct, TaskRequest DTO, PagedResponse, Pagination
└── config.go                # AppConfig, DatabaseConfig structs
utils/
├── errors_response.go       # AppError type, standard error codes, HandleError()
├── errors.go                # Sentinel errors (ErrTaskNotFound200, ErrTaskAlreadyExists200)
└── validator.go             # ValidationError type, human-readable tag messages
config/
├── config.go                # Viper loader with embedded YAML + env override
└── config.yaml              # Default configuration (embedded at build time)
migrations/
├── 0001_create_table_tasks.up.sql    # Create example.tasks + indexes + constraints
└── 0001_create_table_tasks.down.sql  # DROP TABLE example.tasks
test/e2e/                    # End-to-end tests against live PostgreSQL
```

## Common Tasks

```bash
go run .                        # Start server on :3000

make test-all                   # Unit tests + e2e tests
make test-unit                  # Unit tests only (no DB required)
make test-e2e                   # All e2e tests (requires live DB)

make test-e2e-get-task-by-id    # Run TestGetTaskByIDE2E only
make test-e2e-update-task       # Run TestUpdateTaskE2E only
make test-e2e-delete-task       # Run TestDeleteTaskE2E only

# Single test by name
go test ./service/... -run TestCreateTask -v -count=1
```

E2E tests connect using the same env variables as the server, with optional `E2E_DB_*` overrides for CI isolation:

| Env | Fallback | Default |
|---|---|---|
| `E2E_DB_HOST` | `DATABASE__HOST` | `localhost` |
| `E2E_DB_PORT` | `DATABASE__PORT` | `5432` |
| `E2E_DB_USER` | `DATABASE__USER` | `postgres` |
| `E2E_DB_PASSWORD` | `DATABASE__PASSWORD` | _(empty)_ |
| `E2E_DB_NAME` | `DATABASE__DBNAME` | `plms` |
| `E2E_DB_SSLMODE` | `DATABASE__SSLMODE` | `disable` |

Each e2e test truncates and re-seeds `example.tasks` before running, so tests are isolated.

## Database

- Schema: `example`, table: `example.tasks`
- **Soft deletes:** all read queries filter `WHERE deleted_at IS NULL`. Delete sets `deleted_at = CURRENT_TIMESTAMP`.
- **Unique title:** enforced by a partial index `WHERE deleted_at IS NULL`, so a deleted title can be reused.
- **`priority`:** nullable `int`. Valid range 1–5 enforced by a CHECK constraint.
- **`status`:** `varchar(20)` with CHECK constraint — allowed values: `pending`, `doing`, `done`.

To roll back the migration:

```bash
psql -U postgres -d <your_db> -f migrations/0001_create_table_tasks.down.sql
```

---

## API Reference

Base URL: `http://localhost:3000`

All responses are JSON. Errors include a human-readable `error` field. Application-layer errors also include a `code` field.

### Error format

```json
{
  "error": "resource not found",
  "code": "E001",
  "path": "/task/99"
}
```

| Code | HTTP status | Meaning |
|---|---|---|
| `E001` | 404 | Resource not found |
| `E002` | 400 | Invalid request data |
| `E003` | 400 | Duplicate title (active task with same title exists) |
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

| Field | Type | Required | Constraints |
|---|---|---|---|
| `title` | string | yes | Unique among active (non-deleted) tasks |
| `status` | string | yes | `pending` \| `doing` \| `done` |
| `priority` | integer | no | 1–5 |

**201 Created**

```json
{
  "message": "Task created successfully"
}
```

**400 Bad Request** — validation failure

```json
{
  "error": "Validation failed",
  "details": [
    { "field": "Status", "message": "This field is required" }
  ],
  "path": "/task"
}
```

**409 Conflict** — title already in use by an active task

```json
{
  "error": "task with the same title already exists",
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
| `cursor` | integer | `0` | ID of the last item on the previous page. `0` = first page. |
| `size` | integer | `20` | Min 1 |
| `priority` | integer | `0` | `0` = no filter; `1`–`5` = exact match |
| `sort_with` | string | `id` | `id` \| `priority` \| `title` |
| `sort_by` | string | `asc` | `asc` \| `desc` |

Cursor pagination: `pagination.next_cursor` is the ID of the last item returned. Pass it as `cursor` on the next request. A `next_cursor` of `0` means the page was empty.

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
      },
      {
        "id": 2,
        "title": "Deploy to staging",
        "status": "doing",
        "created_at": "2026-04-23T10:05:00Z",
        "updated_at": "2026-04-23T10:05:00Z",
        "priority": null
      }
    ],
    "pagination": {
      "next_cursor": 2,
      "page_size": 2
    }
  }
}
```

**400 Bad Request** — invalid query parameters

```json
{
  "error": "Invalid sort parameters",
  "path": "/tasks"
}
```

---

### GET /task/:id

Get a single task by ID.

```http
GET /task/1 HTTP/1.1
```

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
{
  "error": "resource not found"
}
```

---

### PATCH /task/:id

Partially update a task. Only `status` and `priority` are written to the database; `title` is parsed but not updated. `updated_at` is always refreshed on a successful update.

```http
PATCH /task/1 HTTP/1.1
Content-Type: application/json

{
  "status": "done",
  "priority": 5
}
```

| Field | Type | Required | Constraints |
|---|---|---|---|
| `status` | string | no | `pending` \| `doing` \| `done` |
| `priority` | integer | no | 1–5 |

**200 OK**

```json
{
  "message": "Task updated successfully"
}
```

**400 Bad Request** — invalid `:id` or malformed body

```json
{
  "error": "Invalid task ID",
  "path": "/task/abc"
}
```

**404 Not Found**

```json
{
  "error": "resource not found",
  "code": "E001",
  "path": "/task/99999"
}
```

---

### DELETE /task/:id

Soft-delete a task. Sets `deleted_at = CURRENT_TIMESTAMP`; the row remains in the database. A deleted task's title can be reused by a new task.

```http
DELETE /task/1 HTTP/1.1
```

**200 OK**

```json
{
  "message": "Task deleted successfully"
}
```

**400 Bad Request** — invalid `:id`

```json
{
  "error": "Invalid task ID",
  "path": "/task/abc"
}
```

**404 Not Found** — task does not exist or was already deleted

```json
{
  "error": "resource not found",
  "code": "E001",
  "path": "/task/99999"
}
```

---

## Architecture

```
HTTP request
     |
     v
  handler/       Parse request, validate input (go-playground/validator),
                 call service, write JSON response
     |
     v
  service/       Business logic, secondary validation, map error codes
                 to *AppError
     |
     v
  repository/    Raw SQL (database/sql + lib/pq), returns string error
                 code + Go error
     |
     v
  PostgreSQL     example.tasks
```

Dependency injection is wired manually in `main.go`:

```go
taskRepo    := repository.NewTaskRepository(db)
taskService := service.NewTaskService(taskRepo)
taskHandler := handler.NewTaskHandler(taskService)
```

Both `TaskRepository` and `TaskService` are interfaces, making unit testing with mocks straightforward (see `service/task_service_test.go`).

### Error propagation

1. Repository returns `("E001", nil)` or `("E500", err)`.
2. Service calls `utils.GetAppErrorByCode(code)` to get an `*AppError`.
3. Handler calls `utils.HandleError(c, err)`, which maps `AppError.ErrorCode` to an HTTP status and writes the JSON body.
