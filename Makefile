.PHONY: proto buf-lint sqlc migrate-test abigen run-indexer run-gateway \
        dev-up dev-seed dev-logs dev-verify dev-down

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
# Required: DATABASE_URL, ETH_RPC_URL, CONTRACT_ADDRESS, CHAIN_ID, REDIS_ADDR
# Optional: GRPC_ADDR (":50051"), START_BLOCK ("0"), BATCH_SIZE ("2000"), POLL_INTERVAL ("5s")
run-indexer:
	go run ./services/indexer/cmd/indexer

# Run the gateway locally (requires env vars — see services/gateway/internal/config/config.go)
# Required: INDEXER_GRPC_ADDR
# Optional: HTTP_ADDR (":8080"), REQUEST_TIMEOUT ("5s")
run-gateway:
	go run ./services/gateway/cmd/gateway

# ---- dev stack (docker compose) ----
# Bring up postgres + anvil + indexer (builds indexer image)
dev-up:
	docker compose -f deploy/docker-compose.yml up -d --build

# Deploy contract + issue 2 test certificates against local anvil
dev-seed:
	bash deploy/seed.sh

# Tail indexer + gateway logs
dev-logs:
	docker compose -f deploy/docker-compose.yml logs -f indexer gateway

# Verify indexer via gRPC reflection (requires grpcurl or go run fallback)
dev-verify:
	@GRPCURL=$$(command -v grpcurl 2>/dev/null || echo "go run github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"); \
	echo "--- list services ---"; \
	$$GRPCURL -plaintext localhost:50051 list; \
	echo "--- GetIndexerStatus ---"; \
	$$GRPCURL -plaintext -d '{}' localhost:50051 skillpass.cert.v1.CertificateQuery/GetIndexerStatus; \
	echo "--- ListCertificates ---"; \
	$$GRPCURL -plaintext -d '{}' localhost:50051 skillpass.cert.v1.CertificateQuery/ListCertificates; \
	echo "--- GetCertificate token_id=0 ---"; \
	$$GRPCURL -plaintext -d '{"token_id":"0"}' localhost:50051 skillpass.cert.v1.CertificateQuery/GetCertificate; \
	echo "--- gateway /readyz ---"; \
	curl -sf http://localhost:8080/readyz; echo; \
	echo "--- gateway GET /certificates ---"; \
	curl -sf http://localhost:8080/certificates; echo

# Tear down and remove volumes
dev-down:
	docker compose -f deploy/docker-compose.yml down -v
