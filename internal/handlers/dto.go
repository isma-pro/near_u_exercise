package handlers

import (
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
)

// PlaceOrderRequest is the body for POST /orders.
type PlaceOrderRequest struct {
	AccountID string       `json:"account_id" binding:"required"`
	FundID    string       `json:"fund_id" binding:"required"`
	Side      string       `json:"side" binding:"required,oneof=SUBSCRIPTION REDEMPTION"`
	Amount    domain.Money `json:"amount"`
	Units     domain.Units `json:"units"`
}

// PublishNAVRequest is the body for POST /funds/{id}/navs.
type PublishNAVRequest struct {
	Date string    `json:"date" binding:"required,datetime=2006-01-02"`
	NAV  domain.NAV `json:"nav" binding:"required"`
}

// OrderResponse is the JSON representation of an order.
type OrderResponse struct {
	ID           string       `json:"id"`
	AccountID    string       `json:"account_id"`
	FundID       string       `json:"fund_id"`
	Side         string       `json:"side"`
	Amount       domain.Money `json:"amount,omitempty"`
	Units        domain.Units `json:"units,omitempty"`
	Status       string       `json:"status"`
	TradeDate    string       `json:"trade_date"`
	NAVUsed      domain.NAV   `json:"nav_used,omitempty"`
	PricedUnits  domain.Units `json:"priced_units,omitempty"`
	PricedAmount domain.Money `json:"priced_amount,omitempty"`
	CreatedAt    string       `json:"created_at"`
	UpdatedAt    string       `json:"updated_at"`
}

// AccountResponse is the JSON representation of an account.
type AccountResponse struct {
	ID              string                     `json:"id"`
	Cash            domain.Money               `json:"cash"`
	AvailableCash   domain.Money               `json:"available_cash"`
	Positions       map[string]domain.Units    `json:"positions"`
	AvailableUnits  map[string]domain.Units    `json:"available_units"`
}

// ListOrdersResponse wraps a paginated list of orders.
type ListOrdersResponse struct {
	Orders     []OrderResponse `json:"orders"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

func orderResponse(o *domain.Order) OrderResponse {
	return OrderResponse{
		ID:           o.ID,
		AccountID:    o.AccountID,
		FundID:       o.FundID,
		Side:         string(o.Side),
		Amount:       o.Amount,
		Units:        o.Units,
		Status:       string(o.Status),
		TradeDate:    o.TradeDate.Format("2006-01-02"),
		NAVUsed:      o.NAVUsed,
		PricedUnits:  o.PricedUnits,
		PricedAmount: o.PricedAmount,
		CreatedAt:    o.CreatedAt.Format(timeFormat),
		UpdatedAt:    o.UpdatedAt.Format(timeFormat),
	}
}

const timeFormat = "2006-01-02T15:04:05Z"
