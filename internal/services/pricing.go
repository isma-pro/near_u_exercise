package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/clock"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// PricingService prices all received orders for a fund and trade date.
type PricingService struct {
	mu       sync.Mutex
	funds    repositories.FundRepository
	accounts *AccountService
	orders   repositories.OrderRepository
	events   repositories.EventRepository
	navs     repositories.NAVRepository
	uow      repositories.UnitOfWork
	clock    clock.Clock
}

// NewPricingService returns a new PricingService.
func NewPricingService(
	funds repositories.FundRepository,
	accounts *AccountService,
	orders repositories.OrderRepository,
	events repositories.EventRepository,
	navs repositories.NAVRepository,
	uow repositories.UnitOfWork,
	clk clock.Clock,
) *PricingService {
	return &PricingService{
		funds:    funds,
		accounts: accounts,
		orders:   orders,
		events:   events,
		navs:     navs,
		uow:      uow,
		clock:    clk,
	}
}

// PublishNAV prices every RECEIVED order for the fund and trade date.
// It is idempotent when the same NAV is published again and rejected if a
// different NAV was already published for that date.
func (s *PricingService) PublishNAV(ctx context.Context, fundID string, date time.Time, nav domain.NAV) error {
	if nav <= 0 {
		return fmt.Errorf("%w: nav must be > 0", ErrInvalidOrder)
	}

	fund, err := s.funds.Get(ctx, fundID)
	if err != nil {
		if err == repositories.ErrNotFound {
			return fmt.Errorf("%w: %s", ErrFundNotFound, fundID)
		}
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.uow.Run(ctx, func(txCtx context.Context) error {
		return s.publishInTx(txCtx, fundID, date, nav, fund)
	})
}

func (s *PricingService) publishInTx(ctx context.Context, fundID string, date time.Time, nav domain.NAV, fund *domain.Fund) error {
	navDate := dateOnly(date)

	if existing, ok, _ := s.navs.Get(ctx, fundID, navDate); ok {
		if existing == nav {
			return nil
		}
		return ErrNAVAlreadyPriced
	}

	targetOrders, err := s.orders.List(ctx, repositories.ListOrdersFilter{
		FundID:    fundID,
		Status:    domain.StatusReceived,
		Limit:     10000,
		HasCursor: false,
	})
	if err != nil {
		return err
	}

	var toPrice []*domain.Order
	for _, o := range targetOrders {
		if o.FundID != fundID || o.Status != domain.StatusReceived || !dateOnly(o.TradeDate).Equal(navDate) {
			continue
		}
		toPrice = append(toPrice, o)
	}

	// Precompute all changes. If any account is missing we fail before mutating state.
	accountCache := make(map[string]*domain.Account)
	updates := make([]priceUpdate, 0, len(toPrice))
	now := s.clock.Now()

	for _, order := range toPrice {
		acc, ok := accountCache[order.AccountID]
		if !ok {
			var err error
			acc, err = s.accounts.Get(ctx, order.AccountID)
			if err != nil {
				return fmt.Errorf("account %s not found during pricing: %w", order.AccountID, err)
			}
			accountCache[order.AccountID] = acc
		}

		update := priceUpdate{order: order, account: acc, now: now}
		switch order.Side {
		case domain.SideSubscription:
			units := domain.UnitsForSubscription(order.Amount, nav)
			update.units = units
			update.amount = order.Amount
		case domain.SideRedemption:
			proceeds := domain.MoneyForRedemption(order.Units, nav)
			update.units = order.Units
			update.amount = proceeds
		}
		updates = append(updates, update)
	}

	// All prechecks passed: apply every update and the NAV atomically.
	if err := s.navs.Save(ctx, fundID, navDate, nav); err != nil {
		return err
	}

	for _, u := range updates {
		switch u.order.Side {
		case domain.SideSubscription:
			s.accounts.ApplySubscription(u.account, fundID, u.amount, u.units)
		case domain.SideRedemption:
			s.accounts.ApplyRedemption(u.account, fundID, u.units, u.amount)
		}

		u.order.Status = domain.StatusPriced
		u.order.NAVUsed = nav
		u.order.PricedUnits = u.units
		u.order.PricedAmount = u.amount
		u.order.UpdatedAt = now

		if err := s.accounts.Save(ctx, u.account); err != nil {
			return err
		}
		if err := s.orders.Update(ctx, u.order); err != nil {
			return err
		}
		if err := s.events.Append(ctx, &domain.OrderEvent{
			OrderID:      u.order.ID,
			Type:         domain.EventPriced,
			Status:       domain.StatusPriced,
			Timestamp:    now,
			NAVUsed:      nav,
			PricedUnits:  u.units,
			PricedAmount: u.amount,
		}); err != nil {
			return err
		}
	}

	_ = fund
	return nil
}

type priceUpdate struct {
	order   *domain.Order
	account *domain.Account
	units   domain.Units
	amount  domain.Money
	now     time.Time
}
