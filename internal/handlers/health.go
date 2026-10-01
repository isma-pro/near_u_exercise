package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler exposes health check endpoints.
type HealthHandler struct{}

// NewHealthHandler returns a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check responds with the service health status.
func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
