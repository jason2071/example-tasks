package service

import (
	"errors"
	"testing"

	"example-tasks/internal/model"
	"example-tasks/internal/repository/mocks"
	"example-tasks/internal/utils"

	"github.com/stretchr/testify/mock"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func sampleRequest() model.TaskRequest {
	return model.TaskRequest{
		Title:    strPtr("buy milk"),
		Status:   strPtr("pending"),
		Priority: intPtr(3),
	}
}

func TestCreateTask(t *testing.T) {
	repoErr := errors.New("db down")

	tests := []struct {
		name    string
		code    string
		repoErr error
		wantErr error // exact sentinel expected; nil means no error
	}{
		{name: "success", code: utils.Success.ErrorCode, wantErr: nil},
		{name: "repo returns raw error", code: utils.ErrInternalServer.ErrorCode, repoErr: repoErr, wantErr: repoErr},
		{name: "duplicate maps to AlreadyExists200", code: utils.ErrDuplicateEntry.ErrorCode, wantErr: utils.ErrTaskAlreadyExists200},
		{name: "internal error code falls through", code: utils.ErrInternalServer.ErrorCode, wantErr: utils.ErrInternalServer},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockTaskRepository)
			repo.On("CreateTask", mock.AnythingOfType("model.TaskRequest")).Return(tc.code, tc.repoErr)
			svc := NewTaskService(repo)

			err := svc.CreateTask(sampleRequest())

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("want nil error, got %v", err)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want error %v, got %v", tc.wantErr, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestGetTasks(t *testing.T) {
	rows := []model.Task{{ID: 10}, {ID: 42}}

	tests := []struct {
		name       string
		cursor     int64
		size       int
		priority   int
		sortWith   string
		sortBy     string
		repoRows   []model.Task
		repoErr    error
		repoCalled bool
		wantErr    error
		wantCursor int64
		wantSize   int
	}{
		{name: "negative cursor", cursor: -1, size: 20, sortWith: "id", sortBy: "asc", wantErr: utils.ErrInvalidRequest},
		{name: "priority too high", cursor: 0, size: 20, priority: 6, sortWith: "id", sortBy: "asc", wantErr: utils.ErrInvalidRequest},
		{name: "priority negative", cursor: 0, size: 20, priority: -1, sortWith: "id", sortBy: "asc", wantErr: utils.ErrInvalidRequest},
		{name: "invalid sort field", cursor: 0, size: 20, sortWith: "name", sortBy: "asc", wantErr: utils.ErrInvalidRequest},
		{name: "invalid sort order", cursor: 0, size: 20, sortWith: "id", sortBy: "sideways", wantErr: utils.ErrInvalidRequest},
		{name: "repo error maps to internal", cursor: 0, size: 20, sortWith: "id", sortBy: "asc", repoErr: errors.New("boom"), repoCalled: true, wantErr: utils.ErrInternalServer},
		{name: "empty result yields zero cursor", cursor: 0, size: 20, sortWith: "id", sortBy: "asc", repoRows: nil, repoCalled: true, wantCursor: 0, wantSize: 20},
		{name: "next cursor is last id", cursor: 0, size: 20, sortWith: "id", sortBy: "asc", repoRows: rows, repoCalled: true, wantCursor: 42, wantSize: 20},
		{name: "non-positive size defaults to 20", cursor: 0, size: 0, sortWith: "id", sortBy: "asc", repoRows: rows, repoCalled: true, wantCursor: 42, wantSize: 20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockTaskRepository)
			if tc.repoCalled {
				repo.On("GetTasks", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(tc.repoRows, tc.repoErr)
			}
			svc := NewTaskService(repo)

			res, err := svc.GetTasks(tc.cursor, tc.size, tc.priority, tc.sortWith, tc.sortBy)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want error %v, got %v", tc.wantErr, err)
				}
				if res != nil {
					t.Fatalf("want nil response on error, got %+v", res)
				}
				repo.AssertExpectations(t)
				return
			}

			if err != nil {
				t.Fatalf("want nil error, got %v", err)
			}
			if res.Pagination.NextCursor != tc.wantCursor {
				t.Errorf("next_cursor = %d, want %d", res.Pagination.NextCursor, tc.wantCursor)
			}
			if res.Pagination.PageSize != tc.wantSize {
				t.Errorf("page_size = %d, want %d", res.Pagination.PageSize, tc.wantSize)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestGetTaskByID(t *testing.T) {
	found := model.Task{ID: 7, Title: "found"}

	tests := []struct {
		name    string
		repoT   model.Task
		repoErr error
		wantErr bool
		wantID  int64
	}{
		{name: "repo error is wrapped", repoErr: errors.New("conn reset"), wantErr: true},
		{name: "not found returns empty task no error", repoT: model.Task{ID: 0}, wantID: 0},
		{name: "found returns task", repoT: found, wantID: 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockTaskRepository)
			repo.On("GetTaskByID", int64(1)).Return(tc.repoT, tc.repoErr)
			svc := NewTaskService(repo)

			task, err := svc.GetTaskByID(1)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil")
				}
				repo.AssertExpectations(t)
				return
			}
			if err != nil {
				t.Fatalf("want nil error, got %v", err)
			}
			if task.ID != tc.wantID {
				t.Errorf("task.ID = %d, want %d", task.ID, tc.wantID)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUpdateTask(t *testing.T) {
	tests := []struct {
		name       string
		id         int64
		code       string
		repoErr    error
		repoCalled bool
		wantErr    error
	}{
		{name: "invalid id", id: 0, wantErr: utils.ErrInvalidRequest},
		{name: "repo error maps to internal", id: 1, repoErr: errors.New("boom"), repoCalled: true, wantErr: utils.ErrInternalServer},
		{name: "not found code", id: 1, code: utils.ErrNotFound.ErrorCode, repoCalled: true, wantErr: utils.ErrNotFound},
		{name: "duplicate code", id: 1, code: utils.ErrDuplicateEntry.ErrorCode, repoCalled: true, wantErr: utils.ErrDuplicateEntry},
		{name: "success", id: 1, code: utils.Success.ErrorCode, repoCalled: true, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockTaskRepository)
			if tc.repoCalled {
				repo.On("UpdateTask", tc.id, mock.AnythingOfType("model.TaskRequest")).
					Return(tc.code, tc.repoErr)
			}
			svc := NewTaskService(repo)

			err := svc.UpdateTask(tc.id, sampleRequest())

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("want nil error, got %v", err)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want error %v, got %v", tc.wantErr, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestDeleteTask(t *testing.T) {
	tests := []struct {
		name       string
		id         int64
		code       string
		repoErr    error
		repoCalled bool
		wantErr    error
	}{
		{name: "invalid id", id: 0, wantErr: utils.ErrInvalidRequest},
		{name: "repo error maps to internal", id: 1, repoErr: errors.New("boom"), repoCalled: true, wantErr: utils.ErrInternalServer},
		{name: "not found code", id: 1, code: utils.ErrNotFound.ErrorCode, repoCalled: true, wantErr: utils.ErrNotFound},
		{name: "success", id: 1, code: utils.Success.ErrorCode, repoCalled: true, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockTaskRepository)
			if tc.repoCalled {
				repo.On("DeleteTask", tc.id).Return(tc.code, tc.repoErr)
			}
			svc := NewTaskService(repo)

			err := svc.DeleteTask(tc.id)

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("want nil error, got %v", err)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want error %v, got %v", tc.wantErr, err)
			}
			repo.AssertExpectations(t)
		})
	}
}
