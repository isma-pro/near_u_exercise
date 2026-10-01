package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// txKey is the context key for an active transaction.
type txKey struct{}

// TxFromContext returns the current transaction, if any.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// WithTx runs fn inside a database transaction. The transaction is committed
// if fn returns nil, otherwise rolled back.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Querier adapts either a pgxpool.Pool or a pgx.Tx behind the same interface.
type Querier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// getQuerier returns the transaction from context, or the pool if none exists.
func getQuerier(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return pool
}

// inTx reports whether the context carries an active transaction.
func inTx(ctx context.Context) bool {
	_, ok := TxFromContext(ctx)
	return ok
}
