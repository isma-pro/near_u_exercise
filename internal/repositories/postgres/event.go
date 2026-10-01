package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ismaelucky94/near_u_exercise/internal/domain"
)

// EventRepository is the PostgreSQL-backed implementation.
type EventRepository struct {
	pool *pgxpool.Pool
}

// NewEventRepository returns a new PostgreSQL event repository.
func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

// Append stores an audit event.
func (r *EventRepository) Append(ctx context.Context, event *domain.OrderEvent) error {
	q := getQuerier(ctx, r.pool)
	query := `
		INSERT INTO order_events (order_id, type, status, timestamp, nav_used, priced_units, priced_amount, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := q.Exec(ctx, query,
		event.OrderID, string(event.Type), string(event.Status), event.Timestamp,
		event.NAVUsed, event.PricedUnits, event.PricedAmount, event.Reason,
	)
	return err
}

// ListForOrder returns audit events for an order in chronological order.
func (r *EventRepository) ListForOrder(ctx context.Context, orderID string) ([]*domain.OrderEvent, error) {
	q := getQuerier(ctx, r.pool)
	query := `SELECT order_id, type, status, timestamp, nav_used, priced_units, priced_amount, reason
		FROM order_events
		WHERE order_id = $1
		ORDER BY timestamp, id`
	rows, err := q.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.OrderEvent
	for rows.Next() {
		event := &domain.OrderEvent{}
		var eventType, status string
		if err := rows.Scan(
			&event.OrderID, &eventType, &status, &event.Timestamp,
			&event.NAVUsed, &event.PricedUnits, &event.PricedAmount, &event.Reason,
		); err != nil {
			return nil, err
		}
		event.Type = domain.EventType(eventType)
		event.Status = domain.Status(status)
		events = append(events, event)
	}
	return events, rows.Err()
}
