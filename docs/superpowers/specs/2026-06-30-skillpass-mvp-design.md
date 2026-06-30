# SkillPass — Design Spec (v0.1)

- **Status:** Approved design, ready for `writing-plans` (Phase 1 first)
- **Date:** 2026-06-30
- **Product:** SkillPass — open-source onchain certificate / achievement badge platform
- **Tagline:** Open-source onchain certificate and achievement badge platform
- **Repo:** `skillpass` (monorepo)

---

## 1. Goal

### Product goal
Prove that a platform can issue a certificate/badge to a user's wallet, store the proof of issuance on a blockchain, and display that certificate in the user's dashboard and on a public verification page. Certificates are **non-transferable (soulbound)** so they stay bound to the recipient and cannot be sold or moved.

### Learning goal (explicit, drives the architecture)
The owner is a senior Go/Java backend developer using this project to **learn microservices + gRPC**. This shapes Phase 3: the backend is not bolted on for its own sake — it exists to do real Web3 backend work (event indexing) that the chain cannot do well, and it is kept to the **smallest topology that genuinely teaches microservices without teaching bad habits** (no distributed monolith, no fake service boundaries).

The two goals are kept separate by the phased build order (Section 4): Phases 1–2 ship a complete, portfolio-ready dApp with **no backend**; Phase 3 layers the microservices learning on top of a working foundation.

---

## 2. Roles

- **Issuer / Admin** — the party that issues certificates. For v0.1 this is the single contract owner (deployer). Examples: bootcamp, course creator, developer community, OSS maintainer, internal org.
- **Recipient / User** — receives a certificate. Examples: course participant, GitHub contributor, community member, event attendee.
- **Visitor** — anyone opening a public certificate detail page to verify authenticity. No wallet required.

---

## 3. Scope

### In scope (v0.1, across phases)
1. Connect wallet (wagmi + viem); detect + validate network = Base Sepolia (`chainId 84532`); show address; disconnect.
2. Admin issues a certificate (NFT) to a recipient wallet — admin signs the mint transaction in the browser.
3. User views certificates owned by their connected wallet.
4. Public certificate detail page by token ID, verifiable against the chain, works without a wallet.
5. Certificate is non-transferable (soulbound).
6. Display transaction hash + block explorer link after issuing.
7. **(Phase 3)** Backend read-projection: indexer + gateway over gRPC, Postgres read model, paginated/searchable listing, live "certificate issued" stream.

### Out of scope (v0.1)
Payment, tokenomics, marketplace, staking, DAO, multichain, email login, PDF generator, automatic IPFS upload, organization billing, team management, analytics dashboard, mobile app, on-chain SVG/metadata generation, WalletConnect, AccessControl/roles, a relayer that mints on the admin's behalf.

---

## 4. Phased build order (decision: incremental)

Each phase ships something that runs. Each phase gets its own `writing-plans` plan → implementation cycle.

| Phase | Deliverable | Backend? | Why this order |
|------|-------------|----------|----------------|
| **1. Contract** | `SkillPassCertificate` (Foundry), tested, deployed to Base Sepolia, ABI + address exported | No | Foundation; everything reads/writes it |
| **2. Frontend dApp (direct-onchain)** | Vite SPA: connect, issue, my-certificates, public verify — reads chain directly via event logs | No | A complete, portfolio-ready dApp with zero backend. This is the original MVP |
| **3. Backend (microservices)** | `indexer` + `gateway` (Go), gRPC, Postgres read model, streaming; frontend list/search switches to gateway; verify stays on-chain | Yes | The microservices + gRPC learning layer, built on a working foundation |
| **4. Increments (later)** | Full reorg reconcile, observability/metrics, metadata/notification service + Redis/asynq | Yes | Added when there is a real reason; not v0.1 |

**Key property:** after Phase 2, SkillPass is a finished, demoable product. Phase 3 is additive — if it stalls, Phase 2 still stands.

---

## 5. Architecture

