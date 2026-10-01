package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// NAVRepository is the in-memory implementation of repositories.NAVRepository.
type NAVRepository struct {
	store *Store
}

// NewNAVRepository returns a new in-memory NAV repository.
func NewNAVRepository(store *Store) *NAVRepository {
	return &NAVRepository{store: store}
}

func navKey(fundID string, date time.Time) string {
	return fmt.Sprintf("%s|%s", fundID, date.Format("2006-01-02"))
}

// Get returns the published NAV for a fund and date, if any.
func (r *NAVRepository) Get(ctx context.Context, fundID string, date time.Time) (domain.NAV, bool, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	nav, ok := r.store.navs[navKey(fundID, date)]
	return nav, ok, nil
}

// Save stores the published NAV for a fund and date.
func (r *NAVRepository) Save(ctx context.Context, fundID string, date time.Time, nav domain.NAV) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	key := navKey(fundID, date)
	if existing, ok := r.store.navs[key]; ok && existing != nav {
		return repositories.ErrConflict
	}
	r.store.navs[key] = nav
	return nil
}
