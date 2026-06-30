# SkillPass Phase 3 — Backend (Microservices) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Every Go implementer brief MUST: **invoke `golang-expert` first (hub — auto-chains go-patterns / go-review / go-test / go-error-handling / go-concurrency-patterns + `golang-grpc` on gRPC UCs + senior-backend + senior-security + algorithmic-complexity)**, and carry the Sonar-Go guardrails block (below) verbatim.

**Goal:** Build the SkillPass backend — a 2-service Go microservices system (indexer + gateway) over gRPC, with a Postgres read model — that indexes `CertificateIssued` events and serves fast, paginated, searchable certificate reads to the frontend. This is the microservices + gRPC **learning** layer.

**Architecture (from spec §8 — already designed + cross-model reviewed):** CQRS-lite. The chain is the write model; the indexer owns Postgres (read model). The gateway is a public BFF (REST + SSE) that talks to the indexer ONLY over gRPC and NEVER touches Postgres. Backend never holds keys / never mints.

**Tech Stack:** Go (single module), gRPC + buf, Postgres + sqlc + goose, go-ethereum/ethclient (event subscription), docker-compose (postgres + anvil + services). Hexagonal-lite per service.

## Global Constraints

- **Single Go module** `github.com/oksasatya/skillpass` at repo root (Go code under `services/` + `proto/gen/`). Each service = own `cmd/<name>/main.go` + own `Dockerfile`, deploys independently.
- **The indexer owns Postgres EXCLUSIVELY. The gateway must NOT import or connect to Postgres** — Go `internal/` enforces this at compile time. The ONLY shared importable code is `proto/gen`.
- **gRPC contracts are use-case-shaped, NOT table-shaped.** `ListCertificatesResponse` must not mirror a SQL row 1:1.
- **Hexagonal-lite (HARD):** `domain` imports nothing from adapter/platform; `usecase` depends on ports (interfaces); adapters (grpc/repo/ingest) implement them.
- **Indexer correctness (HARD):** idempotent (`UNIQUE(chain_id, tx_hash, log_index)`), resumable (`last_processed_block` + `block_hash`), reorg-aware storage from day 1 (full reconcile is Phase 4).
- **Sonar-Go from first commit:** `go:S107` ≤7 params (≤5 preferred → Deps/Opts struct), `go:S3776` cognitive ≤15, `go:S1192` const for 3+ duplicated literals; plus `errcheck`, `gosec`, `govulncheck`. Verify: `gofmt → go vet → golangci-lint → go test -race -cover → govulncheck`.
- **⚠️ Prerequisite:** the indexer indexes a DEPLOYED contract. Before BE can run end-to-end, deploy `SkillPassCertificate` to Base Sepolia (or run a local anvil + deploy + issue test certs). `CONTRACT_ADDRESS` + start block are indexer config.
- All work on branch `dev` (solo, no feature branches — per user).

```
# Sonar-Go guardrails — write compliant from the first commit
- go:S107 — ≤7 params (≤5 preferred; past that a Deps/Opts struct).
- go:S3776 — cognitive complexity ≤15 → extract helpers; t.Run subtests.
- go:S1192 — const for any string literal duplicated 3+ times.
- errcheck (handle every error), gosec, govulncheck. Wrap with %w; sentinel errors + errors.Is/As.
```

---

## BE Phase 1 — Indexer + data plane

Stand up the data plane: Go module, proto, Postgres, and the indexer (event worker + gRPC query server).

