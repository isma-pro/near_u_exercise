package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/clock"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/handlers"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories/memory"
	"github.com/ismaelucky94/near_u_exercise/internal/routes"
	"github.com/ismaelucky94/near_u_exercise/internal/seed"
	"github.com/ismaelucky94/near_u_exercise/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter(t *testing.T) (*gin.Engine, *services.OrderService, *services.PricingService, *services.AccountService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	funds, accounts := seed.Load()
	store := memory.NewStore(funds, accounts)

	fundRepo := memory.NewFundRepository(store)
	accountRepo := memory.NewAccountRepository(store)
	orderRepo := memory.NewOrderRepository(store)
	eventRepo := memory.NewEventRepository(store)
	idempotencyRepo := memory.NewIdempotencyRepository(store)
	navRepo := memory.NewNAVRepository(store)

	accSvc := services.NewAccountService(accountRepo)
	calc := services.NewTradeDateCalculator()
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	clk := clock.FixedClock{Instant: now}
	uow := memory.NewUnitOfWork(store)
	orderSvc := services.NewOrderService(fundRepo, accSvc, orderRepo, eventRepo, idempotencyRepo, uow, calc, clk)
	pricingSvc := services.NewPricingService(fundRepo, accSvc, orderRepo, eventRepo, navRepo, uow, clk)

	r := gin.Default()
	routes.Register(r, routes.Dependencies{
		Orders:   orderSvc,
		Accounts: accSvc,
		Pricing:  pricingSvc,
		Ready:    func() bool { return true },
	})
	return r, orderSvc, pricingSvc, accSvc
}

func TestPlaceOrder_Subscription(t *testing.T) {
	r, _, _, _ := setupRouter(t)

	body := []byte(`{"account_id":"ACC-1","fund_id":"FUND-A","side":"SUBSCRIPTION","amount":"1000.00"}`)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "key-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp handlers.OrderResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ACC-1", resp.AccountID)
	assert.Equal(t, "SUBSCRIPTION", resp.Side)
	assert.Equal(t, "RECEIVED", resp.Status)
}

func TestPlaceOrder_MissingIdempotencyKey(t *testing.T) {
	r, _, _, _ := setupRouter(t)

	body := []byte(`{"account_id":"ACC-1","fund_id":"FUND-A","side":"SUBSCRIPTION","amount":"1000.00"}`)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPlaceOrder_InsufficientCash(t *testing.T) {
	r, _, _, _ := setupRouter(t)

	body := []byte(`{"account_id":"ACC-2","fund_id":"FUND-A","side":"SUBSCRIPTION","amount":"1000.00"}`)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "key-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestGetAccount(t *testing.T) {
	r, _, _, _ := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/accounts/ACC-1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp handlers.AccountResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, domain.Money(1_000_000), resp.Cash)
	assert.Equal(t, domain.Units(100_0000), resp.Positions["FUND-A"])
}

func TestPublishNAV(t *testing.T) {
	r, orderSvc, _, _ := setupRouter(t)
	ctx := context.Background()

	order, err := orderSvc.PlaceOrder(ctx, services.PlaceOrderInput{
		AccountID:      "ACC-1",
		FundID:         "FUND-A",
		Side:           domain.SideSubscription,
		Amount:         domain.Money(1000_00),
		IdempotencyKey: "key-1",
		Fingerprint:    "fp-1",
	})
	require.NoError(t, err)

	body := []byte(`{"date":"2026-10-02","nav":"10.0000"}`)
	req := httptest.NewRequest(http.MethodPost, "/funds/FUND-A/navs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	priced, err := orderSvc.GetOrder(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPriced, priced.Status)
}

func TestIdempotencyConflict(t *testing.T) {
	r, _, _, _ := setupRouter(t)

	body1 := []byte(`{"account_id":"ACC-1","fund_id":"FUND-A","side":"SUBSCRIPTION","amount":"1000.00"}`)
	req1 := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", "key-1")
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusCreated, rec1.Code)

	body2 := []byte(`{"account_id":"ACC-1","fund_id":"FUND-A","side":"SUBSCRIPTION","amount":"2000.00"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "key-1")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusConflict, rec2.Code)
}
