.PHONY: proto buf-lint sqlc migrate-test

# Regenerate Go code from proto definitions (remote buf plugins, no local protoc needed)
proto:
	buf generate

# Lint proto files
buf-lint:
	buf lint

# Regenerate sqlc query layer for the indexer
sqlc:
	cd services/indexer && sqlc generate

# Verify migration applies to a throwaway Postgres (requires docker)
# Usage: make migrate-test DSN=postgres://...  (defaults to localhost:55432)
migrate-test:
	docker run --rm -d --name skillpass-migrate-test -e POSTGRES_PASSWORD=pg -p 55432:5432 postgres:17
	sleep 3
	go run ./services/indexer/cmd/verify-migrate/... || (docker stop skillpass-migrate-test; exit 1)
	docker stop skillpass-migrate-test
