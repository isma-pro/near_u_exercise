package memory

import (
	"context"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// FundRepository is the in-memory implementation of repositories.FundRepository.
type FundRepository struct {
	store *Store
}

// NewFundRepository returns a new in-memory fund repository.
func NewFundRepository(store *Store) *FundRepository {
	return &FundRepository{store: store}
}

// Get returns a fund by ID.
func (r *FundRepository) Get(ctx context.Context, id string) (*domain.Fund, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	fund, ok := r.store.funds[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return fund, nil
}

// List returns all funds.
func (r *FundRepository) List(ctx context.Context) ([]*domain.Fund, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	out := make([]*domain.Fund, 0, len(r.store.funds))
	for _, f := range r.store.funds {
		out = append(out, f)
	}
	return out, nil
}