### 5.1 CQRS-lite (chain = write model, indexer = read model)

```
[Wallet] --sign mint tx--> [SkillPassCertificate]      WRITE model (source of truth)
                                  | emit CertificateIssued
                                  v
   [indexer svc] --poll/subscribe--> [Postgres]         READ model (projection)
        | gRPC server (use-case queries + stream)
        v
   [gateway svc] --REST--> [Frontend Vite SPA]
                              | (verify page reads chain directly via viem)
                              v
                         [SkillPassCertificate]
```

### 5.2 Hard architectural rules (the guards against a fake-boundary distributed monolith)

1. **Backend never holds a private key and never mints.** All writes are wallet→contract, signed in the browser by the admin. The backend is read-only relative to the chain. (Security: no custody, no key surface.)
2. **The indexer owns Postgres exclusively. The gateway must NOT touch Postgres directly** — it asks the indexer over gRPC. This is the single structural guard that keeps the service boundary real.
3. **gRPC contracts are use-case-shaped, not table-shaped.** If a response message mirrors a SQL row 1:1, the boundary is fake. Queries are domain operations (`ListCertificates(owner, cursor, filters)`), not "give me rows".
4. **gRPC is internal only** (gateway ↔ indexer). The browser talks REST to the gateway. A **server-streaming** RPC (`StreamCertificateEvents`) is built so gRPC earns its place over plain request/response and provides realtime without Redis.
5. **Chain is the source of truth; the indexer is a convenience projection.** The UI is explicit about this: discovery/search is served from the indexer ("indexed as of block N"); single-certificate verification reads from the chain directly.

### 5.3 Monorepo + the gateway is a BFF (not a heavyweight gateway product)
- **Monorepo, deliberately.** Microservices best practice is independent deployability + data ownership, *not* repo count. Each service still has its own binary, Dockerfile, process, and DB ownership and deploys independently — the monorepo only co-locates the code, shared `proto`, and one `docker compose`. Polyrepo's benefits (per-team ownership, independent CI/release cadence, per-repo access control) are multi-team org concerns that do not apply to a solo project; for a solo learner polyrepo only adds proto version-skew and cross-repo coordination overhead.
- **The `gateway` is a custom Backend-For-Frontend (BFF), an instance of the API-Gateway pattern — not Kong/Envoy/Traefik/AWS API Gateway.** A heavyweight gateway product in front of two services is over-engineering. The BFF earns its place by doing real work: REST/SSE↔gRPC translation, being the single public entry point (the indexer is never internet-exposed), and bridging the realtime stream. A gateway *product* becomes worth it only at many services (5+) needing centralized routing/rate-limit/auth.

### 5.4 Repository layout (monorepo, single Go module)

```
skillpass/
├── apps/web/                          # Vite SPA (Phase 2)
│   └── src/{components/{wallet,certificate,ui},hooks,lib,routes}
├── contracts/                         # Foundry (Phase 1)
│   ├── src/SkillPassCertificate.sol
│   ├── script/Deploy.s.sol
│   ├── test/SkillPassCertificate.t.sol
│   └── foundry.toml
├── proto/                             # gRPC contract — the ONLY shared code (Phase 3)
│   ├── skillpass/cert/v1/certificate.proto
│   ├── buf.yaml · buf.gen.yaml
│   └── gen/go/skillpass/cert/v1/      # generated, committed (buf, not raw protoc)
├── services/                          # Go backend (Phase 3)
│   ├── gateway/                       # BFF: REST+SSE public, gRPC client. NO DB.
│   │   ├── cmd/gateway/main.go
│   │   ├── internal/{http,grpcclient,config}/
│   │   └── Dockerfile
│   └── indexer/                       # owns Postgres, gRPC server, event worker
│       ├── cmd/indexer/main.go
│       ├── internal/
│       │   ├── domain/                # Certificate entity — pure Go (hexagonal core)
│       │   ├── usecase/               # query + ingest use cases
│       │   ├── grpcserver/            # implements proto CertificateQuery
│       │   ├── ingest/                # chain event worker (ethclient) + reorg/resume
│       │   ├── repo/{migrations(goose),queries(sqlc),sqlc.yaml}  # owns DB
│       │   └── config/
│       └── Dockerfile
├── deploy/docker-compose.yml          # postgres + gateway + indexer + anvil
├── go.mod                             # single module: github.com/oksasatya/skillpass
├── Makefile                           # buf generate · migrate · test · run
└── README.md
```

