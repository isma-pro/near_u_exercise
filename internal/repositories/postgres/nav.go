package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
)

// NAVRepository is the PostgreSQL-backed implementation.
type NAVRepository struct {
	pool *pgxpool.Pool
}

// NewNAVRepository returns a new PostgreSQL NAV repository.
func NewNAVRepository(pool *pgxpool.Pool) *NAVRepository {
	return &NAVRepository{pool: pool}
}

// Get returns the published NAV for a fund and date, if any.
func (r *NAVRepository) Get(ctx context.Context, fundID string, date time.Time) (domain.NAV, bool, error) {
	q := getQuerier(ctx, r.pool)
	query := `SELECT nav FROM nav_prices WHERE fund_id = $1 AND trade_date = $2`
	row := q.QueryRow(ctx, query, fundID, date)

	var nav domain.NAV
	if err := row.Scan(&nav); err != nil {
		if err == pgx.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return nav, true, nil
}

// Save inserts or updates the published NAV. Conflicts are handled at the
// application level before calling Save, so this uses a plain INSERT.
func (r *NAVRepository) Save(ctx context.Context, fundID string, date time.Time, nav domain.NAV) error {
	q := getQuerier(ctx, r.pool)
	query := `INSERT INTO nav_prices (fund_id, trade_date, nav) VALUES ($1, $2, $3)`
	_, err := q.Exec(ctx, query, fundID, date, nav)
	return err
}
