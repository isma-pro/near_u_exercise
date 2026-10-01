package memory

import (
	"context"
	"testing"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
	"github.com/ismaelucky94/near_u_exercise/internal/seed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStore(t *testing.T) *Store {
	t.Helper()
	funds, accounts := seed.Load()
	return NewStore(funds, accounts)
}

func TestFundRepository_Get(t *testing.T) {
	store := setupStore(t)
	repo := NewFundRepository(store)

	fund, err := repo.Get(context.Background(), "FUND-A")
	require.NoError(t, err)
	assert.Equal(t, "FUND-A", fund.ID)

	_, err = repo.Get(context.Background(), "UNKNOWN")
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestAccountRepository_Get(t *testing.T) {
	store := setupStore(t)
	repo := NewAccountRepository(store)

	acc, err := repo.Get(context.Background(), "ACC-1")
	require.NoError(t, err)
	assert.Equal(t, domain.Money(1000_00), acc.Cash)
	assert.Equal(t, domain.Units(100_0000), acc.Positions["FUND-A"])
}

func TestOrderRepository_CRUD(t *testing.T) {
	store := setupStore(t)
	repo := NewOrderRepository(store)
	ctx := context.Background()

	o := &domain.Order{
		ID:        "ORDER-1",
		AccountID: "ACC-1",
		FundID:    "FUND-A",
		Side:      domain.SideSubscription,
		Amount:    domain.Money(100_00),
		Status:    domain.StatusReceived,
	}

	require.NoError(t, repo.Create(ctx, o))

	got, err := repo.Get(ctx, "ORDER-1")
	require.NoError(t, err)
	assert.Equal(t, o.ID, got.ID)

	o.Status = domain.StatusPriced
	require.NoError(t, repo.Update(ctx, o))

	got, err = repo.Get(ctx, "ORDER-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPriced, got.Status)
}

func TestOrderRepository_List(t *testing.T) {
	store := setupStore(t)
	repo := NewOrderRepository(store)
	ctx := context.Background()

	for i, id := range []string{"A", "B", "C"} {
		o := &domain.Order{
			ID:        id,
			AccountID: "ACC-1",
			FundID:    "FUND-A",
			Side:      domain.SideSubscription,
			Amount:    domain.Money(100_00),
			Status:    domain.StatusReceived,
		}
		if i == 1 {
			o.Status = domain.StatusPriced
		}
		require.NoError(t, repo.Create(ctx, o))
	}

	filter := repositories.ListOrdersFilter{AccountID: "ACC-1", Status: domain.StatusReceived, Limit: 10}
	got, err := repo.List(ctx, filter)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "A", got[0].ID)
	assert.Equal(t, "C", got[1].ID)
}

func TestEventRepository_ListForOrder(t *testing.T) {
	store := setupStore(t)
	repo := NewEventRepository(store)
	ctx := context.Background()

	events := []*domain.OrderEvent{
		{OrderID: "ORDER-1", Type: domain.EventReceived, Status: domain.StatusReceived},
		{OrderID: "ORDER-1", Type: domain.EventPriced, Status: domain.StatusPriced},
	}
	for _, e := range events {
		require.NoError(t, repo.Append(ctx, e))
	}

	got, err := repo.ListForOrder(ctx, "ORDER-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, domain.EventReceived, got[0].Type)
	assert.Equal(t, domain.EventPriced, got[1].Type)
}

func TestIdempotencyRepository_Conflict(t *testing.T) {
	store := setupStore(t)
	repo := NewIdempotencyRepository(store)
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, "key-1", &repositories.IdempotencyEntry{OrderID: "ORDER-1", Fingerprint: "fp-1"}))

	got, err := repo.Get(ctx, "key-1")
	require.NoError(t, err)
	assert.Equal(t, "ORDER-1", got.OrderID)

	err = repo.Save(ctx, "key-1", &repositories.IdempotencyEntry{OrderID: "ORDER-2", Fingerprint: "fp-2"})
	assert.ErrorIs(t, err, repositories.ErrConflict)
}
