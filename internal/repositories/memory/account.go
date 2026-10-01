package memory

import (
	"context"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// AccountRepository is the in-memory implementation of repositories.AccountRepository.
type AccountRepository struct {
	store *Store
}

// NewAccountRepository returns a new in-memory account repository.
func NewAccountRepository(store *Store) *AccountRepository {
	return &AccountRepository{store: store}
}

// Get returns an account by ID.
func (r *AccountRepository) Get(ctx context.Context, id string) (*domain.Account, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	acc, ok := r.store.accounts[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return acc, nil
}

// Save replaces the stored account with the provided value.
func (r *AccountRepository) Save(ctx context.Context, account *domain.Account) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.accounts[account.ID]; !ok {
		return repositories.ErrNotFound
	}
	if account.Positions == nil {
		account.Positions = map[string]domain.Units{}
	}
	if account.ReservedUnits == nil {
		account.ReservedUnits = map[string]domain.Units{}
	}
	r.store.accounts[account.ID] = account
	return nil
}
