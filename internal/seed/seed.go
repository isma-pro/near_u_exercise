package seed

import (
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
)

// Load returns the hard-coded funds and accounts for the exercise.
func Load() ([]*domain.Fund, []*domain.Account) {
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		panic(err)
	}

	funds := []*domain.Fund{
		{
			ID:       "FUND-A",
			Currency: "EUR",
			CutOff:   time.Date(2000, 1, 1, 12, 0, 0, 0, lisbon),
			Timezone: lisbon,
		},
		{
			ID:       "FUND-B",
			Currency: "EUR",
			CutOff:   time.Date(2000, 1, 1, 12, 0, 0, 0, lisbon),
			Timezone: lisbon,
		},
	}

	accounts := []*domain.Account{
		{
			ID:            "ACC-1",
			Cash:          domain.Money(1_000_000), // €10,000.00
			Positions:     map[string]domain.Units{"FUND-A": domain.Units(100_0000)},
			ReservedUnits: map[string]domain.Units{},
		},
		{
			ID:            "ACC-2",
			Cash:          domain.Money(500_00), // €500.00
			Positions:     map[string]domain.Units{},
			ReservedUnits: map[string]domain.Units{},
		},
	}

	return funds, accounts
}