**Layout rules (these encode the architecture, not just tidiness):**
1. **Only `proto/gen` is shared.** No shared domain/business package imported by both services — a shared domain package is the classic fake-boundary / distributed-monolith smell.
2. **Go `internal/` enforces the boundary at compile time.** `services/gateway` cannot import `services/indexer/internal/...` — the Go toolchain forbids it. The "gateway never touches the indexer's domain/DB" rule is compiler-enforced, not just convention.
3. **Single `go.mod`** (`github.com/oksasatya/skillpass`); each service has its own `cmd/<name>/main.go` + `Dockerfile` and deploys independently — that (not repo/module count) is what makes it microservices. `go.work` with per-service modules is an optional later step if full module independence is wanted.
4. **`buf`** generates proto stubs (modern best practice over raw `protoc`); generated code is committed.
5. **`indexer` is hexagonal-lite:** `domain` (pure, imports nothing) ← `usecase` ← adapters (`grpcserver`, `ingest`, `repo`); only `repo/` speaks Postgres (sqlc + goose). The `gateway` has no `domain/` — it is a thin translation layer.

### 5.5 Why no Redis (yet)
At single-instance scale, Postgres + an in-memory LRU (if even needed) is enough, and gRPC server-streaming delivers the realtime "new certificate" feed without a message broker. Redis enters only when (a) the gateway is scaled horizontally and needs cross-instance fan-out, or (b) a job queue (asynq) appears for async work (OG image generation, email invites) — both Phase 4.

---

## 6. Smart contract — `SkillPassCertificate` (Phase 1)

Solidity `^0.8.24`, Foundry, **OpenZeppelin Contracts 5.x**. `ERC721` + `Ownable` + ERC-5192 (soulbound signal).

### 6.1 Storage
```solidity
struct Certificate {
    string  title;
    string  recipientName;
    string  issuerName;
    string  description;
    string  metadataURI;   // optional; "" allowed
    uint256 issuedAt;      // block.timestamp at mint
}

uint256 private _nextTokenId = 1;
mapping(uint256 tokenId => Certificate) private _certificates;
```
- **No `tokenId` field in the struct** — it is the mapping key (redundant).
- **No `recipient` field in the struct** — `ownerOf(tokenId)` is authoritative and never changes (soulbound). Synthesized in `getCertificate`.
- **No `mapping(address => uint256[]) _ownedTokens`** — deliberately dropped. An unbounded on-chain owner array is an anti-pattern (unbounded read, duplicate ownership truth). Owner enumeration is done off-chain from events (Phase 2 via `getLogs`, Phase 3 via the indexer).

### 6.2 OpenZeppelin 5.x correctness (these are the landmines — implement exactly)
- **Soulbound** via `_update` override using the **previous-owner** check (not `auth`):
  ```solidity
  function _update(address to, uint256 tokenId, address auth) internal override returns (address) {
      address from = _ownerOf(tokenId);
      if (from != address(0) && to != address(0)) revert Soulbound(); // block transfers; allow mint (from==0) and burn (to==0)
      return super._update(to, tokenId, auth);
  }
  ```
- **No `Counters`** (removed in 5.x) → plain `uint256 _nextTokenId`.
- **`Ownable(initialOwner)`** constructor argument is required in 5.x.
- **Removed APIs:** `_exists`, `_beforeTokenTransfer`, `_afterTokenTransfer` are gone. Use `_ownerOf`, `_requireOwned`, `_update`.
- **Block meaningless approvals** on a non-transferable token:
  ```solidity
  function approve(address, uint256) public override { revert ApprovalDisabled(); }
  function setApprovalForAll(address, bool) public override { revert ApprovalDisabled(); }
  ```
