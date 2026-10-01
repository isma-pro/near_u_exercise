package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
	"github.com/ismaelucky94/near_u_exercise/internal/services"
)

// ErrorResponse is the consistent JSON error shape returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// respondError maps domain errors to HTTP status codes and writes a JSON body.
func respondError(c *gin.Context, err error) {
	status, code := mapError(err)
	c.JSON(status, ErrorResponse{Error: err.Error(), Code: code})
}

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, services.ErrInvalidOrder),
		errors.Is(err, services.ErrInvalidCursor):
		return http.StatusBadRequest, "bad_request"
	case errors.Is(err, services.ErrInsufficientCash),
		errors.Is(err, services.ErrInsufficientUnits),
		errors.Is(err, services.ErrOrderNotCancellable),
		errors.Is(err, services.ErrNAVAlreadyPriced):
		return http.StatusUnprocessableEntity, "unprocessable"
	case errors.Is(err, services.ErrIdempotencyConflict),
		errors.Is(err, repositories.ErrConflict):
		return http.StatusConflict, "conflict"
	case errors.Is(err, repositories.ErrNotFound),
		errors.Is(err, services.ErrFundNotFound),
		errors.Is(err, services.ErrAccountNotFound):
		return http.StatusNotFound, "not_found"
	default:
		return http.StatusInternalServerError, "internal"
	}
}
