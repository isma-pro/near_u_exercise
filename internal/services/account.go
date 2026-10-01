package services

import (
	"context"

	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// AccountService encapsulates account-level operations.
type AccountService struct {
	repo repositories.AccountRepository
}

// NewAccountService returns a new AccountService.
func NewAccountService(repo repositories.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

// Get returns an account by ID.
func (s *AccountService) Get(ctx context.Context, id string) (*domain.Account, error) {
	return s.repo.Get(ctx, id)
}

// Save persists an account, ensuring maps are non-nil.
func (s *AccountService) Save(ctx context.Context, account *domain.Account) error {
	if account.Positions == nil {
		account.Positions = map[string]domain.Units{}
	}
	if account.ReservedUnits == nil {
		account.ReservedUnits = map[string]domain.Units{}
	}
	return s.repo.Save(ctx, account)
}

// AvailableCash returns cash minus reserved cash.
func (s *AccountService) AvailableCash(account *domain.Account) domain.Money {
	return account.Cash.Sub(account.ReservedCash)
}

// AvailableUnits returns held units minus reserved units for a fund.
func (s *AccountService) AvailableUnits(account *domain.Account, fundID string) domain.Units {
	return account.Positions[fundID].Sub(account.ReservedUnits[fundID])
}

// ReserveCash reserves cash for a subscription.
func (s *AccountService) ReserveCash(account *domain.Account, amount domain.Money) {
	account.ReservedCash = account.ReservedCash.Add(amount)
}

// ReleaseCash releases reserved cash.
func (s *AccountService) ReleaseCash(account *domain.Account, amount domain.Money) {
	account.ReservedCash = account.ReservedCash.Sub(amount)
}

// ReserveUnits reserves units for a redemption.
func (s *AccountService) ReserveUnits(account *domain.Account, fundID string, units domain.Units) {
	account.ReservedUnits[fundID] = account.ReservedUnits[fundID].Add(units)
}

// ReleaseUnits releases reserved units.
func (s *AccountService) ReleaseUnits(account *domain.Account, fundID string, units domain.Units) {
	account.ReservedUnits[fundID] = account.ReservedUnits[fundID].Sub(units)
}

// ApplySubscription debits cash, releases the reservation and adds units.
func (s *AccountService) ApplySubscription(account *domain.Account, fundID string, amount domain.Money, units domain.Units) {
	account.Cash = account.Cash.Sub(amount)
	account.ReservedCash = account.ReservedCash.Sub(amount)
	account.Positions[fundID] = account.Positions[fundID].Add(units)
}

// ApplyRedemption removes units, releases the reservation and credits proceeds.
func (s *AccountService) ApplyRedemption(account *domain.Account, fundID string, units domain.Units, proceeds domain.Money) {
	account.Positions[fundID] = account.Positions[fundID].Sub(units)
	account.ReservedUnits[fundID] = account.ReservedUnits[fundID].Sub(units)
	account.Cash = account.Cash.Add(proceeds)
}
