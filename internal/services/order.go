package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/ismaelucky94/near_u_exercise/internal/clock"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// PlaceOrderInput contains everything needed to place an order.
type PlaceOrderInput struct {
	AccountID      string
	FundID         string
	Side           domain.Side
	Amount         domain.Money
	Units          domain.Units
	IdempotencyKey string
	Fingerprint    string
}

// ListOrdersResult is the output of ListOrders.
type ListOrdersResult struct {
	Orders     []*domain.Order
	NextCursor string
}

// OrderService handles order placement, cancellation and queries.
type OrderService struct {
	mu          sync.Mutex
	funds       repositories.FundRepository
	accounts    *AccountService
	orders      repositories.OrderRepository
	events      repositories.EventRepository
	idempotency repositories.IdempotencyRepository
	uow         repositories.UnitOfWork
	calc        *TradeDateCalculator
	clock       clock.Clock
	idGen       func() string
}

// NewOrderService returns a new OrderService.
func NewOrderService(
	funds repositories.FundRepository,
	accounts *AccountService,
	orders repositories.OrderRepository,
	events repositories.EventRepository,
	idempotency repositories.IdempotencyRepository,
	uow repositories.UnitOfWork,
	calc *TradeDateCalculator,
	clk clock.Clock,
) *OrderService {
	return &OrderService{
		funds:       funds,
		accounts:    accounts,
		orders:      orders,
		events:      events,
		idempotency: idempotency,
		uow:         uow,
		calc:        calc,
		clock:       clk,
		idGen:       uuid.NewString,
	}
}

// PlaceOrder places a new order or returns an existing one for a repeated idempotency key.
func (s *OrderService) PlaceOrder(ctx context.Context, input PlaceOrderInput) (*domain.Order, error) {
	if err := s.validatePlace(input); err != nil {
		return nil, err
	}

	// Serialize order placement so idempotency, reservation and order creation stay consistent.
	s.mu.Lock()
	defer s.mu.Unlock()

	var order *domain.Order
	err := s.uow.Run(ctx, func(txCtx context.Context) error {
		var err error
		order, err = s.placeInTx(txCtx, input)
		return err
	})
	return order, err
}

func (s *OrderService) placeInTx(ctx context.Context, input PlaceOrderInput) (*domain.Order, error) {
	existing, err := s.idempotency.Get(ctx, input.IdempotencyKey)
	if err == nil {
		if existing.Fingerprint != input.Fingerprint {
			return nil, fmt.Errorf("%w: idempotency key reused with different body", ErrIdempotencyConflict)
		}
		return s.orders.Get(ctx, existing.OrderID)
	}
	if err != repositories.ErrNotFound {
		return nil, err
	}

	fund, err := s.funds.Get(ctx, input.FundID)
	if err != nil {
		if err == repositories.ErrNotFound {
			return nil, fmt.Errorf("%w: %s", ErrFundNotFound, input.FundID)
		}
		return nil, err
	}

	account, err := s.accounts.Get(ctx, input.AccountID)
	if err != nil {
		if err == repositories.ErrNotFound {
			return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, input.AccountID)
		}
		return nil, err
	}

	now := s.clock.Now()
	tradeDate := s.calc.TradeDate(fund, now)

	switch input.Side {
	case domain.SideSubscription:
		if s.accounts.AvailableCash(account).Sub(input.Amount) < 0 {
			return nil, ErrInsufficientCash
		}
		s.accounts.ReserveCash(account, input.Amount)
	case domain.SideRedemption:
		if s.accounts.AvailableUnits(account, input.FundID).Sub(input.Units) < 0 {
			return nil, ErrInsufficientUnits
		}
		s.accounts.ReserveUnits(account, input.FundID, input.Units)
	}

	order := &domain.Order{
		ID:             s.idGen(),
		AccountID:      input.AccountID,
		FundID:         input.FundID,
		Side:           input.Side,
		Amount:         input.Amount,
		Units:          input.Units,
		Status:         domain.StatusReceived,
		TradeDate:      dateOnly(tradeDate),
		CreatedAt:      now,
		UpdatedAt:      now,
		IdempotencyKey: input.IdempotencyKey,
	}

	if err := s.accounts.Save(ctx, account); err != nil {
		return nil, err
	}
	if err := s.orders.Create(ctx, order); err != nil {
		return nil, err
	}
	if err := s.appendEvent(ctx, order, domain.EventReceived, ""); err != nil {
		return nil, err
	}
	if err := s.idempotency.Save(ctx, input.IdempotencyKey, &repositories.IdempotencyEntry{
		OrderID:     order.ID,
		Fingerprint: input.Fingerprint,
	}); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *OrderService) validatePlace(input PlaceOrderInput) error {
	if input.AccountID == "" || input.FundID == "" || input.IdempotencyKey == "" || input.Fingerprint == "" {
		return fmt.Errorf("%w: account_id, fund_id, idempotency key and body are required", ErrInvalidOrder)
	}
	if input.Side != domain.SideSubscription && input.Side != domain.SideRedemption {
		return fmt.Errorf("%w: side must be SUBSCRIPTION or REDEMPTION", ErrInvalidOrder)
	}
	if input.Side == domain.SideSubscription {
		if input.Amount <= 0 {
			return fmt.Errorf("%w: subscription amount must be > 0", ErrInvalidOrder)
		}
	} else {
		if input.Units <= 0 {
			return fmt.Errorf("%w: redemption units must be > 0", ErrInvalidOrder)
		}
	}
	return nil
}

