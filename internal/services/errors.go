package services

import "errors"

// Business-rule errors returned by services.
var (
	ErrInvalidOrder          = errors.New("invalid order")
	ErrFundNotFound          = errors.New("fund not found")
	ErrAccountNotFound       = errors.New("account not found")
	ErrInsufficientCash      = errors.New("insufficient cash")
	ErrInsufficientUnits     = errors.New("insufficient units")
	ErrOrderNotCancellable   = errors.New("order cannot be cancelled")
	ErrNAVAlreadyPriced      = errors.New("nav already priced for this date")
	ErrIdempotencyConflict   = errors.New("idempotency key conflict")
	ErrInvalidCursor         = errors.New("invalid cursor")
)
