package memory

import (
	"context"
	"sort"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// OrderRepository is the in-memory implementation of repositories.OrderRepository.
type OrderRepository struct {
	store *Store
}

// NewOrderRepository returns a new in-memory order repository.
func NewOrderRepository(store *Store) *OrderRepository {
	return &OrderRepository{store: store}
}

// Create stores a new order.
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	r.store.orders[order.ID] = order
	return nil
}

// Get returns an order by ID.
func (r *OrderRepository) Get(ctx context.Context, id string) (*domain.Order, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	order, ok := r.store.orders[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return order, nil
}

// List returns orders matching the filter, sorted by creation time then ID.
func (r *OrderRepository) List(ctx context.Context, filter repositories.ListOrdersFilter) ([]*domain.Order, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	matches := make([]*domain.Order, 0)
	for _, o := range r.store.orders {
		if filter.AccountID != "" && o.AccountID != filter.AccountID {
			continue
		}
		if filter.FundID != "" && o.FundID != filter.FundID {
			continue
		}
		if filter.Status != "" && o.Status != filter.Status {
			continue
		}
		if filter.HasCursor {
			if o.CreatedAt.Before(filter.Cursor.CreatedAt) {
				continue
			}
			if o.CreatedAt.Equal(filter.Cursor.CreatedAt) && o.ID <= filter.Cursor.ID {
				continue
			}
		}
		matches = append(matches, o)
	}

	sort.Slice(matches, func(i, j int) bool {
		if !matches[i].CreatedAt.Equal(matches[j].CreatedAt) {
			return matches[i].CreatedAt.Before(matches[j].CreatedAt)
		}
		return matches[i].ID < matches[j].ID
	})

	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

// Update replaces an existing order.
func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.orders[order.ID]; !ok {
		return repositories.ErrNotFound
	}
	r.store.orders[order.ID] = order
	return nil
}
