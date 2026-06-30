// verify-migrate applies embedded migrations and confirms tables+indexes exist.
// Used during CI/TDD-no verification; not part of the production binary.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oksasatya/skillpass/services/indexer/internal/platform/db"
)

func main() {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "postgres://postgres:pg@localhost:55432/postgres?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	fmt.Println("goose Up: OK")

	tables := []string{"certificates", "indexer_state"}
	for _, t := range tables {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, t,
		).Scan(&exists); err != nil || !exists {
			log.Fatalf("table %q missing: %v", t, err)
		}
		fmt.Printf("table %q: OK\n", t)
	}

	indexes := []string{"idx_certificates_owner_token", "idx_certificates_issuer_name"}
	for _, idx := range indexes {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname = $1)`, idx,
		).Scan(&exists); err != nil || !exists {
			log.Fatalf("index %q missing: %v", idx, err)
		}
		fmt.Printf("index %q: OK\n", idx)
	}

	var dataType string
	if err := pool.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns
		 WHERE table_name = 'certificates' AND column_name = 'token_id'`,
	).Scan(&dataType); err != nil {
		log.Fatalf("token_id column query: %v", err)
	}
	fmt.Printf("token_id column type in pg: %q\n", dataType)

	fmt.Println("All checks passed.")
}
