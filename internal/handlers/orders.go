package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/services"
)

// OrdersHandler handles order endpoints.
type OrdersHandler struct {
	service *services.OrderService
}

// NewOrdersHandler returns a new OrdersHandler.
func NewOrdersHandler(service *services.OrderService) *OrdersHandler {
	return &OrdersHandler{service: service}
}

// PlaceOrder accepts a subscription or redemption.
func (h *OrdersHandler) PlaceOrder(c *gin.Context) {
	key := c.GetHeader("Idempotency-Key")
	if key == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Idempotency-Key header is required", Code: "bad_request"})
		return
	}

	raw, err := c.GetRawData()
	if err != nil {
		respondError(c, services.ErrInvalidOrder)
		return
	}

	var req PlaceOrderRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error(), Code: "bad_request"})
		return
	}

	if err := validatePlaceOrder(req); err != nil {
		respondError(c, err)
		return
	}

	hash := sha256.Sum256(raw)
	fingerprint := hex.EncodeToString(hash[:])

	order, err := h.service.PlaceOrder(c.Request.Context(), services.PlaceOrderInput{
		AccountID:      req.AccountID,
		FundID:         req.FundID,
		Side:           domain.Side(req.Side),
		Amount:         req.Amount,
		Units:          req.Units,
		IdempotencyKey: key,
		Fingerprint:    fingerprint,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, orderResponse(order))
}

func validatePlaceOrder(req PlaceOrderRequest) error {
	if req.Side == string(domain.SideSubscription) {
		if req.Amount <= 0 {
			return services.ErrInvalidOrder
		}
	}
	if req.Side == string(domain.SideRedemption) {
		if req.Units <= 0 {
			return services.ErrInvalidOrder
		}
	}
	return nil
}

// GetOrder returns a single order.
func (h *OrdersHandler) GetOrder(c *gin.Context) {
	order, err := h.service.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, orderResponse(order))
}

// ListOrders returns a paginated list of orders.
func (h *OrdersHandler) ListOrders(c *gin.Context) {
	accountID := c.Query("account_id")
	status := domain.Status(c.Query("status"))
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	cursor := c.Query("cursor")

	result, err := h.service.ListOrders(c.Request.Context(), accountID, status, limit, cursor)
	if err != nil {
		respondError(c, err)
		return
	}

	resp := ListOrdersResponse{
		Orders:     make([]OrderResponse, 0, len(result.Orders)),
		NextCursor: result.NextCursor,
	}
	for _, o := range result.Orders {
		resp.Orders = append(resp.Orders, orderResponse(o))
	}
	c.JSON(http.StatusOK, resp)
}

// CancelOrder cancels a received order.
func (h *OrdersHandler) CancelOrder(c *gin.Context) {
	order, err := h.service.CancelOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, orderResponse(order))
}

// GetEvents returns the audit trail for an order.
func (h *OrdersHandler) GetEvents(c *gin.Context) {
	events, err := h.service.GetAuditEvents(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}

	resp := make([]eventResponse, 0, len(events))
	for _, e := range events {
		resp = append(resp, eventResponse{
			Type:         string(e.Type),
			Status:       string(e.Status),
			Timestamp:    e.Timestamp.Format(timeFormat),
			NavUsed:      e.NAVUsed,
			PricedUnits:  e.PricedUnits,
			PricedAmount: e.PricedAmount,
			Reason:       e.Reason,
		})
	}
	c.JSON(http.StatusOK, resp)
}

type eventResponse struct {
	Type         string       `json:"type"`
	Status       string       `json:"status"`
	Timestamp    string       `json:"timestamp"`
	NavUsed      domain.NAV   `json:"nav_used,omitempty"`
	PricedUnits  domain.Units `json:"priced_units,omitempty"`
	PricedAmount domain.Money `json:"priced_amount,omitempty"`
	Reason       string       `json:"reason,omitempty"`
}
