package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// FundRepository is the PostgreSQL-backed implementation.
type FundRepository struct {
	pool *pgxpool.Pool
}

// NewFundRepository returns a new PostgreSQL fund repository.
func NewFundRepository(pool *pgxpool.Pool) *FundRepository {
	return &FundRepository{pool: pool}
}

// Get returns a fund by ID. All values come from query parameters to avoid SQL injection.
func (r *FundRepository) Get(ctx context.Context, id string) (*domain.Fund, error) {
	var fund domain.Fund
	var tzName string
	var cutoffHour, cutoffMinute int

	query := `SELECT id, currency, timezone, cutoff_hour, cutoff_minute FROM funds WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	if err := row.Scan(&fund.ID, &fund.Currency, &tzName, &cutoffHour, &cutoffMinute); err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, err
	}
	fund.Timezone = loc
	fund.CutOff = time.Date(2000, 1, 1, cutoffHour, cutoffMinute, 0, 0, loc)
	return &fund, nil
}

// List returns all funds.
func (r *FundRepository) List(ctx context.Context) ([]*domain.Fund, error) {
	query := `SELECT id, currency, timezone, cutoff_hour, cutoff_minute FROM funds`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funds []*domain.Fund
	for rows.Next() {
		var fund domain.Fund
		var tzName string
		var cutoffHour, cutoffMinute int
		if err := rows.Scan(&fund.ID, &fund.Currency, &tzName, &cutoffHour, &cutoffMinute); err != nil {
			return nil, err
		}
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			return nil, err
		}
		fund.Timezone = loc
		fund.CutOff = time.Date(2000, 1, 1, cutoffHour, cutoffMinute, 0, 0, loc)
		funds = append(funds, &fund)
	}
	return funds, rows.Err()
}
