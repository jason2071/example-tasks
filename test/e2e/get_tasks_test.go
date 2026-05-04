package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTasksE2E_Success(t *testing.T) {
	app, db := setupTestApp(t)

	if _, err := db.Exec(`
		INSERT INTO example.tasks (title, status, priority)
		VALUES
			('task-a', 'pending', 1),
			('task-b', 'doing', 2),
			('task-c', 'done', 3);
	`); err != nil {
		t.Fatalf("failed to seed tasks: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tasks?size=2&sort_with=id&sort_by=asc", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute get tasks request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	payload := decodeJSONBody(t, resp)
	tasksObj, ok := payload["tasks"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tasks object in response, got %#v", payload["tasks"])
	}

	data, ok := tasksObj["data"].([]interface{})
	if !ok {
		t.Fatalf("expected tasks.data array, got %#v", tasksObj["data"])
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 tasks from page size, got %d", len(data))
	}

	firstTask, ok := data[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first task object, got %#v", data[0])
	}
	if got := firstTask["title"]; got != "task-a" {
		t.Fatalf("expected first task title task-a, got %#v", got)
	}

	secondTask, ok := data[1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected second task object, got %#v", data[1])
	}
	if got := secondTask["title"]; got != "task-b" {
		t.Fatalf("expected second task title task-b, got %#v", got)
	}

	pagination, ok := tasksObj["pagination"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tasks.pagination object, got %#v", tasksObj["pagination"])
	}
	if got := pagination["page_size"]; got != float64(2) {
		t.Fatalf("expected page_size 2, got %#v", got)
	}
	if got := pagination["next_cursor"]; got != float64(2) {
		t.Fatalf("expected next_cursor 2, got %#v", got)
	}
}

func TestGetTasksE2E_InvalidSortParams(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/tasks?sort_with=created_at&sort_by=asc", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute get tasks request with invalid sort params: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	payload := decodeJSONBody(t, resp)
	if got := payload["error"]; got != "Invalid sort parameters" {
		t.Fatalf("expected error message Invalid sort parameters, got %#v", got)
	}
}
