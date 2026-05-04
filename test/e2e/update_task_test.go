package e2e

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateTaskE2E_Success(t *testing.T) {
	app, db := setupTestApp(t)

	if _, err := db.Exec(`
		INSERT INTO example.tasks (title, status, priority)
		VALUES ('task-to-update', 'pending', 1);
	`); err != nil {
		t.Fatalf("failed to seed task for update: %v", err)
	}

	updateBody := `{"title":"updated-task","status":"done","priority":5}`
	req := httptest.NewRequest(http.MethodPatch, "/task/1", strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute update task request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	payload := decodeJSONBody(t, resp)
	if got := payload["message"]; got != "Task updated successfully" {
		t.Fatalf("expected success message, got %#v", got)
	}

	var title, status string
	var priority int
	if err := db.QueryRow(`
		SELECT title, status, priority
		FROM example.tasks
		WHERE id = 1
	`).Scan(&title, &status, &priority); err != nil {
		t.Fatalf("failed to verify updated task in DB: %v", err)
	}

	if title != "updated-task" || status != "done" || priority != 5 {
		t.Fatalf("unexpected DB values after update: title=%s status=%s priority=%d", title, status, priority)
	}
}

func TestUpdateTaskE2E_NotFound(t *testing.T) {
	app, _ := setupTestApp(t)

	updateBody := `{"title":"does-not-matter"}`
	req := httptest.NewRequest(http.MethodPatch, "/task/99999", strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute update task request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	payload := decodeJSONBody(t, resp)
	if got := payload["error"]; got != "resource not found" {
		t.Fatalf("expected error message resource not found, got %#v", got)
	}
	if got := payload["code"]; got != "E001" {
		t.Fatalf("expected error code E001, got %#v", got)
	}
	if got := payload["path"]; got != "/task/99999" {
		t.Fatalf("expected path /task/99999, got %#v", got)
	}
}
