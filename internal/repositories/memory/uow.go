package memory

import (
	"context"

	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// UnitOfWork is the in-memory implementation. It relies on the store-level
// mutexes inside each repository for atomicity.
type UnitOfWork struct {
	store *Store
}

// NewUnitOfWork returns a new in-memory unit of work.
func NewUnitOfWork(store *Store) *UnitOfWork {
	return &UnitOfWork{store: store}
}

// Run executes fn directly. The caller is responsible for higher-level locking.
func (u *UnitOfWork) Run(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

var _ repositories.UnitOfWork = (*UnitOfWork)(nil)
