package handler

import (
	"log"
	"net/http"

	"example-tasks/internal/model"
	"example-tasks/internal/service"
	"example-tasks/internal/utils"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	Service service.HealthService
	AppInfo model.AppInfo
}

func NewHealthHandler(svc service.HealthService, appInfo model.AppInfo) *HealthHandler {
	return &HealthHandler{Service: svc, AppInfo: appInfo}
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (h *HealthHandler) HealthCheck(c *gin.Context) {
	if err := h.Service.Ping(c.Request.Context()); err != nil {
		log.Printf("health check ping failed: %v", err)
		utils.HandleError(c, utils.ErrInternalServer)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"database": "connected",
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	if err := h.Service.Ping(c.Request.Context()); err != nil {
		log.Printf("health check ping failed: %v", err)
		utils.HandleError(c, utils.ErrInternalServer)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (h *HealthHandler) Info(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":     h.AppInfo.Name,
		"version":     h.AppInfo.Version,
		"description": h.AppInfo.Description,
		"environment": h.AppInfo.Environment,
		"status":      "running",
	})
}
