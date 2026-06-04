package handler

import (
	"errors"
	"example-tasks/internal/service"
	"log"
	"net/http"
	"strconv"

	"example-tasks/internal/model"

	"example-tasks/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

func (h *TaskHandlerImpl) CreateTask(c *gin.Context) {
	path := c.Request.URL.Path

	var taskRequest model.TaskRequest
	if err := c.ShouldBindJSON(&taskRequest); err != nil {
		log.Println("Invalid request body:", err)

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"path":  path,
		})
		return
	}

	if err := validate.Struct(&taskRequest); err != nil {
		var errs []utils.ValidationError
		for _, err := range err.(validator.ValidationErrors) {
			var element utils.ValidationError
			element.Field = err.Field()
			element.Message = utils.MsgForTag(err.Tag(), err.Param())
			errs = append(errs, element)
		}

		log.Println("Validation errors:", errs)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": errs,
			"path":    path,
		})
		return
	}

	err := h.taskService.CreateTask(taskRequest)
	if err != nil {
		if errors.Is(err, utils.ErrTaskAlreadyExists200) {
			c.JSON(http.StatusConflict, gin.H{
				"error": utils.ErrTaskAlreadyExists200.Error(),
				"code":  "E003",
				"path":  path,
			})
			return
		}

		if utils.HandleError(c, err) {
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create task",
			"path":  path,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully",
	})
}

func (h *TaskHandlerImpl) GetTasks(c *gin.Context) {
	path := c.Request.URL.Path

	cursor, err := strconv.ParseInt(c.DefaultQuery("cursor", "0"), 10, 64)
	if err != nil || cursor < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid cursor",
			"path":  path,
		})
		return
	}

	size, err := strconv.Atoi(c.DefaultQuery("size", "20"))
	if err != nil || size < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid size",
			"path":  path,
		})
		return
	}

	priority, err := strconv.Atoi(c.DefaultQuery("priority", "0"))
	if err != nil || priority < 0 || priority > 5 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid priority",
			"path":  path,
		})
		return
	}

	sortWith := c.DefaultQuery("sort_with", "id")
	sortBy := c.DefaultQuery("sort_by", "asc")

	// Validate sort parameters
	allowedSortFields := map[string]bool{"id": true, "priority": true, "title": true}
	allowedSortOrders := map[string]bool{"asc": true, "desc": true}

	if !allowedSortFields[sortWith] || !allowedSortOrders[sortBy] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid sort parameters",
			"path":  path,
		})
		return
	}

	tasks, err := h.taskService.GetTasks(cursor, size, priority, sortWith, sortBy)
	if err != nil {
		if utils.HandleError(c, err) {
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve tasks",
			"path":  path,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
	})
}

func (h *TaskHandlerImpl) GetTaskByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	task, err := h.taskService.GetTaskByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve task",
		})
		return
	}

	if task.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": utils.ErrTaskNotFound200.Error(),
			"code":  "E001",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task": task,
	})
}

func (h *TaskHandlerImpl) UpdateTask(c *gin.Context) {
	path := c.Request.URL.Path
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		log.Println("Invalid task ID format:", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid task ID",
			"path":  path,
		})
		return
	}

	var taskRequest model.TaskRequest
	if err := c.ShouldBindJSON(&taskRequest); err != nil {
		log.Println("Invalid request body:", err)

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"path":  path,
		})
		return
	}

	err = h.taskService.UpdateTask(id, taskRequest)
	if err != nil {
		if _, ok := err.(*utils.AppError); ok {
			utils.HandleError(c, err)
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update task",
			"path":  path,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task updated successfully",
	})
}

func (h *TaskHandlerImpl) DeleteTask(c *gin.Context) {
	path := c.Request.URL.Path
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid task ID",
			"path":  path,
		})
		return
	}

	err = h.taskService.DeleteTask(id)
	if err != nil {
		if _, ok := err.(*utils.AppError); ok {
			utils.HandleError(c, err)
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete task",
			"path":  path,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task deleted successfully",
	})
}
