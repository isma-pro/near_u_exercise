package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// IdempotencyRepository is the PostgreSQL-backed implementation.
type IdempotencyRepository struct {
	pool *pgxpool.Pool
}

// NewIdempotencyRepository returns a new PostgreSQL idempotency repository.
func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{pool: pool}
}

// Get returns the stored entry for a key, locking the row when inside a
// transaction so concurrent requests with the same key serialise correctly.
func (r *IdempotencyRepository) Get(ctx context.Context, key string) (*repositories.IdempotencyEntry, error) {
	q := getQuerier(ctx, r.pool)
	lock := ""
	if inTx(ctx) {
		lock = "FOR UPDATE"
	}
	query := fmt.Sprintf(`SELECT order_id, fingerprint FROM idempotency_keys WHERE key = $1 %s`, lock)
	row := q.QueryRow(ctx, query, key)

	entry := &repositories.IdempotencyEntry{}
	if err := row.Scan(&entry.OrderID, &entry.Fingerprint); err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return entry, nil
}

// Save inserts an idempotency entry. The unique constraint on `key` ensures
// conflicting concurrent inserts are caught by the database.
func (r *IdempotencyRepository) Save(ctx context.Context, key string, entry *repositories.IdempotencyEntry) error {
	q := getQuerier(ctx, r.pool)
	query := `INSERT INTO idempotency_keys (key, order_id, fingerprint) VALUES ($1, $2, $3)`
	if _, err := q.Exec(ctx, query, key, entry.OrderID, entry.Fingerprint); err != nil {
		return err
	}
	return nil
}
