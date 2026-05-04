package service

import (
	"errors"
	"example-tasks/model"
	"example-tasks/utils"
	"testing"
)

// mockTaskRepository implements repository.TaskRepository for testing.
type mockTaskRepository struct {
	createTaskFunc func(task model.TaskRequest) (string, error)
}

func (m *mockTaskRepository) CreateTask(task model.TaskRequest) (string, error) {
	return m.createTaskFunc(task)
}

func (m *mockTaskRepository) GetTasks(cursor int64, size, priority int, sortWith, sortBy string) ([]model.Task, error) {
	panic("not implemented")
}

func (m *mockTaskRepository) GetTaskByID(id int64) (model.Task, error) {
	panic("not implemented")
}

func (m *mockTaskRepository) UpdateTask(id int64, task model.TaskRequest) (string, error) {
	panic("not implemented")
}

func (m *mockTaskRepository) DeleteTask(id int64) (string, error) {
	panic("not implemented")
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

var errDBFailed = errors.New("db connection failed")

func TestCreateTask(t *testing.T) {
	validRequest := model.TaskRequest{
		Title:    strPtr("Test Task"),
		Status:   strPtr("pending"),
		Priority: intPtr(2),
	}

	tests := []struct {
		name      string
		request   model.TaskRequest
		repoCode  string
		repoErr   error
		wantErr   error
		wantNoErr bool
	}{
		{
			name:      "success",
			request:   validRequest,
			repoCode:  utils.Success.ErrorCode,
			repoErr:   nil,
			wantNoErr: true,
		},
		{
			name:     "repo returns internal error",
			request:  validRequest,
			repoCode: utils.ErrInternalServer.ErrorCode,
			repoErr:  errDBFailed,
			wantErr:  errDBFailed,
		},
		{
			name:     "duplicate entry via error code",
			request:  validRequest,
			repoCode: utils.ErrDuplicateEntry.ErrorCode,
			repoErr:  nil,
			wantErr:  utils.ErrTaskAlreadyExists200,
		},
		{
			name:     "not found error code from repo",
			request:  validRequest,
			repoCode: utils.ErrNotFound.ErrorCode,
			repoErr:  nil,
			wantErr:  utils.ErrNotFound,
		},
		{
			name:     "invalid request error code from repo",
			request:  validRequest,
			repoCode: utils.ErrInvalidRequest.ErrorCode,
			repoErr:  nil,
			wantErr:  utils.ErrInvalidRequest,
		},
		{
			name:     "unknown error code from repo",
			request:  validRequest,
			repoCode: "TOTALLY_UNKNOWN",
			repoErr:  nil,
			wantErr:  nil, // non-nil AppError with UNKNOWN code — just check non-nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTaskRepository{
				createTaskFunc: func(task model.TaskRequest) (string, error) {
					return tt.repoCode, tt.repoErr
				},
			}

			svc := NewTaskService(mock)
			err := svc.CreateTask(tt.request)

			if tt.wantNoErr {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// For unknown error code case: just verify error is non-nil.
			if tt.wantErr == nil {
				return
			}

			if err != tt.wantErr {
				t.Errorf("got error %v; want %v", err, tt.wantErr)
			}
		})
	}
}
