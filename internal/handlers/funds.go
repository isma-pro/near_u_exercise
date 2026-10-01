package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/services"
)

// FundsHandler handles fund endpoints.
type FundsHandler struct {
	service *services.PricingService
}

// NewFundsHandler returns a new FundsHandler.
func NewFundsHandler(service *services.PricingService) *FundsHandler {
	return &FundsHandler{service: service}
}

// PublishNAV publishes a NAV for a fund and date.
func (h *FundsHandler) PublishNAV(c *gin.Context) {
	var req PublishNAVRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error(), Code: "bad_request"})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error(), Code: "bad_request"})
		return
	}

	if err := h.service.PublishNAV(c.Request.Context(), c.Param("id"), date, req.NAV); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
