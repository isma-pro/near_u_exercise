package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Money represents cash as an integer number of minor units (cents).
// 1 EUR = 100 Money. Never use floating-point for money.
type Money int64

// Units represents fund units as an integer number of 10^-4 units.
// 1 unit = 10_000 Units. Never use floating-point for units.
type Units int64

// NAV represents a net asset value as an integer number of 10^-4 per unit.
// 1 EUR per unit = 10_000 NAV. Never use floating-point for NAV.
type NAV int64

const (
	moneyScale  = 2
	unitsScale  = 4
	navScale    = 4
	moneyFactor = int64(1) * 100
	unitsFactor = int64(1) * 10000
	navFactor   = int64(1) * 10000
	// pricingFactor is the product of the two 10^4 scales divided by the cash scale (100).
	// It is used when converting redemption units * NAV back to cents.
	pricingFactor = int64(unitsFactor) * int64(navFactor) / moneyFactor // 1_000_000
)

// ParseMoney parses a decimal string with up to 2 decimal places.
func ParseMoney(s string) (Money, error) {
	v, err := parseFixedPoint(s, moneyScale)
	if err != nil {
		return 0, fmt.Errorf("invalid money %q: %w", s, err)
	}
	return Money(v), nil
}

// ParseUnits parses a decimal string with up to 4 decimal places.
func ParseUnits(s string) (Units, error) {
	v, err := parseFixedPoint(s, unitsScale)
	if err != nil {
		return 0, fmt.Errorf("invalid units %q: %w", s, err)
	}
	return Units(v), nil
}

// ParseNAV parses a decimal string with up to 4 decimal places.
func ParseNAV(s string) (NAV, error) {
	v, err := parseFixedPoint(s, navScale)
	if err != nil {
		return 0, fmt.Errorf("invalid nav %q: %w", s, err)
	}
	return NAV(v), nil
}

func parseFixedPoint(s string, scale int) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty value")
	}

	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}

	parts := strings.SplitN(s, ".", 2)
	if len(parts) == 1 {
		parts = append(parts, "")
	}

	whole, frac := parts[0], parts[1]
	if whole == "" {
		whole = "0"
	}
	if len(frac) > scale {
		return 0, fmt.Errorf("too many decimal places (max %d)", scale)
	}
	frac = frac + strings.Repeat("0", scale-len(frac))

	if _, err := strconv.ParseUint(whole, 10, 64); err != nil {
		return 0, fmt.Errorf("invalid whole part")
	}

	combined := whole + frac
	v, err := strconv.ParseInt(combined, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("value out of range")
	}

	if negative {
		v = -v
	}
	return v, nil
}

// String formats Money with exactly 2 decimal places.
func (m Money) String() string {
	return formatFixedPoint(int64(m), moneyScale)
}

// String formats Units with exactly 4 decimal places.
func (u Units) String() string {
	return formatFixedPoint(int64(u), unitsScale)
}

// String formats NAV with exactly 4 decimal places.
func (n NAV) String() string {
	return formatFixedPoint(int64(n), navScale)
}

func formatFixedPoint(v int64, scale int) string {
	negative := v < 0
	if negative {
		v = -v
	}

	factor := int64(1)
	for i := 0; i < scale; i++ {
		factor *= 10
	}

	whole := v / factor
	frac := v % factor

	s := fmt.Sprintf("%d.%0*d", whole, scale, frac)
	if negative {
		s = "-" + s
	}
	return s
}

// MarshalJSON serialises the value as a JSON string.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

// UnmarshalJSON expects a JSON string with up to 2 decimal places.
func (m *Money) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseMoney(raw)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

func (u Units) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *Units) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseUnits(raw)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

func (n NAV) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.String())
}

func (n *NAV) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseNAV(raw)
	if err != nil {
		return err
	}
	*n = parsed
	return nil
}

// UnitsForSubscription returns the units for a subscription, rounded down to 4 decimals.
// units = amount / nav, converted to the fixed-point scales.
func UnitsForSubscription(amount Money, nav NAV) Units {
	if nav == 0 {
		return 0
	}
	// amount_euros / nav_euros = (amount/100) / (nav/10000)
	// Convert to 10^-4 units: amount * 10000 * 10000 / (nav * 100)
	// = amount * 1_000_000 / nav
	return Units((int64(amount) * 1_000_000) / int64(nav))
}

// MoneyForRedemption returns the proceeds for a redemption, rounded down to 2 decimals.
// proceeds = units * nav, converted to cents.
func MoneyForRedemption(units Units, nav NAV) Money {
	// (units/10000) * (nav/10000) euros = units * nav / 100_000_000 euros
	// Convert to cents: units * nav / 1_000_000
	return Money((int64(units) * int64(nav)) / pricingFactor)
}

// Add returns m + other.
func (m Money) Add(other Money) Money {
	return m + other
}

// Sub returns m - other.
func (m Money) Sub(other Money) Money {
	return m - other
}

// Add returns u + other.
func (u Units) Add(other Units) Units {
	return u + other
}

// Sub returns u - other.
func (u Units) Sub(other Units) Units {
	return u - other
}
