package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example-tasks/internal/model"
	"example-tasks/internal/service/mocks"
	"example-tasks/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

func init() { gin.SetMode(gin.TestMode) }

// newCtx builds a gin context + recorder for a single handler call.
// params lets callers set route params like :id.
func newCtx(method, target, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	return c, w
}

func validBody() string {
	return `{"title":"buy milk","status":"pending","priority":3}`
}

func TestHandlerCreateTask(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		svcCalled  bool
		svcErr     error
		wantStatus int
	}{
		{name: "bad json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "validation fail (missing title)", body: `{"status":"pending"}`, wantStatus: http.StatusBadRequest},
		{name: "duplicate -> 409", body: validBody(), svcCalled: true, svcErr: utils.ErrTaskAlreadyExists200, wantStatus: http.StatusConflict},
		{name: "app error E002 -> 400", body: validBody(), svcCalled: true, svcErr: utils.ErrInvalidRequest, wantStatus: http.StatusBadRequest},
		{name: "internal error -> 500", body: validBody(), svcCalled: true, svcErr: utils.ErrInternalServer, wantStatus: http.StatusInternalServerError},
		{name: "success -> 201", body: validBody(), svcCalled: true, wantStatus: http.StatusCreated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockTaskService(t)
			if tc.svcCalled {
				svc.On("CreateTask", mock.AnythingOfType("model.TaskRequest")).Return(tc.svcErr)
			}
			h := NewTaskHandler(svc)

			c, w := newCtx(http.MethodPost, "/task", tc.body, nil)
			h.CreateTask(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerGetTasks(t *testing.T) {
	paged := &model.PagedResponse{
		Data:       []model.Task{{ID: 1}},
		Pagination: model.Pagination{NextCursor: 1, PageSize: 20},
	}

	tests := []struct {
		name       string
		query      string
		svcCalled  bool
		svcRes     *model.PagedResponse
		svcErr     error
		wantStatus int
	}{
		{name: "invalid cursor", query: "?cursor=-1", wantStatus: http.StatusBadRequest},
		{name: "invalid size", query: "?size=0", wantStatus: http.StatusBadRequest},
		{name: "invalid priority", query: "?priority=9", wantStatus: http.StatusBadRequest},
		{name: "invalid sort field", query: "?sort_with=name", wantStatus: http.StatusBadRequest},
		{name: "service app error -> 500", query: "", svcCalled: true, svcErr: utils.ErrInternalServer, wantStatus: http.StatusInternalServerError},
		{name: "success -> 200", query: "", svcCalled: true, svcRes: paged, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockTaskService(t)
			if tc.svcCalled {
				svc.On("GetTasks", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(tc.svcRes, tc.svcErr)
			}
			h := NewTaskHandler(svc)

			c, w := newCtx(http.MethodGet, "/tasks"+tc.query, "", nil)
			h.GetTasks(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerGetTaskByID(t *testing.T) {
	tests := []struct {
		name       string
		svcTask    model.Task
		svcErr     error
		wantStatus int
	}{
		{name: "service error -> 500", svcErr: utils.ErrInternalServer, wantStatus: http.StatusInternalServerError},
		{name: "not found (ID 0) -> 404", svcTask: model.Task{ID: 0}, wantStatus: http.StatusNotFound},
		{name: "found -> 200", svcTask: model.Task{ID: 7}, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockTaskService(t)
			svc.On("GetTaskByID", int64(7)).Return(tc.svcTask, tc.svcErr)
			h := NewTaskHandler(svc)

			c, w := newCtx(http.MethodGet, "/task/7", "", gin.Params{{Key: "id", Value: "7"}})
			h.GetTaskByID(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerUpdateTask(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		svcCalled  bool
		svcErr     error
		wantStatus int
	}{
		{name: "invalid id", id: "0", body: validBody(), wantStatus: http.StatusBadRequest},
		{name: "bad json", id: "1", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "app error E001 -> 404", id: "1", body: validBody(), svcCalled: true, svcErr: utils.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "non-app error -> 500", id: "1", body: validBody(), svcCalled: true, svcErr: errors.New("boom"), wantStatus: http.StatusInternalServerError},
		{name: "success -> 200", id: "1", body: validBody(), svcCalled: true, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockTaskService(t)
			if tc.svcCalled {
				svc.On("UpdateTask", int64(1), mock.AnythingOfType("model.TaskRequest")).Return(tc.svcErr)
			}
			h := NewTaskHandler(svc)

			c, w := newCtx(http.MethodPatch, "/task/"+tc.id, tc.body, gin.Params{{Key: "id", Value: tc.id}})
			h.UpdateTask(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerDeleteTask(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		svcCalled  bool
		svcErr     error
		wantStatus int
	}{
		{name: "invalid id", id: "0", wantStatus: http.StatusBadRequest},
		{name: "app error E001 -> 404", id: "1", svcCalled: true, svcErr: utils.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "non-app error -> 500", id: "1", svcCalled: true, svcErr: utils.ErrInternalServer, wantStatus: http.StatusInternalServerError},
		{name: "success -> 200", id: "1", svcCalled: true, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockTaskService(t)
			if tc.svcCalled {
				svc.On("DeleteTask", int64(1)).Return(tc.svcErr)
			}
			h := NewTaskHandler(svc)

			c, w := newCtx(http.MethodDelete, "/task/"+tc.id, "", gin.Params{{Key: "id", Value: tc.id}})
			h.DeleteTask(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}
