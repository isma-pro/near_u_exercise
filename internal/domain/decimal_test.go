package domain

import (
	"encoding/json"
	"testing"
)

func TestParseMoney(t *testing.T) {
	cases := []struct {
		in   string
		want Money
		err  bool
	}{
		{"1000.00", 1000_00, false},
		{"1000", 1000_00, false},
		{"0.01", 1, false},
		{"0.10", 10, false},
		{"-12.34", -1234, false},
		{"", 0, true},
		{"12.345", 0, true},
		{"abc", 0, true},
	}
	for _, tc := range cases {
		got, err := ParseMoney(tc.in)
		if tc.err {
			if err == nil {
				t.Errorf("ParseMoney(%q) expected error, got %v", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseMoney(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseMoney(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestMoneyString(t *testing.T) {
	cases := []struct {
		m    Money
		want string
	}{
		{1000_00, "1000.00"},
		{1, "0.01"},
		{10, "0.10"},
		{-1234, "-12.34"},
	}
	for _, tc := range cases {
		if got := tc.m.String(); got != tc.want {
			t.Errorf("Money(%d).String() = %q, want %q", tc.m, got, tc.want)
		}
	}
}

func TestMoneyJSON(t *testing.T) {
	m := Money(1234_56)
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"1234.56"` {
		t.Fatalf("marshal money: %s", b)
	}

	var parsed Money
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed != m {
		t.Fatalf("round-trip money: got %v, want %v", parsed, m)
	}
}

func TestUnitsForSubscription(t *testing.T) {
	cases := []struct {
		amount Money
		nav    NAV
		want   Units
	}{
		// €1,000.00 / 12.3456 = 81.0005 units (rounded down)
		{1000_00, NAV(12_3456), Units(81_0005)},
		// €100.00 / 10.0000 = 10 units
		{100_00, NAV(10_0000), Units(10_0000)},
		// €1.00 / 3.3333 = 0.3000 units
		{1_00, NAV(3_3333), Units(3000)},
	}
	for _, tc := range cases {
		got := UnitsForSubscription(tc.amount, tc.nav)
		if got != tc.want {
			t.Errorf("UnitsForSubscription(%v, %v) = %v, want %v", tc.amount, tc.nav, got, tc.want)
		}
	}
}

func TestMoneyForRedemption(t *testing.T) {
	cases := []struct {
		units Units
		nav   NAV
		want  Money
	}{
		// 33.3333 units * 12.3456 = €411.5194... rounded down to €411.51
		{Units(33_3333), NAV(12_3456), Money(411_51)},
		// 10 units * 10.0000 = €100.00
		{Units(10_0000), NAV(10_0000), Money(100_00)},
	}
	for _, tc := range cases {
		got := MoneyForRedemption(tc.units, tc.nav)
		if got != tc.want {
			t.Errorf("MoneyForRedemption(%v, %v) = %v, want %v", tc.units, tc.nav, got, tc.want)
		}
	}
}
