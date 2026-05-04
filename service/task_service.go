package service

import (
	"example-tasks/model"
	"example-tasks/repository"
	"example-tasks/utils"
	"fmt"
)

type TaskService interface {
	CreateTask(task model.TaskRequest) error
	GetTasks(cursor int64, size, priority int, sortWith, sortBy string) (*model.PagedResponse, error)
	GetTaskByID(id int64) (model.Task, error)
	UpdateTask(id int64, task model.TaskRequest) error
	DeleteTask(id int64) error
}

type TaskServiceImpl struct {
	taskRepository repository.TaskRepository
}

func NewTaskService(taskRepository repository.TaskRepository) *TaskServiceImpl {
	return &TaskServiceImpl{
		taskRepository: taskRepository,
	}
}

func (s *TaskServiceImpl) CreateTask(task model.TaskRequest) error {
	errorCode, err := s.taskRepository.CreateTask(task)
	if err != nil {
		return err
	}

	appErr := utils.GetAppErrorByCode(errorCode)
	if appErr == utils.ErrDuplicateEntry {
		return utils.ErrTaskAlreadyExists200
	}

	if appErr != utils.Success {
		return appErr
	}

	return nil
}

func (s *TaskServiceImpl) GetTasks(cursor int64, size, priority int, sortWith, sortBy string) (*model.PagedResponse, error) {
	if cursor < 0 || priority < 0 || priority > 5 {
		return nil, utils.ErrInvalidRequest
	}

	allowedSortFields := map[string]bool{"id": true, "priority": true, "title": true}
	allowedSortOrders := map[string]bool{"asc": true, "desc": true}
	if !allowedSortFields[sortWith] || !allowedSortOrders[sortBy] {
		return nil, utils.ErrInvalidRequest
	}

	if size <= 0 {
		size = 20
	}

	tasks, err := s.taskRepository.GetTasks(cursor, size, priority, sortWith, sortBy)
	if err != nil {
		return nil, utils.ErrInternalServer
	}

	nextCursor := int64(0)
	if len(tasks) > 0 {
		nextCursor = tasks[len(tasks)-1].ID
	}

	return &model.PagedResponse{
		Data: tasks,
		Pagination: model.Pagination{
			NextCursor: nextCursor,
			PageSize:   size,
		},
	}, nil
}

func (s *TaskServiceImpl) GetTaskByID(id int64) (model.Task, error) {

	task, err := s.taskRepository.GetTaskByID(id)
	if err != nil {
		return model.Task{}, fmt.Errorf("failed to retrieve task: %v", err)
	}

	if task.ID == 0 {
		return model.Task{}, nil
	}

	return task, nil
}

func (s *TaskServiceImpl) UpdateTask(id int64, task model.TaskRequest) error {
	if id < 1 {
		return utils.ErrInvalidRequest
	}

	errorCode, err := s.taskRepository.UpdateTask(id, task)
	if err != nil {
		return utils.ErrInternalServer
	}

	appErr := utils.GetAppErrorByCode(errorCode)
	if appErr != utils.Success {
		return appErr
	}
	return nil
}

func (s *TaskServiceImpl) DeleteTask(id int64) error {
	if id < 1 {
		return utils.ErrInvalidRequest
	}

	errorCode, err := s.taskRepository.DeleteTask(id)
	if err != nil {
		return utils.ErrInternalServer
	}

	appErr := utils.GetAppErrorByCode(errorCode)
	if appErr != utils.Success {
		return appErr
	}

	return nil

}
