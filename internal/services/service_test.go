package services

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/clock"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories/memory"
	"github.com/ismaelucky94/near_u_exercise/internal/seed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupServices(t *testing.T, now time.Time) (*OrderService, *PricingService, *AccountService, *memory.Store) {
	t.Helper()
	funds, accounts := seed.Load()
	store := memory.NewStore(funds, accounts)

	fundRepo := memory.NewFundRepository(store)
	accountRepo := memory.NewAccountRepository(store)
	orderRepo := memory.NewOrderRepository(store)
	eventRepo := memory.NewEventRepository(store)
	idempotencyRepo := memory.NewIdempotencyRepository(store)
	navRepo := memory.NewNAVRepository(store)

	accSvc := NewAccountService(accountRepo)
	calc := NewTradeDateCalculator()
	clk := clock.FixedClock{Instant: now}
	orderSvc := NewOrderService(fundRepo, accSvc, orderRepo, eventRepo, idempotencyRepo, calc, clk)
	pricingSvc := NewPricingService(fundRepo, accSvc, orderRepo, eventRepo, navRepo, clk)

	return orderSvc, pricingSvc, accSvc, store
}

func TestOrderService_PlaceSubscription(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	orderSvc, _, accSvc, _ := setupServices(t, now)
	ctx := context.Background()

	order, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(1000_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.StatusReceived, order.Status)
	assert.Equal(t, "2026-10-02", order.TradeDate.Format("2006-01-02"))

	acc, err := accSvc.Get(ctx, "ACC-1")
	require.NoError(t, err)
	assert.Equal(t, domain.Money(1000_00), acc.ReservedCash)
	assert.Equal(t, domain.Money(1000_00), acc.Cash.Sub(acc.AvailableCash()))
}

func TestOrderService_InsufficientCash(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	orderSvc, _, _, _ := setupServices(t, now)
	ctx := context.Background()

	_, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-2",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(1000_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	assert.ErrorIs(t, err, ErrInsufficientCash)
}

func TestOrderService_ConcurrentRedemptions(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	orderSvc, _, accSvc, _ := setupServices(t, now)
	ctx := context.Background()

	const goroutines = 150
	var wg sync.WaitGroup
	successes := make(chan *domain.Order, goroutines)
	failures := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			order, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
				AccountID:      "ACC-1",
				FundID:         "FUND-A",
				Side:           domain.SideRedemption,
				Units:          domain.Units(1_0000),
				IdempotencyKey: fmt.Sprintf("key-%d", i),
				Fingerprint:    fmt.Sprintf("fp-%d", i),
			})
			if err != nil {
				failures <- err
				return
			}
			successes <- order
		}(i)
	}
	wg.Wait()
	close(successes)
	close(failures)

	successCount := 0
	for range successes {
		successCount++
	}
	failureCount := 0
	for err := range failures {
		if err == ErrInsufficientUnits {
			failureCount++
		}
	}

	assert.Equal(t, 100, successCount, "only 100 one-unit redemptions should reserve")
	assert.Equal(t, 50, failureCount)

	acc, err := accSvc.Get(ctx, "ACC-1")
	require.NoError(t, err)
	assert.LessOrEqual(t, domain.Units(0), acc.AvailableUnits("FUND-A"))
	assert.Equal(t, domain.Units(100_0000), acc.ReservedUnits["FUND-A"])
}

