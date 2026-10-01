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
	y, m, d := local.Date()
	cutoff := time.Date(y, m, d, fund.CutOff.Hour(), fund.CutOff.Minute(), 0, 0, fund.Timezone)

	candidate := civilDate{y, m, d}
	if !local.Before(cutoff) {
		candidate = candidate.addDays(1)
	}

	for candidate.weekday() == time.Saturday || candidate.weekday() == time.Sunday {
		candidate = candidate.addDays(1)
	}

	return candidate.toTime()
}

// civilDate holds a calendar date without a time-of-day or location.
type civilDate struct {
	year  int
	month time.Month
	day   int
}

func (d civilDate) addDays(n int) civilDate {
	t := time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, n)
	y, m, day := t.Date()
	return civilDate{y, m, day}
}

func (d civilDate) weekday() time.Weekday {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC).Weekday()
}

func (d civilDate) toTime() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}
