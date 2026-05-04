package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"example-tasks/model"
	"example-tasks/utils"

	"github.com/gofiber/fiber/v2"
)

// ---------------------------------------------------------------------------
// Manual mock for service.TaskService
// ---------------------------------------------------------------------------

type mockTaskService struct {
	createTaskFunc  func(task model.TaskRequest) error
	getTasksFunc    func(cursor int64, size, priority int, sortWith, sortBy string) (*model.PagedResponse, error)
	getTaskByIDFunc func(id int64) (model.Task, error)
	updateTaskFunc  func(id int64, task model.TaskRequest) error
	deleteTaskFunc  func(id int64) error
}

func (m *mockTaskService) CreateTask(task model.TaskRequest) error {
	return m.createTaskFunc(task)
}

func (m *mockTaskService) GetTasks(cursor int64, size, priority int, sortWith, sortBy string) (*model.PagedResponse, error) {
	return m.getTasksFunc(cursor, size, priority, sortWith, sortBy)
}

func (m *mockTaskService) GetTaskByID(id int64) (model.Task, error) {
	return m.getTaskByIDFunc(id)
}

func (m *mockTaskService) UpdateTask(id int64, task model.TaskRequest) error {
	return m.updateTaskFunc(id, task)
}

func (m *mockTaskService) DeleteTask(id int64) error {
	return m.deleteTaskFunc(id)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func newApp(svc *mockTaskService) *fiber.App {
	app := fiber.New()
	h := NewTaskHandler(svc)
	app.Post("/task", h.CreateTask)
	app.Get("/tasks", h.GetTasks)
	app.Get("/task/:id", h.GetTaskByID)
	app.Patch("/task/:id", h.UpdateTask)
	app.Delete("/task/:id", h.DeleteTask)
	return app
}

func doRequest(app *fiber.App, method, url string, body interface{}) (*http.Response, map[string]interface{}) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, url, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, _ := app.Test(req, -1)

	var result map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &result)

	return resp, result
}

// ---------------------------------------------------------------------------
// CreateTask
// ---------------------------------------------------------------------------

