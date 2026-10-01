package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler exposes liveness and readiness endpoints.
type HealthHandler struct {
	ready func() bool
}

// NewHealthHandler returns a new HealthHandler.
func NewHealthHandler(ready func() bool) *HealthHandler {
	if ready == nil {
		ready = func() bool { return true }
	}
	return &HealthHandler{ready: ready}
}

// Liveness always reports ok.
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness reports whether the service is ready to handle traffic.
func (h *HealthHandler) Readiness(c *gin.Context) {
	if h.ready() {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
}
