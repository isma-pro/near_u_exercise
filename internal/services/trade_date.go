package services

import (
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
)

// TradeDateCalculator computes the trade date for an order based on the fund's
// cut-off time and timezone.
type TradeDateCalculator struct{}

// NewTradeDateCalculator returns a new calculator.
func NewTradeDateCalculator() *TradeDateCalculator {
	return &TradeDateCalculator{}
}

// TradeDate returns the next business day on or after the candidate derived from
// the received time and the fund cut-off.
func (c *TradeDateCalculator) TradeDate(fund *domain.Fund, received time.Time) time.Time {
	local := received.In(fund.Timezone)
	cutoff := fund.CutOffFor(local)

	candidate := local
	if !local.Before(cutoff) {
		candidate = candidate.AddDate(0, 0, 1)
	}

	candidate = startOfDay(candidate)
	for isWeekend(candidate) {
		candidate = candidate.AddDate(0, 0, 1)
	}

	return candidate.UTC()
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func isWeekend(t time.Time) bool {
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}