func TestCreateTask_Handler(t *testing.T) {
	validBody := map[string]interface{}{
		"title":    "Buy groceries",
		"status":   "pending",
		"priority": 3,
	}

	tests := []struct {
		name           string
		body           interface{}
		rawBody        string
		useSvcFunc     func(task model.TaskRequest) error
		wantStatus     int
		wantMessageKey string
		wantErrorKey   string
	}{
		{
			name: "happy path - task created",
			body: validBody,
			useSvcFunc: func(task model.TaskRequest) error {
				return nil
			},
			wantStatus:     fiber.StatusCreated,
			wantMessageKey: "message",
		},
		{
			name:       "invalid body - malformed JSON",
			rawBody:    `{bad json`,
			wantStatus: fiber.StatusBadRequest,
			wantErrorKey: "error",
		},
		{
			name: "validation error - missing title",
			body: map[string]interface{}{
				"status":   "pending",
				"priority": 2,
			},
			wantStatus:   fiber.StatusBadRequest,
			wantErrorKey: "error",
		},
		{
			name: "validation error - missing status",
			body: map[string]interface{}{
				"title":    "My task",
				"priority": 1,
			},
			wantStatus:   fiber.StatusBadRequest,
			wantErrorKey: "error",
		},
		{
			name: "validation error - priority out of range",
			body: map[string]interface{}{
				"title":    "My task",
				"status":   "pending",
				"priority": 10,
			},
			wantStatus:   fiber.StatusBadRequest,
			wantErrorKey: "error",
		},
		{
			name: "service error - duplicate task",
			body: validBody,
			useSvcFunc: func(task model.TaskRequest) error {
				return utils.ErrTaskAlreadyExists200
			},
			wantStatus:   fiber.StatusConflict,
			wantErrorKey: "error",
		},
		{
			name: "service error - AppError E500",
			body: validBody,
			useSvcFunc: func(task model.TaskRequest) error {
				return utils.ErrInternalServer
			},
			wantStatus:   fiber.StatusInternalServerError,
			wantErrorKey: "error",
		},
		{
			// CreateTask handler only intercepts ErrTaskAlreadyExists200 explicitly;
			// for other AppErrors it calls HandleError but ignores the return value,
			// so it falls through to the generic 500 branch.
			name: "service error - AppError E002 falls through to 500",
			body: validBody,
			useSvcFunc: func(task model.TaskRequest) error {
				return utils.ErrInvalidRequest
			},
			wantStatus:   fiber.StatusInternalServerError,
			wantErrorKey: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTaskService{
				createTaskFunc: tt.useSvcFunc,
			}
			app := newApp(svc)

			var resp *http.Response
			var result map[string]interface{}

			if tt.rawBody != "" {
				req := httptest.NewRequest("POST", "/task", bytes.NewBufferString(tt.rawBody))
				req.Header.Set("Content-Type", "application/json")
				resp, _ = app.Test(req, -1)
				b, _ := io.ReadAll(resp.Body)
				_ = json.Unmarshal(b, &result)
			} else {
				resp, result = doRequest(app, "POST", "/task", tt.body)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d; want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantMessageKey != "" {
				if _, ok := result[tt.wantMessageKey]; !ok {
					t.Errorf("response missing key %q; got %v", tt.wantMessageKey, result)
				}
			}
			if tt.wantErrorKey != "" {
				if _, ok := result[tt.wantErrorKey]; !ok {
					t.Errorf("response missing key %q; got %v", tt.wantErrorKey, result)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetTasks
// ---------------------------------------------------------------------------

func TestGetTasks_Handler(t *testing.T) {
	successSvc := func(cursor int64, size, priority int, sortWith, sortBy string) (*model.PagedResponse, error) {
		return &model.PagedResponse{
			Data:       []model.Task{{ID: 1, Title: "Task 1", Status: "pending"}},
			Pagination: model.Pagination{NextCursor: 1, PageSize: size},
		}, nil
	}

	tests := []struct {
		name        string
		query       string
		svcFunc     func(cursor int64, size, priority int, sortWith, sortBy string) (*model.PagedResponse, error)
		wantStatus  int
		wantDataKey string
	}{
		{
			name:        "happy path - default params",
			query:       "/tasks",
			svcFunc:     successSvc,
			wantStatus:  fiber.StatusOK,
			wantDataKey: "tasks",
		},
		{
			name:        "happy path - explicit params",
			query:       "/tasks?cursor=5&size=10&priority=3&sort_with=priority&sort_by=desc",
			svcFunc:     successSvc,
			wantStatus:  fiber.StatusOK,
			wantDataKey: "tasks",
		},
		{
			name:       "invalid cursor - negative",
			query:      "/tasks?cursor=-1",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid cursor - non-numeric",
			query:      "/tasks?cursor=abc",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid size - zero",
			query:      "/tasks?size=0",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid size - non-numeric",
			query:      "/tasks?size=big",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid priority - above 5",
			query:      "/tasks?priority=6",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid priority - negative",
			query:      "/tasks?priority=-1",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid sort_with field",
			query:      "/tasks?sort_with=bogus",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid sort_by order",
			query:      "/tasks?sort_by=random",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "service error - AppError E500",
			query: "/tasks",
			svcFunc: func(cursor int64, size, priority int, sortWith, sortBy string) (*model.PagedResponse, error) {
				return nil, utils.ErrInternalServer
			},
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTaskService{
				getTasksFunc: tt.svcFunc,
			}
			app := newApp(svc)
			resp, result := doRequest(app, "GET", tt.query, nil)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d; want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantDataKey != "" {
				if _, ok := result[tt.wantDataKey]; !ok {
					t.Errorf("response missing key %q; got %v", tt.wantDataKey, result)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetTaskByID
// ---------------------------------------------------------------------------

func TestGetTaskByID_Handler(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		svcFunc     func(id int64) (model.Task, error)
		wantStatus  int
		wantTaskKey string
	}{
		{
			name: "happy path - task found",
			url:  "/task/1",
			svcFunc: func(id int64) (model.Task, error) {
				return model.Task{ID: 1, Title: "Task 1", Status: "pending"}, nil
			},
			wantStatus:  fiber.StatusOK,
			wantTaskKey: "task",
		},
		{
			name: "not found - service returns zero-ID task",
			url:  "/task/99",
			svcFunc: func(id int64) (model.Task, error) {
				return model.Task{}, nil // ID == 0 → not found branch
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "service error - DB failure",
			url:  "/task/1",
			svcFunc: func(id int64) (model.Task, error) {
				return model.Task{}, utils.ErrInternalServer
			},
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTaskService{
				getTaskByIDFunc: tt.svcFunc,
			}
			app := newApp(svc)
			resp, result := doRequest(app, "GET", tt.url, nil)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d; want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantTaskKey != "" {
				if _, ok := result[tt.wantTaskKey]; !ok {
					t.Errorf("response missing key %q; got %v", tt.wantTaskKey, result)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateTask
// ---------------------------------------------------------------------------

func TestUpdateTask_Handler(t *testing.T) {
	validBody := map[string]interface{}{
		"title":    "Updated title",
		"status":   "done",
		"priority": 1,
	}

	tests := []struct {
		name         string
		url          string
		body         interface{}
		rawBody      string
		svcFunc      func(id int64, task model.TaskRequest) error
		wantStatus   int
		wantRespKey  string
	}{
		{
			name: "happy path - task updated",
			url:  "/task/1",
			body: validBody,
			svcFunc: func(id int64, task model.TaskRequest) error {
				return nil
			},
			wantStatus:  fiber.StatusOK,
			wantRespKey: "message",
		},
		{
			name:       "invalid ID - zero",
			url:        "/task/0",
			body:       validBody,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid ID - non-numeric",
			url:        "/task/abc",
			body:       validBody,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid body - malformed JSON",
			url:        "/task/1",
			rawBody:    `{bad`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service error - not found (AppError E001)",
			url:  "/task/1",
			body: validBody,
			svcFunc: func(id int64, task model.TaskRequest) error {
				return utils.ErrNotFound
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "service error - invalid request (AppError E002)",
			url:  "/task/1",
			body: validBody,
			svcFunc: func(id int64, task model.TaskRequest) error {
				return utils.ErrInvalidRequest
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service error - internal server (AppError E500)",
			url:  "/task/1",
			body: validBody,
			svcFunc: func(id int64, task model.TaskRequest) error {
				return utils.ErrInternalServer
			},
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTaskService{
				updateTaskFunc: tt.svcFunc,
			}
			app := newApp(svc)

			var resp *http.Response
			var result map[string]interface{}

			if tt.rawBody != "" {
				req := httptest.NewRequest("PATCH", tt.url, bytes.NewBufferString(tt.rawBody))
				req.Header.Set("Content-Type", "application/json")
				resp, _ = app.Test(req, -1)
				b, _ := io.ReadAll(resp.Body)
				_ = json.Unmarshal(b, &result)
			} else {
				resp, result = doRequest(app, "PATCH", tt.url, tt.body)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d; want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantRespKey != "" {
				if _, ok := result[tt.wantRespKey]; !ok {
					t.Errorf("response missing key %q; got %v", tt.wantRespKey, result)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DeleteTask
// ---------------------------------------------------------------------------

func TestDeleteTask_Handler(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		svcFunc     func(id int64) error
		wantStatus  int
		wantRespKey string
	}{
		{
			name: "happy path - task deleted",
			url:  "/task/1",
			svcFunc: func(id int64) error {
				return nil
			},
			wantStatus:  fiber.StatusOK,
			wantRespKey: "message",
		},
		{
			name:       "invalid ID - zero",
			url:        "/task/0",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid ID - non-numeric",
			url:        "/task/abc",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service error - not found (AppError E001)",
			url:  "/task/1",
			svcFunc: func(id int64) error {
				return utils.ErrNotFound
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "service error - invalid request (AppError E002)",
			url:  "/task/1",
			svcFunc: func(id int64) error {
				return utils.ErrInvalidRequest
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service error - internal server (AppError E500)",
			url:  "/task/1",
			svcFunc: func(id int64) error {
				return utils.ErrInternalServer
			},
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTaskService{
				deleteTaskFunc: tt.svcFunc,
			}
			app := newApp(svc)
			resp, result := doRequest(app, "DELETE", tt.url, nil)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d; want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantRespKey != "" {
				if _, ok := result[tt.wantRespKey]; !ok {
					t.Errorf("response missing key %q; got %v", tt.wantRespKey, result)
				}
			}
		})
	}
}
