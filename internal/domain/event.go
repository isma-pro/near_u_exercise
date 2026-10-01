package domain

import "time"

// EventType records what happened to an order.
type EventType string

const (
	EventReceived EventType = "RECEIVED"
	EventCancelled EventType = "CANCELLED"
	EventPriced    EventType = "PRICED"
	EventRejected  EventType = "REJECTED"
)

// OrderEvent is an immutable audit entry for an order state change.
type OrderEvent struct {
	OrderID      string
	Type         EventType
	Status       Status
	Timestamp    time.Time
	NAVUsed      NAV
	PricedUnits  Units
	PricedAmount Money
	Reason       string
}