- **ERC-5192** (`locked(tokenId) => true`; emit `Locked(tokenId)` at mint; advertise interface id `0xb45a3c0e` in `supportsInterface`).

### 6.3 Functions
```solidity
function issueCertificate(
    address recipient,
    string calldata title,
    string calldata recipientName,
    string calldata issuerName,
    string calldata description,
    string calldata metadataURI
) external onlyOwner returns (uint256 tokenId);

function getCertificate(uint256 tokenId) external view returns (Certificate memory cert, address recipient);
function tokenURI(uint256 tokenId) public view override returns (string memory); // returns metadataURI or ""
function locked(uint256 tokenId) external view returns (bool); // always true
function totalSupply() external view returns (uint256); // _nextTokenId - 1
```

`issueCertificate` body (order matters — CEI):
1. `if (recipient == address(0)) revert ZeroRecipient();`
2. Length-guard every string against `MAX_*` constants → `revert StringTooLong()` (unbounded strings make gas unpredictable).
3. `tokenId = _nextTokenId++;`
4. Store `_certificates[tokenId]` (with `issuedAt = block.timestamp`) **before** minting.
5. `emit CertificateIssued(...); emit Locked(tokenId);`
6. `_safeMint(recipient, tokenId);` (external call last — `_safeMint` may call a contract recipient).

### 6.4 Events
```solidity
event CertificateIssued(
    uint256 indexed tokenId,
    address indexed recipient,
    string  title,
    string  issuerName,
    uint256 issuedAt
);
event Locked(uint256 tokenId); // ERC-5192
```
`tokenId` and `recipient` are indexed — required so Phase 2 (`getLogs` by recipient) and Phase 3 (indexer) can filter efficiently.

### 6.5 Custom errors
`Soulbound()`, `ApprovalDisabled()`, `ZeroRecipient()`, `StringTooLong()` (gas-cheaper than string reverts).

### 6.6 Tests (Foundry, TDD — see §10)
Owner-only issue (non-owner reverts), zero-address recipient reverts, string-too-long reverts, getter round-trip, `CertificateIssued` + `Locked` emission, transfer reverts (`transferFrom` + both `safeTransferFrom` overloads), `approve`/`setApprovalForAll` revert, `locked()` returns true, `tokenURI` returns stored URI / `""`, `totalSupply` increments.

### 6.7 Deployment
`script/Deploy.s.sol` deploys to Base Sepolia, verifies on Basescan, and the workflow **exports ABI + deployed address** into the frontend (`apps/web/lib/contract.ts`) and `proto`/indexer config. A reproducible deploy is part of Definition of Done.

---

## 7. Frontend — Vite SPA (Phase 2, evolves in Phase 3)

Vite + React + TypeScript + Tailwind + shadcn/ui. wagmi + viem. Client-side router (TanStack Router standalone or React Router).

### 7.1 Routes
| Route | Purpose | Auth |
|------|---------|------|
| `/` | Landing | public |
| `/app` | Dashboard: connect wallet, network status | wallet |
| `/app/issue` | Issue form | owner-only (UI reads `owner()`; hidden/disabled for non-owners) |
| `/app/my-certificates` | Certificates owned by connected wallet | wallet |
| `/certificates/:tokenId` | Public verification detail | **public, no wallet** |

### 7.2 Connectors + network
- Connectors: **injected (MetaMask) + Coinbase Wallet** (Base-native). WalletConnect deferred.
- `NetworkGuard`: if `chainId !== 84532`, show a banner + `switchChain` button and disable all write actions.

