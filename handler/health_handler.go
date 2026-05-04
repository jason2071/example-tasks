package handler

import (
	"log"

	"example-tasks/model"
	"example-tasks/service"
	"example-tasks/utils"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	Service service.HealthService
	AppInfo model.AppInfo
}

func NewHealthHandler(svc service.HealthService, appInfo model.AppInfo) *HealthHandler {
	return &HealthHandler{Service: svc, AppInfo: appInfo}
}

func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	if err := h.Service.Ping(c.Context()); err != nil {
		log.Printf("health check ping failed: %v", err)
		return utils.HandleError(c, utils.ErrInternalServer)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":   "ok",
		"database": "connected",
	})
}

func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	if err := h.Service.Ping(c.Context()); err != nil {
		log.Printf("health check ping failed: %v", err)
		return utils.HandleError(c, utils.ErrInternalServer)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *HealthHandler) Info(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"service":     h.AppInfo.Name,
		"version":     h.AppInfo.Version,
		"description": h.AppInfo.Description,
		"environment": h.AppInfo.Environment,
		"status":      "running",
	})
}
