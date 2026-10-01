package domain

import "time"

// Side is the direction of an order.
type Side string

const (
	SideSubscription Side = "SUBSCRIPTION"
	SideRedemption Side = "REDEMPTION"
)

// Status is the lifecycle state of an order.
type Status string

const (
	StatusReceived Status = "RECEIVED"
	StatusPriced   Status = "PRICED"
	StatusCancelled Status = "CANCELLED"
	StatusRejected Status = "REJECTED"
)

// Order represents a subscription or redemption placed by an account.
type Order struct {
	ID          string
	AccountID   string
	FundID      string
	Side        Side
	Amount      Money  // used for subscriptions
	Units       Units  // used for redemptions
	Status      Status
	TradeDate   time.Time // midnight UTC of the trade date
	NAVUsed     NAV
	PricedUnits Units
	PricedAmount Money
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IdempotencyKey string
}

// IsSubscription reports whether the order is a subscription.
func (o Order) IsSubscription() bool {
	return o.Side == SideSubscription
}

// IsRedemption reports whether the order is a redemption.
func (o Order) IsRedemption() bool {
	return o.Side == SideRedemption
}
