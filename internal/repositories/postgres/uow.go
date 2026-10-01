package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// UnitOfWork is the PostgreSQL implementation using a database transaction.
type UnitOfWork struct {
	pool *pgxpool.Pool
}

// NewUnitOfWork returns a new PostgreSQL unit of work.
func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

// Run executes fn inside a database transaction.
func (u *UnitOfWork) Run(ctx context.Context, fn func(context.Context) error) error {
	return WithTx(ctx, u.pool, fn)
}

var _ repositories.UnitOfWork = (*UnitOfWork)(nil)