- **Task 1 — Go module + proto + buf:** `go mod init github.com/oksasatya/skillpass`; `proto/skillpass/cert/v1/certificate.proto` defining `service CertificateQuery { GetCertificate; ListCertificates(owner,cursor,filters); StreamCertificateEvents(stream); GetIndexerStatus }` with use-case-shaped messages; `buf.yaml` + `buf.gen.yaml`; generate + commit `proto/gen/go`. *(TDD: no — config/codegen.)*
- **Task 2 — Postgres schema (goose + sqlc):** migrations for `certificates` (token_id PK, owner_address, fields…, chain_id, tx_hash, log_index, block_number, block_hash, `UNIQUE(chain_id,tx_hash,log_index)`, indexes on owner_address + issuer_name) and `indexer_state` (last_processed_block + last_processed_hash); sqlc queries (upsert cert, get by id, list by owner with cursor, get/set indexer_state). *(TDD: no — SQL/migration; assert_no_leak-style integration test where applicable.)*
- **Task 3 — Domain + ports:** `services/indexer/internal/domain` (Certificate entity, value objects) + `usecase` ports (CertificateRepo, EventSource). Pure Go. *(TDD: yes — domain logic.)*
- **Task 4 — Postgres repo adapter:** implements CertificateRepo via sqlc; idempotent upsert; cursor pagination. *(TDD: yes — integration with testcontainers/local pg.)*
- **Task 5 — Event worker (ingest adapter):** ethclient poll/subscribe `CertificateIssued` from the deployed contract → map → repo upsert → advance `last_processed_block`+`block_hash`; resumable on restart; idempotent on replay. *(TDD: yes — decode/map + resume logic with a fake EventSource.)*
- **Task 6 — gRPC server:** implements `CertificateQuery` (GetCertificate, ListCertificates, GetIndexerStatus) over the usecase; health/readiness (ready iff DB reachable). `cmd/indexer/main.go` wires worker + gRPC server. *(TDD: yes for query mapping; no for main wiring.)*
- **Task 7 — docker-compose dev:** `deploy/docker-compose.yml` = postgres + anvil + indexer; a make target to deploy the contract to anvil + seed a couple of issuances so the indexer has data. *(TDD: no — ops.)*

**BE-1 DoD:** `docker compose up` runs postgres + anvil + indexer; the indexer ingests seeded `CertificateIssued` events idempotently + resumably into Postgres; the gRPC `ListCertificates`/`GetCertificate` return them; `golangci-lint` + `go test -race` green.

---

## BE Phase 2 — Gateway BFF + streaming + FE wiring

The public edge + realtime, and switch the frontend's list/search to the gateway.

- **Task 1 — Gateway skeleton + gRPC client:** `services/gateway` (own `internal/`, NO Postgres import); a typed gRPC client to the indexer; config (indexer addr, port). Health/readiness (ready iff indexer reachable). *(TDD: no — wiring; yes for any mapping.)*
- **Task 2 — REST API:** REST endpoints the frontend needs (`GET /certificates?owner=&cursor=&q=`, `GET /certificates/:tokenId`) → call indexer gRPC → shape responses for the UI. Graceful degradation when indexer is down (stale/last-known + clear error). *(TDD: yes — request shaping/validation.)*
- **Task 3 — Streaming (the reason gRPC earns its place):** indexer `StreamCertificateEvents` server-streaming → gateway bridges to **SSE** (`GET /certificates/stream`) → frontend live "certificate issued" updates. No Redis (single instance). *(TDD: yes for the bridge logic; no for transport.)*
- **Task 4 — Observability + graceful shutdown:** structured logs + request IDs, gRPC status metrics, index-lag gauge; graceful shutdown (reverse order). *(TDD: no — infra.)*
- **Task 5 — FE wiring:** switch the frontend My Certificates **list/search** to read from the gateway REST (paginated/searchable); keep the public verify page reading the **chain directly** (authoritative). Add a live "new certificate" SSE subscription on My Certificates. *(FE task → invoke `frontend-design`; TDD: yes for the data hook mappers.)*

**BE-2 DoD:** gateway serves paginated/searchable certs over gRPC (never touching Postgres); SSE pushes new certs to the UI; `docker compose up` runs the whole stack; FE list reads gateway, verify reads chain.

---

## Phase 4 (later, not now)
Full reorg reconcile (after N confirmations), richer observability/tracing, a 3rd domain service (metadata/notification) + Redis/asynq when async work appears, horizontal-scale fan-out.

## Execution notes
- Default writer = Claude per task; cross-model review per §18/§20 where valuable. golang-expert hub on every Go UC.
- The contract must be deployed (Base Sepolia or local anvil) before BE-1 Task 5 can ingest real events — local anvil is the dev default.
- Recommended: execute BE-1 fully (working indexer) before BE-2, mirroring the Phase 1→2 incremental discipline.