func TestOrderService_CancelBeforeCutoff(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	orderSvc, _, accSvc, _ := setupServices(t, now)
	ctx := context.Background()

	order, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(100_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	require.NoError(t, err)

	cancelled, err := orderSvc.CancelOrder(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusCancelled, cancelled.Status)

	acc, err := accSvc.Get(ctx, "ACC-1")
	require.NoError(t, err)
	assert.Equal(t, domain.Money(0), acc.ReservedCash)
}

func TestOrderService_CancelAfterCutoff(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	// 13:00 is after the 12:00 cutoff, so the trade date is the next business day (Monday).
	now := time.Date(2026, 10, 2, 13, 0, 0, 0, lisbon)
	orderSvc, _, _, _ := setupServices(t, now)
	ctx := context.Background()

	order, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(100_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-10-05", order.TradeDate.Format("2006-01-02"))

	// Still on Friday before Monday's cutoff, so cancellation should succeed.
	_, err = orderSvc.CancelOrder(ctx, order.ID)
	require.NoError(t, err)
}

func TestPricingService_PublishNAV(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	orderSvc, pricingSvc, accSvc, _ := setupServices(t, now)
	ctx := context.Background()
	tradeDate := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)

	order, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(1000_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	require.NoError(t, err)

	err = pricingSvc.PublishNAV(ctx, "FUND-A", tradeDate, domain.NAV(10_0000))
	require.NoError(t, err)

	priced, err := orderSvc.GetOrder(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPriced, priced.Status)
	assert.Equal(t, domain.Units(10_0000), priced.PricedUnits)

	acc, err := accSvc.Get(ctx, "ACC-1")
	require.NoError(t, err)
	assert.Equal(t, domain.Money(1000_0000-1000_00), acc.Cash)
	assert.Equal(t, domain.Units(100_0000+10_0000), acc.Positions["FUND-A"])
}

func TestPricingService_IdempotentNAV(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	orderSvc, pricingSvc, _, _ := setupServices(t, now)
	ctx := context.Background()
	tradeDate := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)

	order, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(1000_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	require.NoError(t, err)

	err = pricingSvc.PublishNAV(ctx, "FUND-A", tradeDate, domain.NAV(10_0000))
	require.NoError(t, err)

	err = pricingSvc.PublishNAV(ctx, "FUND-A", tradeDate, domain.NAV(10_0000))
	require.NoError(t, err)

	_, err = orderSvc.GetOrder(ctx, order.ID)
	require.NoError(t, err)
}

func TestPricingService_NavMismatch(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	_, pricingSvc, _, _ := setupServices(t, now)
	ctx := context.Background()
	tradeDate := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)

	require.NoError(t, pricingSvc.PublishNAV(ctx, "FUND-A", tradeDate, domain.NAV(10_0000)))
	err := pricingSvc.PublishNAV(ctx, "FUND-A", tradeDate, domain.NAV(11_0000))
	assert.ErrorIs(t, err, ErrNAVAlreadyPriced)
}

func TestOrderService_Idempotency(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	orderSvc, _, _, _ := setupServices(t, now)
	ctx := context.Background()

	first, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(100_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	require.NoError(t, err)

	second, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(100_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	_, err = orderSvc.PlaceOrder(ctx, PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(200_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-2",
	})
	assert.ErrorIs(t, err, ErrIdempotencyConflict)
}

func TestOrderService_ListOrders(t *testing.T) {
	lisbon, _ := time.LoadLocation("Europe/Lisbon")
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, lisbon)
	orderSvc, _, _, _ := setupServices(t, now)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := orderSvc.PlaceOrder(ctx, PlaceOrderInput{
			AccountID:      "ACC-1",
			FundID:         "FUND-A",
			Side:           domain.SideSubscription,
			Amount:         domain.Money(1_00),
			IdempotencyKey: fmt.Sprintf("list-key-%d", i),
			Fingerprint:    fmt.Sprintf("list-fp-%d", i),
		})
		require.NoError(t, err)
	}

	result, err := orderSvc.ListOrders(ctx, "ACC-1", domain.StatusReceived, 2, "")
	require.NoError(t, err)
	assert.Len(t, result.Orders, 2)
	assert.NotEmpty(t, result.NextCursor)

	result2, err := orderSvc.ListOrders(ctx, "ACC-1", domain.StatusReceived, 2, result.NextCursor)
	require.NoError(t, err)
	assert.Len(t, result2.Orders, 2)
	assert.NotEqual(t, result.Orders[0].ID, result2.Orders[0].ID)
}
