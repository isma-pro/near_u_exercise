package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate applies the supplied SQL migration.
func Migrate(ctx context.Context, pool *pgxpool.Pool, upSQL string) error {
	_, err := pool.Exec(ctx, upSQL)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// MigrateFromFile reads a migration file and applies it.
func MigrateFromFile(ctx context.Context, pool *pgxpool.Pool, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}
	return Migrate(ctx, pool, string(data))
}