### 7.3 Read strategy (the hybrid)
- **My Certificates / lists (Phase 2):** read `CertificateIssued` logs filtered by the indexed `recipient` via viem (`getContractEvents` / `getLogs`), then `getCertificate(tokenId)` per token (batched with viem `multicall` — see §9). **Phase 3:** this list switches to the gateway REST API (paginated/searchable).
- **Public verify `/certificates/:tokenId` (all phases):** reads from the **chain directly** via a viem public client. This page must prove against the chain, not a centralized index. Phase 3 may enrich with indexer data but the authoritative read stays on-chain. Shows "verified on-chain" + contract address + explorer link.

### 7.4 Hooks + components
- `hooks/`: `useCertificates` (owner list + per-token detail), `useIssueCertificate` (`writeContract` → `waitForTransactionReceipt`), `useIsAdmin` (`owner()` vs connected address).
- `components/wallet/`: `ConnectButton`, `NetworkGuard`. `components/certificate/`: `CertificateCard`, `CertificateDetail`, `IssueForm`. `components/ui/`: shadcn primitives.
- `lib/`: `wagmi.ts` (chains/connectors/transports), `contract.ts` (address + ABI), `chains.ts`.

### 7.5 Required UX (from cross-model review)
- **Privacy disclosure (mandatory):** `recipientName`, `title`, `description` become **permanent and public on-chain**. The `IssueForm` must warn the admin of this irreversibility before submission, with an explicit acknowledgement.
- Address validation (`isAddress`) and string length limits mirroring the contract `MAX_*` constants.
- Every read has explicit loading / error / empty states.
- Detail page: invalid/non-existent token id → not-found state.
- After issuing: success state with token id, transaction hash, and Basescan link (`sepolia.basescan.org`).

### 7.6 Card / detail fields
- **Card:** Title, Issuer, Recipient Name, Issued Date, Token ID.
- **Detail:** Title, Description, Recipient Name, Recipient Wallet, Issuer Name, Issued Date, Token ID, Contract Address, Metadata URI, Explorer link.

---

## 8. Backend — microservices (Phase 3)

Two Go services in the monorepo. gRPC inside, REST outside. Postgres owned exclusively by the indexer.

### 8.1 `indexer` service (owns Postgres)
- **Event worker:** poll/subscribe `CertificateIssued` from Base Sepolia (go-ethereum / ethclient), upsert into Postgres, advance `last_processed_block` (+ block hash). Resumable: on restart, continues from the stored block. Idempotent: replays do not double-insert.
- **gRPC server:** use-case-shaped queries + a server-streaming feed.
- **Health/readiness:** ready only if the DB is reachable.

### 8.2 `gateway` service (frontend-facing, no DB access)
- Public REST API consumed by the frontend; request shaping/validation. **No auth in v0.1** — the gateway serves only public certificate data, so there is nothing to protect; admin gating for issuing is enforced on-chain (`onlyOwner`), not by the gateway. SIWE (Sign-In-With-Ethereum) is noted as a clean Phase-4 learning add once private data exists (issuer profiles, drafts).
- Calls the indexer over gRPC for all certificate data. **Never connects to Postgres.**
- Exposes SSE/WebSocket to the browser backed by the indexer's `StreamCertificateEvents`.
- **Health/readiness:** ready only if the indexer is reachable; degrades gracefully (serves stale/last-known state, surfaces "indexer unavailable") when it is not.

### 8.3 gRPC contract (`proto/`, use-case-shaped)
```proto
service CertificateQuery {
  rpc GetCertificate (GetCertificateRequest) returns (Certificate);
  rpc ListCertificates (ListCertificatesRequest) returns (ListCertificatesResponse); // owner + cursor + filters
  rpc StreamCertificateEvents (StreamRequest) returns (stream CertificateEvent);     // realtime; the reason gRPC is here
  rpc GetIndexerStatus (GetIndexerStatusRequest) returns (IndexerStatus);            // block lag, healthy
}
```
- Messages are domain-shaped (cursor pagination, filter fields), **not** raw table rows. `ListCertificatesResponse` must not be a 1:1 mirror of the `certificates` row.
- `proto/` holds the schema; generated code is committed; removed field numbers are reserved (versioning discipline).

