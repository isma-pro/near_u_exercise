package services

import (
	"testing"
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/stretchr/testify/assert"
)

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func TestTradeDateCalculator(t *testing.T) {
	lisbon := mustLocation("Europe/Lisbon")
	fund := &domain.Fund{
		ID:       "FUND-A",
		Timezone: lisbon,
		CutOff:   time.Date(2000, 1, 1, 12, 0, 0, 0, lisbon),
	}

	calc := NewTradeDateCalculator()

	cases := []struct {
		name     string
		received time.Time
		want     string
	}{
		{
			name:     "friday before cutoff is priced same day",
			received: time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon),
			want:     "2026-10-02",
		},
		{
			name:     "friday at cutoff is next business day",
			received: time.Date(2026, 10, 2, 12, 0, 0, 0, lisbon),
			want:     "2026-10-05",
		},
		{
			name:     "saturday before cutoff rolls to monday",
			received: time.Date(2026, 10, 3, 10, 0, 0, 0, lisbon),
			want:     "2026-10-05",
		},
		{
			name:     "sunday after cutoff rolls to monday",
			received: time.Date(2026, 10, 4, 14, 0, 0, 0, lisbon),
			want:     "2026-10-05",
		},
		{
			name:     "dst change week friday still uses local time",
			received: time.Date(2026, 10, 23, 10, 30, 0, 0, time.UTC),
			want:     "2026-10-23",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calc.TradeDate(fund, tc.received).Format("2006-01-02")
			assert.Equal(t, tc.want, got)
		})
	}
}
