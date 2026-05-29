package router

import (
	"example-tasks/internal/handler"

	"github.com/gofiber/fiber/v2"
)

// Setup wires all HTTP routes onto the Fiber app.
func Setup(app *fiber.App, taskHandler *handler.TaskHandlerImpl, healthHandler *handler.HealthHandler) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/live", healthHandler.Live)
	app.Get("/ready", healthHandler.Ready)
	app.Get("/info", healthHandler.Info)

	app.Post("/task", taskHandler.CreateTask)
	app.Get("/tasks", taskHandler.GetTasks)
	app.Get("/task/:id", taskHandler.GetTaskByID)
	app.Patch("/task/:id", taskHandler.UpdateTask)
	app.Delete("/task/:id", taskHandler.DeleteTask)
}
