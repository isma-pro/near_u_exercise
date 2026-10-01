package memory

import (
	"context"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
)

// EventRepository is the in-memory implementation of repositories.EventRepository.
type EventRepository struct {
	store *Store
}

// NewEventRepository returns a new in-memory event repository.
func NewEventRepository(store *Store) *EventRepository {
	return &EventRepository{store: store}
}

// Append stores an audit event for an order.
func (r *EventRepository) Append(ctx context.Context, event *domain.OrderEvent) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	r.store.events[event.OrderID] = append(r.store.events[event.OrderID], event)
	return nil
}

// ListForOrder returns audit events for an order in chronological order.
func (r *EventRepository) ListForOrder(ctx context.Context, orderID string) ([]*domain.OrderEvent, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	events := r.store.events[orderID]
	out := make([]*domain.OrderEvent, len(events))
	copy(out, events)
	return out, nil
}
