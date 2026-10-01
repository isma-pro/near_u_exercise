package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
)

// Seed inserts the hard-coded funds and accounts into PostgreSQL.
func Seed(ctx context.Context, pool *pgxpool.Pool, funds []*domain.Fund, accounts []*domain.Account) error {
	for _, f := range funds {
		query := `
			INSERT INTO funds (id, currency, timezone, cutoff_hour, cutoff_minute)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO NOTHING
		`
		if _, err := pool.Exec(ctx, query,
			f.ID, f.Currency, f.Timezone.String(), f.CutOff.Hour(), f.CutOff.Minute()); err != nil {
			return fmt.Errorf("seed fund %s: %w", f.ID, err)
		}
	}

	for _, a := range accounts {
		query := `
			INSERT INTO accounts (id, cash, reserved_cash)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO NOTHING
		`
		if _, err := pool.Exec(ctx, query, a.ID, a.Cash, a.ReservedCash); err != nil {
			return fmt.Errorf("seed account %s: %w", a.ID, err)
		}

		for fundID, units := range a.Positions {
			posQuery := `
				INSERT INTO positions (account_id, fund_id, units, reserved_units)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (account_id, fund_id) DO NOTHING
			`
			if _, err := pool.Exec(ctx, posQuery, a.ID, fundID, units, a.ReservedUnits[fundID]); err != nil {
				return fmt.Errorf("seed position %s/%s: %w", a.ID, fundID, err)
			}
		}
	}

	return nil
}
