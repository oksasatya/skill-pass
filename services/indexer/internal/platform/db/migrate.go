// Package db provides database infrastructure: migrations and connection helpers.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/oksasatya/skillpass/services/indexer/migrations"
	"github.com/pressly/goose/v3"
)

const (
	gooseDriver    = "pgx"
	gooseDirection = "up"
	gooseMigrDir   = "."
)

// Migrate runs all pending goose Up migrations against the given pool.
// It opens a temporary *sql.DB via pgx stdlib for goose (goose needs *sql.DB),
// while the caller keeps the pgxpool for runtime queries.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// pgx/stdlib wraps the pool's connector — no extra connection opened.
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close() //nolint:errcheck // best-effort close of migration handle

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger()) // ponytail: silence goose banner; enable if debug needed

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db.Migrate: set dialect: %w", err)
	}

	if err := goose.RunContext(ctx, gooseDirection, sqlDB, gooseMigrDir); err != nil {
		return fmt.Errorf("db.Migrate: run %s: %w", gooseDirection, err)
	}

	return nil
}
