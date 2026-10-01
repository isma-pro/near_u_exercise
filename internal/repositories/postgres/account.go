package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// AccountRepository is the PostgreSQL-backed implementation.
type AccountRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepository returns a new PostgreSQL account repository.
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

// Get returns an account by ID, including its positions. When called inside a
// transaction it locks the account row and its positions to prevent overspending.
func (r *AccountRepository) Get(ctx context.Context, id string) (*domain.Account, error) {
	q := getQuerier(ctx, r.pool)
	account := &domain.Account{
		ID:            id,
		Positions:     map[string]domain.Units{},
		ReservedUnits: map[string]domain.Units{},
	}

	lock := ""
	if inTx(ctx) {
		lock = "FOR UPDATE"
	}

	query := fmt.Sprintf(`SELECT cash, reserved_cash FROM accounts WHERE id = $1 %s`, lock)
	err := q.QueryRow(ctx, query, id).Scan(&account.Cash, &account.ReservedCash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	posQuery := fmt.Sprintf(`SELECT fund_id, units, reserved_units FROM positions WHERE account_id = $1 %s`, lock)
	rows, err := q.Query(ctx, posQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var fundID string
		var units, reserved domain.Units
		if err := rows.Scan(&fundID, &units, &reserved); err != nil {
			return nil, err
		}
		account.Positions[fundID] = units
		account.ReservedUnits[fundID] = reserved
	}

	return account, nil
}

// Save updates an account's cash and reserved positions atomically within a caller-provided transaction.
func (r *AccountRepository) Save(ctx context.Context, account *domain.Account) error {
	tx, ok := TxFromContext(ctx)
	if !ok {
		return repositories.ErrNotFound
	}

	query := `UPDATE accounts SET cash = $1, reserved_cash = $2, updated_at = NOW() WHERE id = $3`
	if _, err := tx.Exec(ctx, query, account.Cash, account.ReservedCash, account.ID); err != nil {
		return err
	}

	posQuery := `
		INSERT INTO positions (account_id, fund_id, units, reserved_units)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_id, fund_id)
		DO UPDATE SET units = EXCLUDED.units, reserved_units = EXCLUDED.reserved_units
	`
	for fundID, units := range account.Positions {
		reserved := account.ReservedUnits[fundID]
		if _, err := tx.Exec(ctx, posQuery, account.ID, fundID, units, reserved); err != nil {
			return err
		}
	}

	return nil
}
