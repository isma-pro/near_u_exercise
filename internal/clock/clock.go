package clock

import "time"

// Clock abstracts time so domain logic can be tested deterministically.
type Clock interface {
	Now() time.Time
}

// RealClock returns the current system time.
type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

// FixedClock returns a fixed instant. Useful for tests.
type FixedClock struct {
	Instant time.Time
}

func (c FixedClock) Now() time.Time {
	return c.Instant.UTC()
}

// Advance returns a new FixedClock moved by d.
func (c FixedClock) Advance(d time.Duration) FixedClock {
	return FixedClock{Instant: c.Instant.Add(d)}
}
