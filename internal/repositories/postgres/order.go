package postgres

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// OrderRepository is the PostgreSQL-backed implementation.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository returns a new PostgreSQL order repository.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Create stores a new order.
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	q := getQuerier(ctx, r.pool)
	query := `
		INSERT INTO orders (id, account_id, fund_id, side, amount, units, status, trade_date, nav_used, priced_units, priced_amount, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := q.Exec(ctx, query,
		order.ID, order.AccountID, order.FundID, string(order.Side),
		order.Amount, order.Units, string(order.Status), order.TradeDate,
		order.NAVUsed, order.PricedUnits, order.PricedAmount,
		order.IdempotencyKey, order.CreatedAt, order.UpdatedAt,
	)
	return err
}

// Get returns an order by ID. Locks the row when inside a transaction.
func (r *OrderRepository) Get(ctx context.Context, id string) (*domain.Order, error) {
	q := getQuerier(ctx, r.pool)
	lock := ""
	if inTx(ctx) {
		lock = "FOR UPDATE"
	}
	query := fmt.Sprintf(`SELECT id, account_id, fund_id, side, amount, units, status, trade_date, nav_used, priced_units, priced_amount, idempotency_key, created_at, updated_at FROM orders WHERE id = $1 %s`, lock)
	row := q.QueryRow(ctx, query, id)

	order := &domain.Order{}
	var side, status string
	if err := row.Scan(
		&order.ID, &order.AccountID, &order.FundID, &side, &order.Amount, &order.Units,
		&status, &order.TradeDate, &order.NAVUsed, &order.PricedUnits,
		&order.PricedAmount, &order.IdempotencyKey, &order.CreatedAt, &order.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	order.Side = domain.Side(side)
	order.Status = domain.Status(status)
	return order, nil
}

// List returns orders matching the filter, sorted by created_at then id.
func (r *OrderRepository) List(ctx context.Context, filter repositories.ListOrdersFilter) ([]*domain.Order, error) {
	q := getQuerier(ctx, r.pool)
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	args := []interface{}{limit}
	where := ""
	argCount := 1

	if filter.AccountID != "" {
		argCount++
		where += " AND account_id = $" + intToString(argCount)
		args = append(args, filter.AccountID)
	}
	if filter.FundID != "" {
		argCount++
		where += " AND fund_id = $" + intToString(argCount)
		args = append(args, filter.FundID)
	}
	if filter.Status != "" {
		argCount++
		where += " AND status = $" + intToString(argCount)
		args = append(args, string(filter.Status))
	}
	if filter.HasCursor {
		argCount += 2
		where += " AND (created_at, id) > ($" + intToString(argCount-1) + ", $" + intToString(argCount) + ")"
		args = append(args, filter.Cursor.CreatedAt, filter.Cursor.ID)
	}

	lock := ""
	if inTx(ctx) {
		lock = "FOR UPDATE"
	}
	query := fmt.Sprintf(`SELECT id, account_id, fund_id, side, amount, units, status, trade_date, nav_used, priced_units, priced_amount, idempotency_key, created_at, updated_at
		FROM orders
		WHERE 1=1` + where + `
		ORDER BY created_at, id
		LIMIT $1 %s`, lock)

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOrders(rows)
}

// Update replaces an existing order.
func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	q := getQuerier(ctx, r.pool)
	query := `
		UPDATE orders
		SET account_id = $1, fund_id = $2, side = $3, amount = $4, units = $5, status = $6,
		    trade_date = $7, nav_used = $8, priced_units = $9, priced_amount = $10,
		    idempotency_key = $11, updated_at = $12
		WHERE id = $13
	`
	_, err := q.Exec(ctx, query,
		order.AccountID, order.FundID, string(order.Side), order.Amount, order.Units,
		string(order.Status), order.TradeDate, order.NAVUsed, order.PricedUnits, order.PricedAmount,
		order.IdempotencyKey, order.UpdatedAt, order.ID,
	)
	return err
}

func scanOrders(rows pgx.Rows) ([]*domain.Order, error) {
	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		var side, status string
		if err := rows.Scan(
			&order.ID, &order.AccountID, &order.FundID, &side, &order.Amount, &order.Units,
			&status, &order.TradeDate, &order.NAVUsed, &order.PricedUnits,
			&order.PricedAmount, &order.IdempotencyKey, &order.CreatedAt, &order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		order.Side = domain.Side(side)
		order.Status = domain.Status(status)
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func intToString(i int) string {
	return strconv.Itoa(i)
}
