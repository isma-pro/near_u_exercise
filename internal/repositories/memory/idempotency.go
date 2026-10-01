package memory

import (
	"context"

	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// IdempotencyRepository is the in-memory implementation of repositories.IdempotencyRepository.
type IdempotencyRepository struct {
	store *Store
}

// NewIdempotencyRepository returns a new in-memory idempotency repository.
func NewIdempotencyRepository(store *Store) *IdempotencyRepository {
	return &IdempotencyRepository{store: store}
}

// Get returns the stored idempotency entry for a key, or ErrNotFound.
func (r *IdempotencyRepository) Get(ctx context.Context, key string) (*repositories.IdempotencyEntry, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	entry, ok := r.store.idempotency[key]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return entry, nil
}

// Save stores an idempotency entry. It returns ErrConflict if the key already exists
// with a different fingerprint.
func (r *IdempotencyRepository) Save(ctx context.Context, key string, entry *repositories.IdempotencyEntry) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if existing, ok := r.store.idempotency[key]; ok && existing.Fingerprint != entry.Fingerprint {
		return repositories.ErrConflict
	}
	r.store.idempotency[key] = entry
	return nil
}