// CancelOrder cancels a RECEIVED order before the cut-off of its trade date.
func (s *OrderService) CancelOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var order *domain.Order
	err := s.uow.Run(ctx, func(txCtx context.Context) error {
		var err error
		order, err = s.cancelInTx(txCtx, orderID)
		return err
	})
	return order, err
}

func (s *OrderService) cancelInTx(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := s.orders.Get(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != domain.StatusReceived {
		return nil, ErrOrderNotCancellable
	}

	fund, err := s.funds.Get(ctx, order.FundID)
	if err != nil {
		return nil, err
	}

	cutoff := fund.CutOffFor(order.TradeDate)
	if !s.clock.Now().Before(cutoff) {
		return nil, ErrOrderNotCancellable
	}

	account, err := s.accounts.Get(ctx, order.AccountID)
	if err != nil {
		return nil, err
	}

	switch order.Side {
	case domain.SideSubscription:
		s.accounts.ReleaseCash(account, order.Amount)
	case domain.SideRedemption:
		s.accounts.ReleaseUnits(account, order.FundID, order.Units)
	}

	order.Status = domain.StatusCancelled
	order.UpdatedAt = s.clock.Now()

	if err := s.accounts.Save(ctx, account); err != nil {
		return nil, err
	}
	if err := s.orders.Update(ctx, order); err != nil {
		return nil, err
	}
	if err := s.appendEvent(ctx, order, domain.EventCancelled, ""); err != nil {
		return nil, err
	}

	return order, nil
}

// GetOrder returns a single order.
func (s *OrderService) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return s.orders.Get(ctx, id)
}

// ListOrders returns a paginated list of orders.
func (s *OrderService) ListOrders(ctx context.Context, accountID string, status domain.Status, limit int, cursor string) (*ListOrdersResult, error) {
	if limit <= 0 {
		limit = 20
	}

	filter := repositories.ListOrdersFilter{
		AccountID: accountID,
		Status:    status,
		Limit:     limit + 1, // fetch one extra to decide if there is a next page
	}
	if cursor != "" {
		c, err := decodeCursor(cursor)
		if err != nil {
			return nil, ErrInvalidCursor
		}
		filter.Cursor = c
		filter.HasCursor = true
	}

	orders, err := s.orders.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := &ListOrdersResult{Orders: orders}
	if len(orders) > limit {
		result.Orders = orders[:limit]
		last := result.Orders[len(result.Orders)-1]
		result.NextCursor = encodeCursor(repositories.ListCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		})
	}
	return result, nil
}

// GetAuditEvents returns the audit trail for an order.
func (s *OrderService) GetAuditEvents(ctx context.Context, orderID string) ([]*domain.OrderEvent, error) {
	return s.events.ListForOrder(ctx, orderID)
}

func (s *OrderService) appendEvent(ctx context.Context, order *domain.Order, eventType domain.EventType, reason string) error {
	return s.events.Append(ctx, &domain.OrderEvent{
		OrderID:      order.ID,
		Type:         eventType,
		Status:       order.Status,
		Timestamp:    s.clock.Now(),
		NAVUsed:      order.NAVUsed,
		PricedUnits:  order.PricedUnits,
		PricedAmount: order.PricedAmount,
		Reason:       reason,
	})
}