### 8.4 Postgres read model (goose migrations + sqlc, from day one)
```sql
CREATE TABLE certificates (
    token_id        BIGINT PRIMARY KEY,
    owner_address   TEXT        NOT NULL,
    title           TEXT        NOT NULL,
    recipient_name  TEXT        NOT NULL,
    issuer_name     TEXT        NOT NULL,
    description     TEXT        NOT NULL,
    metadata_uri    TEXT,
    issued_at       TIMESTAMPTZ NOT NULL,
    chain_id        BIGINT      NOT NULL,
    tx_hash         TEXT        NOT NULL,
    log_index       INTEGER     NOT NULL,
    block_number    BIGINT      NOT NULL,
    block_hash      TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chain_id, tx_hash, log_index)   -- idempotency: replays cannot double-insert
);
CREATE INDEX idx_certificates_owner  ON certificates (owner_address);
CREATE INDEX idx_certificates_issuer ON certificates (issuer_name);

CREATE TABLE indexer_state (
    id                   SMALLINT PRIMARY KEY DEFAULT 1,
    chain_id             BIGINT      NOT NULL,
    last_processed_block BIGINT      NOT NULL,
    last_processed_hash  TEXT        NOT NULL,   -- enables reorg detection
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 8.5 Indexing correctness (learning-grade)
- **Idempotency:** the `UNIQUE (chain_id, tx_hash, log_index)` constraint + upsert.
- **Resume:** `last_processed_block` advanced transactionally with the inserts.
- **Reorg awareness (day 1) vs reconcile (Phase 4):** store `block_hash` and `last_processed_hash` from day one so reorgs are *detectable*; full reconcile-after-N-confirmations logic is a Phase 4 increment. Base Sepolia reorgs are rare but real.

### 8.6 Local dev ergonomics
One `docker compose up` in `deploy/` brings up Postgres + gateway + indexer + a local chain (anvil). Without this the friction is ops friction, not microservices learning.

---

## 9. Algorithmic complexity (data paths)

- **Phase 2 owner list (`getLogs` by indexed `recipient`):** `O(k)` in the number of the user's certificates; per-token detail fetched with viem `multicall` → one RPC round-trip, **no N+1**.
- **Phase 3 `ListCertificates`:** indexed lookup on `owner_address` (`O(log n)` + page size), cursor-paginated — bounded, no unbounded array return.
- **Dropped on-chain owner array** removes the one unbounded-gas read path the original design carried.
- Space: read model is `O(n)` in total certificates (one row each), which is the point of a projection.

*(Penjelasan ke user disampaikan dalam Bahasa Indonesia, empat-beat, saat implementasi.)*

---

## 10. Code quality gates

### Go (Phase 3) — Sonar-Go + golangci, written-compliant on first commit
- `go:S107` ≤7 params (prefer ≤5; use a Deps/Opts struct beyond that).
- `go:S3776` cognitive complexity ≤15 (extract helpers; `t.Run` subtests).
- `go:S1192` const for any string literal duplicated 3+ times.
- Plus `errcheck`, `gosec`, `govulncheck`. Hexagonal layering per service (domain imports nothing from adapter/platform). Invoke `golang-expert` (hub) + `golang-grpc` for the gRPC work.
- Verification: `gofmt → go vet → golangci-lint → go test -race -cover (≥80%) → govulncheck → Sonar`.

### Solidity (Phase 1)
- `forge fmt`, `forge test` (full coverage of §6.6), optional `slither`. No compiler warnings.

### Frontend (Phase 2)
- TypeScript `strict`; Sonar-TS rules (readonly props, no nested ternaries, real elements over ARIA roles, `globalThis` not `window`). `react-doctor` finishing pass (lint/a11y/bundle). Tailwind v4 syntax only.

### TDD verdicts (§16, per area)
- **Contract — TDD: yes.** Clear input→output contracts (access control, soulbound reverts, getter round-trip). Foundry red test before implementation.
- **Indexer core — TDD: yes.** Event-decode → row mapping, idempotency, resume-from-block, reorg detection are checkable contracts.
- **Gateway — TDD: yes for validation/mapping logic** (input shaping, cursor decode); **no for thin REST→gRPC wiring** (add tests after + normal-path regression). SIWE deferred to Phase 4, so no auth tests in v0.1.
- **Frontend — TDD: no.** Visual/wiring; verify by running + screenshots + `react-doctor`; add tests after for hook logic (address validation, network guard).

---

## 11. Definition of Done

**Phase 1:** contract deployed to Base Sepolia, verified on Basescan; full test suite green; ABI + address exported.

**Phase 2:** frontend connects wallet; admin issues a certificate (signs in browser); user sees their certificates; public verify page works logged-out and proves against chain; certificate is non-transferable; wrong-network handled; privacy warning present before issuing; tx hash + explorer link shown.

**Phase 3:** indexer ingests events idempotently and resumably into Postgres; gateway serves paginated/searchable list over gRPC (and never touches Postgres); live "certificate issued" stream reaches the UI; `docker compose up` runs the whole stack locally; health/readiness wired.

**Overall:** README with run / deploy / demo instructions + screenshots.

---

## 12. Roadmap (post-v0.1)
v0.2 IPFS metadata upload · v0.3 issuer/org profile · v0.4 certificate PDF generator · v0.5 email invite recipient · v0.6 analytics dashboard · v0.7 multi-issuer (AccessControl) · v1.0 production-ready.

---

## 13. Decision log (resolved during brainstorming)
- **tokenURI:** minimal — store optional `metadataURI`, `tokenURI()` returns it or `""`. On-chain JSON/SVG deferred. *(Renders blank in external wallets; the app's own pages display fields via `getCertificate`.)*
- **Frontend framework:** Vite SPA (pure client). TanStack Start SSR rejected — its only justification (server-side OG/SEO for the public page) requires a server doing RPC reads, which contradicts the "no backend for the dApp" line and is a Phase-4-or-later concern.
- **Admin model:** `Ownable` single-owner (deployer). AccessControl/roles deferred to v0.7.
- **Connectors:** injected + Coinbase Wallet. WalletConnect deferred (projectId dependency).
- **On-chain owner array:** dropped. Enumeration off-chain via events/indexer.
- **Backend:** none for Phases 1–2; indexer + gateway (Phase 3) for the microservices learning goal. Backend never holds keys / never mints.
- **Redis:** not in v0.1. gRPC streaming covers realtime; Redis enters with horizontal scale or a job queue (Phase 4).
- **Build order:** incremental (contract → dApp → backend → increments).
- **Service decomposition axis:** by **technical concern** (gateway/BFF vs indexer/data), NOT by business domain. SkillPass has a single core domain (certificates); splitting it into `certificate-service` / `user-service` / etc. now would be fake bounded contexts = distributed monolith. The `orderservice`/`productservice` pattern is domain decomposition, which applies when multiple real domains exist. Additional domain-services (`issuer-service`, `notification-service`) are added in Phase 4 when those domains genuinely exist — same shape (gateway in front, N domain-services behind), just grown honestly.

---

## Appendix — cross-model review (Codex / GPT-5.5)
Two adversarial cross-model passes (per global rule §19/§20) stress-tested (a) the contract + frontend design and (b) the microservices topology. Key adopted refinements: previous-owner soulbound check (not `auth`); revert `approve`/`setApprovalForAll`; drop the on-chain owner array; indexer owns Postgres exclusively; use-case-shaped (not table-shaped) gRPC; build the streaming endpoint so gRPC is not ceremony; hybrid reads (list via gateway, verify via chain); idempotency + resumable + reorg-aware indexing; one-command local dev. Full transcripts were rendered in-chat during brainstorming.
