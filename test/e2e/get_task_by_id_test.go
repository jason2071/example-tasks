package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTaskByIDE2E_Success(t *testing.T) {
	app, db := setupTestApp(t)

	if _, err := db.Exec(`
		INSERT INTO example.tasks (title, status, priority)
		VALUES ('task-by-id', 'doing', 4);
	`); err != nil {
		t.Fatalf("failed to seed task: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/task/1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute get task by id request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	payload := decodeJSONBody(t, resp)
	taskObj, ok := payload["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got %#v", payload["task"])
	}

	if got := taskObj["id"]; got != float64(1) {
		t.Fatalf("expected id 1, got %#v", got)
	}
	if got := taskObj["title"]; got != "task-by-id" {
		t.Fatalf("expected title task-by-id, got %#v", got)
	}
	if got := taskObj["status"]; got != "doing" {
		t.Fatalf("expected status doing, got %#v", got)
	}
	if got := taskObj["priority"]; got != float64(4) {
		t.Fatalf("expected priority 4, got %#v", got)
	}
}

func TestGetTaskByIDE2E_NotFound(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/task/99999", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute get task by id request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	payload := decodeJSONBody(t, resp)
	if got := payload["error"]; got != "resource not found" {
		t.Fatalf("expected error message resource not found, got %#v", got)
	}
}
