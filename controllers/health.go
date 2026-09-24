package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"rag-document-assistant/services"
)

type HealthController struct {
	postgres services.HealthChecker
	qdrant   services.HealthChecker
}

func NewHealthController(postgres, qdrant services.HealthChecker) *HealthController {
	return &HealthController{postgres: postgres, qdrant: qdrant}
}

func (h *HealthController) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	dependencies := gin.H{"postgres": "up", "qdrant": "up"}
	status := http.StatusOK
	if err := h.postgres.Ping(ctx); err != nil {
		dependencies["postgres"] = "down"
		status = http.StatusServiceUnavailable
	}
	if err := h.qdrant.Ping(ctx); err != nil {
		dependencies["qdrant"] = "down"
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"status": map[bool]string{true: "ok", false: "degraded"}[status == http.StatusOK], "dependencies": dependencies})
}
