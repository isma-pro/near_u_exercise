package domain

import "time"

// Fund represents a fund with its pricing currency and daily cut-off.
type Fund struct {
	ID       string
	Currency string
	CutOff   time.Time     // wall-clock representation of 12:00 in the fund's timezone
	Timezone *time.Location
}

// CutOffFor returns the cut-off time for a given date in the fund's timezone.
func (f Fund) CutOffFor(date time.Time) time.Time {
	y, m, d := date.In(f.Timezone).Date()
	return time.Date(y, m, d, f.CutOff.Hour(), f.CutOff.Minute(), 0, 0, f.Timezone)
}
