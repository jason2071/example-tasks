package handler

import (
	"errors"
	"example-tasks/service"
	"log"
	"strconv"

	"example-tasks/model"

	"example-tasks/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type TaskHandlerImpl struct {
	taskService service.TaskService
}

func NewTaskHandler(taskService service.TaskService) *TaskHandlerImpl {
	return &TaskHandlerImpl{
		taskService: taskService,
	}
}

var validate = validator.New()

func (h *TaskHandlerImpl) CreateTask(c *fiber.Ctx) error {
	path := c.Path()

	var taskRequest model.TaskRequest
	if err := c.BodyParser(&taskRequest); err != nil {
		log.Println("Invalid request body:", err)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"path":  path,
		})
	}

	if err := validate.Struct(&taskRequest); err != nil {
		var errors []utils.ValidationError
		for _, err := range err.(validator.ValidationErrors) {
			var element utils.ValidationError
			element.Field = err.Field()
			element.Message = utils.MsgForTag(err.Tag(), err.Param())
			errors = append(errors, element)
		}

		log.Println("Validation errors:", errors)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": errors,
			"path":    path,
		})
	}

	err := h.taskService.CreateTask(taskRequest)
	if err != nil {
		if errors.Is(err, utils.ErrTaskAlreadyExists200) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": utils.ErrTaskAlreadyExists200.Error(),
				"code":  "E003",
				"path":  path,
			})
		}

		if handled := utils.HandleError(c, err); handled != nil {
			return handled
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create task",
			"path":  path,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Task created successfully",
	})
}

func (h *TaskHandlerImpl) GetTasks(c *fiber.Ctx) error {
	path := c.Path()

	cursor, err := strconv.ParseInt(c.Query("cursor", "0"), 10, 64)
	if err != nil || cursor < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid cursor",
			"path":  path,
		})
	}

	size, err := strconv.Atoi(c.Query("size", "20"))
	if err != nil || size < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid size",
			"path":  path,
		})
	}

	priority, err := strconv.Atoi(c.Query("priority", "0"))
	if err != nil || priority < 0 || priority > 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid priority",
			"path":  path,
		})
	}

	sortWith := c.Query("sort_with", "id")
	sortBy := c.Query("sort_by", "asc")

	// Validate sort parameters
	allowedSortFields := map[string]bool{"id": true, "priority": true, "title": true}
	allowedSortOrders := map[string]bool{"asc": true, "desc": true}

	if !allowedSortFields[sortWith] || !allowedSortOrders[sortBy] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid sort parameters",
			"path":  path,
		})
	}

	tasks, err := h.taskService.GetTasks(cursor, size, priority, sortWith, sortBy)
	if err != nil {
		if handled := utils.HandleError(c, err); handled != nil {
			return handled
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve tasks",
			"path":  path,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"tasks": tasks,
	})
}

func (h *TaskHandlerImpl) GetTaskByID(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)

	task, err := h.taskService.GetTaskByID(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve task",
		})
	}

	if task.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": utils.ErrTaskNotFound200.Error(),
			"code":  "E001",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"task": task,
	})
}

func (h *TaskHandlerImpl) UpdateTask(c *fiber.Ctx) error {
	path := c.Path()
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		log.Println("Invalid task ID format:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
			"path":  path,
		})
	}

	var taskRequest model.TaskRequest
	if err := c.BodyParser(&taskRequest); err != nil {
		log.Println("Invalid request body:", err)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"path":  path,
		})
	}

	err = h.taskService.UpdateTask(id, taskRequest)
	if err != nil {
		if _, ok := err.(*utils.AppError); ok {
			_ = utils.HandleError(c, err)
			return nil
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update task",
			"path":  path,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Task updated successfully",
	})
}

func (h *TaskHandlerImpl) DeleteTask(c *fiber.Ctx) error {
	path := c.Path()
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
			"path":  path,
		})
	}

	err = h.taskService.DeleteTask(id)
	if err != nil {
		if _, ok := err.(*utils.AppError); ok {
			_ = utils.HandleError(c, err)
			return nil
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete task",
			"path":  path,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Task deleted successfully",
	})
}
