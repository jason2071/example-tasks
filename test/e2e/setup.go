package e2e

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"testing"

	"example-tasks/handler"
	"example-tasks/repository"
	"example-tasks/service"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

func setupTestApp(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()

	db := openTestDB(t)
	ensureSchemaAndTable(t, db)
	cleanDB(t, db)

	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)
	taskHandler := handler.NewTaskHandler(taskService)

	app := fiber.New()
	app.Post("/task", taskHandler.CreateTask)
	app.Get("/tasks", taskHandler.GetTasks)
	app.Get("/task/:id", taskHandler.GetTaskByID)
	app.Patch("/task/:id", taskHandler.UpdateTask)
	app.Delete("/task/:id", taskHandler.DeleteTask)

	t.Cleanup(func() {
		cleanDB(t, db)
		_ = db.Close()
	})

	return app, db
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	host := getEnv("E2E_DB_HOST", getEnv("DATABASE__HOST", "localhost"))
	portStr := getEnv("E2E_DB_PORT", getEnv("DATABASE__PORT", "5432"))
	user := getEnv("E2E_DB_USER", getEnv("DATABASE__USER", "postgres"))
	password := getEnv("E2E_DB_PASSWORD", getEnv("DATABASE__PASSWORD", ""))
	database := getEnv("E2E_DB_NAME", getEnv("DATABASE__DBNAME", "plms"))
	sslmode := getEnv("E2E_DB_SSLMODE", getEnv("DATABASE__SSLMODE", "disable"))

	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("invalid DB port %q: %v", portStr, err)
	}

	pqInfo := fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s",
		host,
		port,
		user,
		database,
		sslmode,
	)
	if password != "" {
		pqInfo += fmt.Sprintf(" password=%s", password)
	}

	db, err := sql.Open("postgres", pqInfo)
	if err != nil {
		t.Fatalf("failed to open test DB connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("failed to ping test DB: %v", err)
	}

	return db
}

func ensureSchemaAndTable(t *testing.T, db *sql.DB) {
	t.Helper()

	bootstrapSQL := `
CREATE SCHEMA IF NOT EXISTS example;
CREATE TABLE IF NOT EXISTS example.tasks (
	id BIGSERIAL PRIMARY KEY,
	title text NOT NULL,
	status varchar(20) DEFAULT 'pending',
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
	updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
	deleted_at timestamptz NULL,
	priority int NULL,
	CONSTRAINT check_campaign_status CHECK (status IN ('pending', 'doing', 'done')),
	CONSTRAINT check_priority_range CHECK (
		priority IS NULL OR (priority >= 1 AND priority <= 5)
	)
);
ALTER TABLE example.tasks ADD COLUMN IF NOT EXISTS deleted_at timestamptz NULL;
DROP INDEX IF EXISTS uq_example_tasks_ref;
CREATE UNIQUE INDEX IF NOT EXISTS uq_example_tasks_title_active ON example.tasks (title)
WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_example_tasks_priority ON example.tasks (priority);
`
	if _, err := db.Exec(bootstrapSQL); err != nil {
		t.Fatalf("failed to bootstrap schema/table: %v", err)
	}
}

func cleanDB(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`TRUNCATE TABLE example.tasks RESTART IDENTITY;`); err != nil {
		t.Fatalf("failed to clean DB: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
