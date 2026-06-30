.PHONY: proto buf-lint sqlc migrate-test abigen run-indexer

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
# Regenerate Go binding from ABI (requires go-ethereum in go.mod)
abigen:
	go run github.com/ethereum/go-ethereum/cmd/abigen \
		--abi apps/web/src/lib/SkillPassCertificate.abi.json \
		--pkg binding \
		--type SkillPassCertificate \
		--out services/indexer/internal/adapter/chain/binding/skillpass.go

migrate-test:
	docker run --rm -d --name skillpass-migrate-test -e POSTGRES_PASSWORD=pg -p 55432:5432 postgres:17
	sleep 3
	go run ./services/indexer/cmd/verify-migrate/... || (docker stop skillpass-migrate-test; exit 1)
	docker stop skillpass-migrate-test

# Run the indexer locally (requires env vars — see services/indexer/internal/config/config.go)
# Required: DATABASE_URL, ETH_RPC_URL, CONTRACT_ADDRESS, CHAIN_ID
# Optional: GRPC_ADDR (":50051"), START_BLOCK ("0"), BATCH_SIZE ("2000"), POLL_INTERVAL ("5s")
run-indexer:
	go run ./services/indexer/cmd/indexer
