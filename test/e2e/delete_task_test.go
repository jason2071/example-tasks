package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteTaskE2E_Success(t *testing.T) {
	app, db := setupTestApp(t)

	if _, err := db.Exec(`
		INSERT INTO example.tasks (title, status, priority)
		VALUES ('task-to-delete', 'doing', 3);
	`); err != nil {
		t.Fatalf("failed to seed task for delete: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/task/1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute delete task request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	payload := decodeJSONBody(t, resp)
	if got := payload["message"]; got != "Task deleted successfully" {
		t.Fatalf("expected success message, got %#v", got)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM example.tasks WHERE id = 1 AND deleted_at IS NOT NULL`).Scan(&count); err != nil {
		t.Fatalf("failed to verify soft delete in DB: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected row to be soft deleted, got count: %d", count)
	}
}

func TestDeleteTaskE2E_NotFound(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/task/99999", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute delete task request: %v", err)
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
