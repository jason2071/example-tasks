package router

import (
	"net/http"

	"example-tasks/internal/handler"

	"github.com/gin-gonic/gin"
)

// Setup wires all HTTP routes onto the Gin engine.
func Setup(app *gin.Engine, taskHandler *handler.TaskHandlerImpl, healthHandler *handler.HealthHandler) {
	app.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	app.GET("/live", healthHandler.Live)
	app.GET("/ready", healthHandler.Ready)
	app.GET("/info", healthHandler.Info)

	app.POST("/task", taskHandler.CreateTask)
	app.GET("/tasks", taskHandler.GetTasks)
	app.GET("/task/:id", taskHandler.GetTaskByID)
	app.PATCH("/task/:id", taskHandler.UpdateTask)
	app.DELETE("/task/:id", taskHandler.DeleteTask)
}
