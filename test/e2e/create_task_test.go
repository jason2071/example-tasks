package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTaskE2E_SuccessAndDuplicate(t *testing.T) {
	app, db := setupTestApp(t)

	testDataPath := filepath.Join("..", "..", "testdata", "create_task.json")
	body, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/task", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute create task request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	respPayload := decodeJSONBody(t, resp)
	if got := respPayload["message"]; got != "Task created successfully" {
		t.Fatalf("expected success message, got %#v", got)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM example.tasks WHERE title = 'e2e-create-task'`).Scan(&count); err != nil {
		t.Fatalf("failed to verify DB row count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 row in DB, got %d", count)
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/task", strings.NewReader(string(body)))
	dupReq.Header.Set("Content-Type", "application/json")

	dupResp, err := app.Test(dupReq, -1)
	if err != nil {
		t.Fatalf("failed to execute duplicate create task request: %v", err)
	}

	if dupResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, dupResp.StatusCode)
	}

	dupPayload := decodeJSONBody(t, dupResp)
	if got := dupPayload["error"]; got != "task with the same title already exists" {
		t.Fatalf("expected duplicate error message, got %#v", got)
	}
	if got := dupPayload["path"]; got != "/task" {
		t.Fatalf("expected response path /task, got %#v", got)
	}
}

func decodeJSONBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()

	defer resp.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON response body: %v", err)
	}

	return body
}
