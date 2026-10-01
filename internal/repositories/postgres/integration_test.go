//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/db"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
	"github.com/ismaelucky94/near_u_exercise/internal/seed"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testDeps struct {
	pool      *pgxpool.Pool
	container *postgres.PostgresContainer
	funds     []*domain.Fund
	accounts  []*domain.Account
}

func setupTestDB(t *testing.T) *testDeps {
	t.Helper()

	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("nearu"),
		postgres.WithUsername("nearu"),
		postgres.WithPassword("nearu"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	pool, err := db.Open(ctx, connStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	require.NoError(t, db.MigrateFromFile(ctx, pool, "migrations/001_init.up.sql"))

	funds, accounts := seed.Load()
	require.NoError(t, Seed(ctx, pool, funds, accounts))

	return &testDeps{pool: pool, container: container, funds: funds, accounts: accounts}
}

func TestFundRepository_Get(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewFundRepository(deps.pool)

	fund, err := repo.Get(context.Background(), "FUND-A")
	require.NoError(t, err)
	assert.Equal(t, "FUND-A", fund.ID)
	assert.Equal(t, "EUR", fund.Currency)

	_, err = repo.Get(context.Background(), "UNKNOWN")
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestFundRepository_List(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewFundRepository(deps.pool)

	funds, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, funds, 2)
	assert.ElementsMatch(t, []string{"FUND-A", "FUND-B"}, []string{funds[0].ID, funds[1].ID})
}

func TestAccountRepository_Get(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewAccountRepository(deps.pool)

	acc, err := repo.Get(context.Background(), "ACC-1")
	require.NoError(t, err)
	assert.Equal(t, domain.Money(1_000_000), acc.Cash)
	assert.Equal(t, domain.Units(100_0000), acc.Positions["FUND-A"])

	_, err = repo.Get(context.Background(), "UNKNOWN")
	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestAccountRepository_Save(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewAccountRepository(deps.pool)
	uow := NewUnitOfWork(deps.pool)
	ctx := context.Background()

	acc, err := repo.Get(ctx, "ACC-1")
	require.NoError(t, err)

	acc.Cash -= domain.Money(500_00)
	acc.ReservedCash += domain.Money(500_00)

	err = uow.Run(ctx, func(txCtx context.Context) error {
		return repo.Save(txCtx, acc)
	})
	require.NoError(t, err)

	got, err := repo.Get(ctx, "ACC-1")
	require.NoError(t, err)
	assert.Equal(t, domain.Money(1_000_000-500_00), got.Cash)
	assert.Equal(t, domain.Money(500_00), got.ReservedCash)
}

func TestOrderRepository_CRUD(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewOrderRepository(deps.pool)
	ctx := context.Background()

	o := &domain.Order{
		ID:        "ORDER-1",
		AccountID: "ACC-1",
		FundID:    "FUND-A",
		Side:      domain.SideSubscription,
		Amount:    domain.Money(100_00),
		Units:     domain.Units(10_0000),
		Status:    domain.StatusReceived,
		TradeDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	require.NoError(t, repo.Create(ctx, o))

	got, err := repo.Get(ctx, "ORDER-1")
	require.NoError(t, err)
	assert.Equal(t, o.ID, got.ID)
	assert.Equal(t, o.Amount, got.Amount)

	o.Status = domain.StatusPriced
	o.NAVUsed = domain.NAV(100_0000)
	o.PricedUnits = domain.Units(10_0000)
	o.PricedAmount = domain.Money(100_00)
	require.NoError(t, repo.Update(ctx, o))

	got, err = repo.Get(ctx, "ORDER-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPriced, got.Status)
	assert.Equal(t, domain.NAV(100_0000), got.NAVUsed)
}

func TestOrderRepository_List(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewOrderRepository(deps.pool)
	ctx := context.Background()

	base := time.Now().UTC()
	for i, id := range []string{"A", "B", "C"} {
		o := &domain.Order{
			ID:        id,
			AccountID: "ACC-1",
			FundID:    "FUND-A",
			Side:      domain.SideSubscription,
			Amount:    domain.Money(100_00),
			Units:     domain.Units(10_0000),
			Status:    domain.StatusReceived,
			TradeDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			CreatedAt: base.Add(time.Duration(i) * time.Second),
			UpdatedAt: base.Add(time.Duration(i) * time.Second),
		}
		if id == "B" {
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

func TestOrderRepository_ListCursor(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewOrderRepository(deps.pool)
	ctx := context.Background()

	base := time.Now().UTC()
	for i, id := range []string{"A", "B", "C"} {
		o := &domain.Order{
			ID:        id,
			AccountID: "ACC-1",
			FundID:    "FUND-A",
			Side:      domain.SideSubscription,
			Amount:    domain.Money(100_00),
			Units:     domain.Units(10_0000),
			Status:    domain.StatusReceived,
			TradeDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			CreatedAt: base.Add(time.Duration(i) * time.Second),
			UpdatedAt: base.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, repo.Create(ctx, o))
	}

	all, err := repo.List(ctx, repositories.ListOrdersFilter{AccountID: "ACC-1", Limit: 10})
	require.NoError(t, err)
	require.Len(t, all, 3)

	second := all[1]
	filter := repositories.ListOrdersFilter{
		AccountID: "ACC-1",
		Limit:     10,
		HasCursor: true,
		Cursor: repositories.ListCursor{
			CreatedAt: second.CreatedAt,
			ID:        second.ID,
		},
	}
	got, err := repo.List(ctx, filter)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "C", got[0].ID)
}

func TestEventRepository_ListForOrder(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewEventRepository(deps.pool)
	ctx := context.Background()

	require.NoError(t, repo.Append(ctx, &domain.OrderEvent{
		OrderID: "ORDER-1",
		Type:    domain.EventReceived,
		Status:  domain.StatusReceived,
	}))
	require.NoError(t, repo.Append(ctx, &domain.OrderEvent{
		OrderID: "ORDER-1",
		Type:    domain.EventPriced,
		Status:  domain.StatusPriced,
	}))

	got, err := repo.ListForOrder(ctx, "ORDER-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, domain.EventReceived, got[0].Type)
	assert.Equal(t, domain.EventPriced, got[1].Type)
}

func TestIdempotencyRepository_Conflict(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewIdempotencyRepository(deps.pool)
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, "key-1", &repositories.IdempotencyEntry{OrderID: "ORDER-1", Fingerprint: "fp-1"}))

	got, err := repo.Get(ctx, "key-1")
	require.NoError(t, err)
	assert.Equal(t, "ORDER-1", got.OrderID)

	err = repo.Save(ctx, "key-1", &repositories.IdempotencyEntry{OrderID: "ORDER-2", Fingerprint: "fp-2"})
	assert.Error(t, err)
}

func TestNAVRepository_GetAndSave(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewNAVRepository(deps.pool)
	ctx := context.Background()
	date := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	_, ok, err := repo.Get(ctx, "FUND-A", date)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, repo.Save(ctx, "FUND-A", date, domain.NAV(123_4567)))

	got, ok, err := repo.Get(ctx, "FUND-A", date)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, domain.NAV(123_4567), got)
}

func TestUnitOfWork_Rollback(t *testing.T) {
	deps := setupTestDB(t)
	repo := NewAccountRepository(deps.pool)
	uow := NewUnitOfWork(deps.pool)
	ctx := context.Background()

	acc, err := repo.Get(ctx, "ACC-1")
	require.NoError(t, err)
	originalCash := acc.Cash

	err = uow.Run(ctx, func(txCtx context.Context) error {
		acc.Cash -= domain.Money(100_00)
		if err := repo.Save(txCtx, acc); err != nil {
			return err
		}
		return repositories.ErrConflict
	})
	assert.ErrorIs(t, err, repositories.ErrConflict)

	got, err := repo.Get(ctx, "ACC-1")
	require.NoError(t, err)
	assert.Equal(t, originalCash, got.Cash)
}
