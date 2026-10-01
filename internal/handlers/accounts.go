package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/services"
)

// AccountsHandler handles account endpoints.
type AccountsHandler struct {
	service *services.AccountService
}

// NewAccountsHandler returns a new AccountsHandler.
func NewAccountsHandler(service *services.AccountService) *AccountsHandler {
	return &AccountsHandler{service: service}
}

// GetAccount returns cash, available cash, positions and available units.
func (h *AccountsHandler) GetAccount(c *gin.Context) {
	account, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}

	availableUnits := make(map[string]domain.Units, len(account.Positions))
	for fundID := range account.Positions {
		availableUnits[fundID] = account.AvailableUnits(fundID)
	}
	for fundID, reserved := range account.ReservedUnits {
		if _, ok := availableUnits[fundID]; !ok && reserved > 0 {
			availableUnits[fundID] = account.AvailableUnits(fundID)
		}
	}

	c.JSON(http.StatusOK, AccountResponse{
		ID:             account.ID,
		Cash:           account.Cash,
		AvailableCash:  account.AvailableCash(),
		Positions:      account.Positions,
		AvailableUnits: availableUnits,
	})
}
