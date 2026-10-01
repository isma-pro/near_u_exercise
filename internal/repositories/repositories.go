package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
)

// Common repository errors.
var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrAlreadyPriced = errors.New("nav already priced with a different value")
)

// ListCursor is used for keyset pagination on orders.
type ListCursor struct {
	CreatedAt time.Time
	ID        string
}

// ListOrdersFilter controls pagination and filtering for orders.
type ListOrdersFilter struct {
	AccountID string
	FundID    string
	Status    domain.Status
	Limit     int
	Cursor    ListCursor
	HasCursor bool
}

// FundRepository provides access to fund definitions.
type FundRepository interface {
	Get(ctx context.Context, id string) (*domain.Fund, error)
	List(ctx context.Context) ([]*domain.Fund, error)
}

// AccountRepository provides access to accounts and their positions.
type AccountRepository interface {
	Get(ctx context.Context, id string) (*domain.Account, error)
	Save(ctx context.Context, account *domain.Account) error
}

// OrderRepository provides access to orders.
type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	Get(ctx context.Context, id string) (*domain.Order, error)
	List(ctx context.Context, filter ListOrdersFilter) ([]*domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
}

// EventRepository provides access to order audit events.
type EventRepository interface {
	Append(ctx context.Context, event *domain.OrderEvent) error
	ListForOrder(ctx context.Context, orderID string) ([]*domain.OrderEvent, error)
}

// IdempotencyEntry stores the mapping between a key and its resulting order.
type IdempotencyEntry struct {
	OrderID     string
	Fingerprint string
}

// IdempotencyRepository stores idempotency keys and their fingerprints.
type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (*IdempotencyEntry, error)
	Save(ctx context.Context, key string, entry *IdempotencyEntry) error
}

// NAVRepository stores published NAVs per fund and date.
type NAVRepository interface {
	Get(ctx context.Context, fundID string, date time.Time) (domain.NAV, bool, error)
	Save(ctx context.Context, fundID string, date time.Time, nav domain.NAV) error
}
