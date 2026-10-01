package memory

import (
	"sync"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// Store holds all in-memory state and provides a single mutex for consistency.
type Store struct {
	mu          sync.RWMutex
	funds       map[string]*domain.Fund
	accounts    map[string]*domain.Account
	orders      map[string]*domain.Order
	events      map[string][]*domain.OrderEvent
	idempotency map[string]*repositories.IdempotencyEntry
	navs        map[string]domain.NAV
}

// NewStore returns an empty Store seeded with the provided funds and accounts.
func NewStore(funds []*domain.Fund, accounts []*domain.Account) *Store {
	fundMap := make(map[string]*domain.Fund, len(funds))
	for _, f := range funds {
		fundMap[f.ID] = f
	}

	accountMap := make(map[string]*domain.Account, len(accounts))
	for _, a := range accounts {
		if a.Positions == nil {
			a.Positions = map[string]domain.Units{}
		}
		if a.ReservedUnits == nil {
			a.ReservedUnits = map[string]domain.Units{}
		}
		accountMap[a.ID] = a
	}

	return &Store{
		funds:       fundMap,
		accounts:    accountMap,
		orders:      make(map[string]*domain.Order),
		events:      make(map[string][]*domain.OrderEvent),
		idempotency: make(map[string]*repositories.IdempotencyEntry),
		navs:        make(map[string]domain.NAV),
	}
}
