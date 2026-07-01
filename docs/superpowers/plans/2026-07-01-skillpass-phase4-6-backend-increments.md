# SkillPass Phase 4–6 — Backend Increments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Every Go task: **invoke `golang-expert` first (hub — auto-chains go-patterns / go-review / go-test / go-error-handling / go-concurrency-patterns + algorithmic-complexity + senior-backend + senior-security)** and carry the Sonar-Go guardrails block below verbatim.

**Goal:** Close three backend gaps deferred from BE-1/BE-2 — reorg reconcile (Phase 4), a certificate-issuance time-series trend endpoint (Phase 5), and a Redis+asynq cache/job layer for that trend endpoint (Phase 6) — on the existing `services/indexer` + `services/gateway` Go module.

**Architecture:** Phase 4 is indexer-internal (worker + repo + chain adapter). Phase 5 adds one new gRPC method + REST wrapper, following the exact `GetCertificate`/`ListCertificates` pattern already in the codebase. Phase 6 introduces Redis + asynq **behind the indexer only** — the gateway's contract is unchanged from Phase 5.

**Tech Stack:** Go 1.26 (existing module), pgx/v5 + sqlc + goose (existing), `github.com/redis/go-redis/v9` (new, Phase 6), `github.com/hibiken/asynq` (new, Phase 6). No frontend work in this plan — backend-first per the approved spec; a follow-up FE plan wires the trend endpoint into the UI only if/when needed.

**Spec:** `docs/superpowers/specs/2026-07-01-skillpass-phase4-6-backend-increments-design.md` — read it in full before starting; this plan implements it exactly, including the cross-model-debate corrections already folded into the spec (canonical-hash checkpoint fix, cert-row-independent reorg rewind, cache-behind-indexer boundary).

## Global Constraints

- **Single Go module** `github.com/oksasatya/skillpass` at repo root — all new code lives under `services/indexer/` or `services/gateway/`, no new modules.
- **Hexagonal-lite (HARD):** `domain` and `usecase` packages import nothing from `adapter`, `platform`, gRPC, Redis, or asynq. Adapters (`adapter/chain`, `adapter/postgres`, `adapter/grpc`, `adapter/cache`, `adapter/asynqjobs`) implement `usecase`-defined port interfaces.
- **Gateway boundary (HARD):** the gateway's Go code gains **zero** new imports of Redis/asynq/Postgres in this plan. It only ever calls the indexer's gRPC `CertificateQuery` service — Phase 6's cache lives entirely inside the indexer.
- **Reorg confirmation depth:** 12 blocks (fixed, not configurable — matches the approved spec).
- **Sonar-Go from first commit** (paste into every task):

```
# Sonar-Go guardrails — write compliant from the first commit
- go:S107 — ≤7 params (≤5 preferred; past that a Deps/Opts struct).
- go:S3776 — cognitive complexity ≤15 → extract helpers; t.Run subtests.
- go:S1192 — const for any string literal duplicated 3+ times.
- errcheck (handle every error), gosec, govulncheck. Wrap with %w; sentinel errors + errors.Is/As.
```

- **TDD verdicts are stated per task below** — honor them exactly; a task with `TDD: yes` writes the failing test FIRST.

---

## Phase 4 — Reorg reconcile

### Task 1: `EventSource.BlockHash` + `CertificateRepo.DeleteFromBlock` ports and adapters

**TDD: yes for `DeleteFromBlock`** (real DB deletion boundary — correctness-critical, easy to get the `>=` vs `>` boundary wrong). **TDD: no for `BlockHash`** (thin RPC wrapper, mirrors the existing untested `HeadBlock`/`IssuedLogs` methods on the same adapter).

**Files:**
- Modify: `services/indexer/internal/usecase/ports.go`
- Modify: `services/indexer/internal/adapter/chain/eventsource.go`
- Modify: `services/indexer/internal/db/queries.sql`
- Modify: `services/indexer/internal/adapter/postgres/certificate_repo.go`
- Test: `services/indexer/internal/adapter/postgres/certificate_repo_test.go`

**Interfaces:**
- Produces: `usecase.EventSource.BlockHash(ctx context.Context, blockNumber uint64) (string, error)`; `usecase.CertificateRepo.DeleteFromBlock(ctx context.Context, chainID int64, blockNumber uint64) error`.

- [ ] **Step 1: Add both port methods**

Edit `services/indexer/internal/usecase/ports.go` — add to the `EventSource` interface:

```go
	// BlockHash returns the canonical header hash of the given block number.
	BlockHash(ctx context.Context, blockNumber uint64) (string, error)
```

And add to the `CertificateRepo` interface:

```go
	// DeleteFromBlock removes all certificates at or above blockNumber for chainID —
	// used by reorg reconcile to roll back the confirmation window.
	DeleteFromBlock(ctx context.Context, chainID int64, blockNumber uint64) error
```

- [ ] **Step 2: Implement `BlockHash` on the chain adapter**

Edit `services/indexer/internal/adapter/chain/eventsource.go` — add after `HeadBlock`:

```go
// BlockHash returns the canonical header hash of the given block number.
func (e *EventSource) BlockHash(ctx context.Context, blockNumber uint64) (string, error) {
	header, err := e.client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return "", fmt.Errorf("header by number %d: %w", blockNumber, err)
	}
	return header.Hash().Hex(), nil
}
```

- [ ] **Step 3: Add the delete query**

Append to `services/indexer/internal/db/queries.sql`:

```sql
-- DeleteCertificatesFromBlock removes all certificates at or above the given block number
-- (chain-scoped) — used by reorg reconcile to roll back the confirmation window.
-- name: DeleteCertificatesFromBlock :exec
DELETE FROM certificates WHERE chain_id = $1 AND block_number >= $2;
```

- [ ] **Step 4: Regenerate sqlc**

Run: `cd services/indexer && sqlc generate`
Expected: `internal/db/queries.sql.go` gains `DeleteCertificatesFromBlock(ctx context.Context, arg DeleteCertificatesFromBlockParams) error` and `DeleteCertificatesFromBlockParams{ChainID int64; BlockNumber int64}`.

- [ ] **Step 5: Write the failing test**

Add to `services/indexer/internal/adapter/postgres/certificate_repo_test.go`:

```go
func TestDeleteFromBlock(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	below := makeCert("1")
	below.ChainID = 1
	below.BlockNumber = 100
	atBoundary := makeCertOwner("2", "0xabcdef1234567890abcdef1234567890abcdef12")
	atBoundary.ChainID = 1
	atBoundary.BlockNumber = 150
	above := makeCertOwner("3", "0xabcdef1234567890abcdef1234567890abcdef12")
	above.ChainID = 1
	above.BlockNumber = 200

	for _, c := range []domain.Certificate{below, atBoundary, above} {
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert %s: %v", c.TokenID, err)
		}
	}

	if err := repo.DeleteFromBlock(ctx, 1, 150); err != nil {
		t.Fatalf("DeleteFromBlock: %v", err)
	}

	if _, err := repo.GetByTokenID(ctx, "1"); err != nil {
		t.Fatalf("token 1 (below boundary) should survive: %v", err)
	}
	if _, err := repo.GetByTokenID(ctx, "2"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("token 2 (at boundary, inclusive) should be deleted, got err=%v", err)
	}
	if _, err := repo.GetByTokenID(ctx, "3"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("token 3 (above boundary) should be deleted, got err=%v", err)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `cd services/indexer && go test ./internal/adapter/postgres/... -run TestDeleteFromBlock -v`
Expected: FAIL — `repo.DeleteFromBlock undefined` (compile error, since `CertificateRepo` doesn't implement it yet).

- [ ] **Step 7: Implement `DeleteFromBlock` on the Postgres repo**

Edit `services/indexer/internal/adapter/postgres/certificate_repo.go` — add after `SaveState`:

```go
// DeleteFromBlock removes all certificates at or above blockNumber for chainID —
// O(k) rows deleted, where k is the number of certificates in the reorg window
// (bounded by the 12-block confirmation depth, small at any realistic scale).
func (r *CertificateRepo) DeleteFromBlock(ctx context.Context, chainID int64, blockNumber uint64) error {
	err := r.queries.DeleteCertificatesFromBlock(ctx, db.DeleteCertificatesFromBlockParams{
		ChainID:     chainID,
		BlockNumber: int64(blockNumber), //nolint:gosec // blockNumber realistically << max int64
	})
	if err != nil {
		return fmt.Errorf("postgres.CertificateRepo.DeleteFromBlock: %w", err)
	}
	return nil
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `cd services/indexer && go test ./internal/adapter/postgres/... -run TestDeleteFromBlock -v`
Expected: PASS

- [ ] **Step 9: Full build + vet + commit**

Run: `go build ./... && go vet ./... && gofmt -l services/indexer`
Expected: clean (no output from gofmt, no errors)

```bash
git add services/indexer/internal/usecase/ports.go services/indexer/internal/adapter/chain/eventsource.go services/indexer/internal/db/queries.sql services/indexer/internal/db/queries.sql.go services/indexer/internal/adapter/postgres/certificate_repo.go services/indexer/internal/adapter/postgres/certificate_repo_test.go
git commit -m "feat(indexer): BlockHash + DeleteFromBlock ports for reorg reconcile"
```

---

### Task 2: Worker reconcile logic + canonical-checkpoint-hash fix

**TDD: yes** — this is exactly the bug-prone, reproducible-failure class of logic the project's TDD gate targets (a reorg is a state-machine transition with a clear input→output contract).

**Files:**
- Modify: `services/indexer/internal/usecase/worker.go`
- Modify: `services/indexer/internal/usecase/worker_test.go`

**Interfaces:**
- Consumes: `EventSource.BlockHash` and `CertificateRepo.DeleteFromBlock` from Task 1.
- Produces: `Worker.poll` now self-heals on a detected reorg; no new exported API.

- [ ] **Step 1: Extend the test fakes**

Edit `services/indexer/internal/usecase/worker_test.go` — add a `blockHashes` field to `fakeEventSource` and a `BlockHash` method, and a `DeleteFromBlock` method on `fakeRepo`:

```go
// add to the fakeEventSource struct:
	blockHashes map[uint64]string // canonical hash override per block; deterministic default if unset

// add method:
func (f *fakeEventSource) BlockHash(_ context.Context, blockNumber uint64) (string, error) {
	if h, ok := f.blockHashes[blockNumber]; ok {
		return h, nil
	}
	return fmt.Sprintf("0xcanonical%d", blockNumber), nil
}

// add to fakeRepo:
func (r *fakeRepo) DeleteFromBlock(_ context.Context, _ int64, blockNumber uint64) error {
	for tokenID, c := range r.certs {
		if uint64(c.BlockNumber) >= blockNumber {
			delete(r.certs, tokenID)
		}
	}
	return nil
}
```

- [ ] **Step 2: Write the failing tests**

Append to `services/indexer/internal/usecase/worker_test.go`:

```go
func TestWorker_Reconcile_NoReorg_IsNoop(t *testing.T) {
	repo := newFakeRepo()
	repo.state = domain.IndexerState{ChainID: 31337, LastProcessedBlock: 10, LastProcessedHash: "0xcanonical10"}
	repo.certs["1"] = domain.Certificate{TokenID: "1", BlockNumber: 5}

	src := &fakeEventSource{head: 10} // BlockHash(10) defaults to "0xcanonical10" — matches stored
	w := newWorker(src, repo)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if _, ok := repo.certs["1"]; !ok {
		t.Fatal("no reorg should have occurred — cert must survive")
	}
}

func TestWorker_Reconcile_DetectsReorgAndRewinds(t *testing.T) {
	repo := newFakeRepo()
	repo.state = domain.IndexerState{ChainID: 31337, LastProcessedBlock: 20, LastProcessedHash: "0xstale-hash"}
	repo.certs["1"] = domain.Certificate{TokenID: "1", BlockNumber: 5}  // below the rewind window — survives
	repo.certs["2"] = domain.Certificate{TokenID: "2", BlockNumber: 15} // within [20-12+1, 20] — deleted

	src := &fakeEventSource{
		head: 20,
		blockHashes: map[uint64]string{
			20: "0xcanonical20", // mismatches stored "0xstale-hash" -> reorg detected
			8:  "0xcanonical8",  // rewindTo = 20-12 = 8
		},
	}
	w := newWorker(src, repo)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}

	if _, ok := repo.certs["1"]; !ok {
		t.Fatal("token 1 (block 5, below rewind point) must survive")
	}
	if _, ok := repo.certs["2"]; ok {
		t.Fatal("token 2 (block 15, within rewound window) must be deleted")
	}
	if repo.state.LastProcessedBlock != 8 {
		t.Fatalf("state.LastProcessedBlock = %d, want 8 (rewound)", repo.state.LastProcessedBlock)
	}
	if repo.state.LastProcessedHash != "0xcanonical8" {
		t.Fatalf("state.LastProcessedHash = %q, want canonical hash of the rewound block", repo.state.LastProcessedHash)
	}
}

func TestWorker_Reconcile_ColdStart_IsNoop(t *testing.T) {
	repo := newFakeRepo() // zero-value state: LastProcessedHash == ""
	src := &fakeEventSource{head: 0}
	w := newWorker(src, repo)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	// must not panic or attempt a delete on an uninitialized checkpoint
}

func TestWorker_ChecksInCanonicalHash_NotLastLogHash(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 5,
		logs: map[uint64][]domain.IssuedLog{
			1: {sampleLog("1", 1)}, // this log's own BlockHash field is "0xaabbcc" (see sampleLog)
		},
		certs:       map[string]domain.OnchainCertificate{"1": sampleCert("1")},
		blockHashes: map[uint64]string{5: "0xcanonical-head-5"},
	}
	w := newWorker(src, repo)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if repo.state.LastProcessedHash != "0xcanonical-head-5" {
		t.Fatalf("state.LastProcessedHash = %q, want the canonical head hash, not the log's own block hash (0xaabbcc)", repo.state.LastProcessedHash)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/indexer && go test ./internal/usecase/... -run TestWorker_Reconcile -v`
Expected: FAIL — `reconcile` doesn't exist yet, and the checkpoint hash still comes from the last log.

- [ ] **Step 4: Implement `reconcile` and fix the checkpoint hash**

Edit `services/indexer/internal/usecase/worker.go`. Add the constant near the top:

```go
// reorgWindow is the confirmation depth: a reorg is only ever reconciled within this many
// blocks of the checkpoint. Fixed, not configurable — matches the project's chosen finality
// assumption (see the Phase 4 design spec).
const reorgWindow = 12
```

Replace the body of `poll` (keep the signature) so it calls `reconcile` first and fixes the checkpoint hash at the end:

```go
// poll fetches one batch of blocks, processes each log, and advances state.
// A batch failure returns an error WITHOUT advancing state — crash-safe re-process.
func (w *Worker) poll(ctx context.Context) error {
	if err := w.reconcile(ctx); err != nil {
		return fmt.Errorf("reconcile: %w", err)
	}

	head, err := w.src.HeadBlock(ctx)
	if err != nil {
		return fmt.Errorf("head block: %w", err)
	}
	if w.next > head {
		return nil // nothing to do
	}

	to := min(w.next+w.cfg.BatchSize-1, head)
	logs, err := w.src.IssuedLogs(ctx, w.next, to)
	if err != nil {
		return fmt.Errorf("issued logs [%d,%d]: %w", w.next, to, err)
	}

	// ponytail: sequential N eth_calls per batch; bounded-parallel with errgroup+SetLimit if volume grows
	for _, l := range logs {
		if err := w.processLog(ctx, l); err != nil {
			return err // state not advanced; batch re-tried next poll
		}
	}

	// Canonical hash of `to`, NOT the last log's hash — this is the Phase 4 checkpoint fix.
	// A block range with no logs still needs a trustworthy checkpoint hash for the next
	// reconcile check to compare against.
	canonicalHash, err := w.src.BlockHash(ctx, to)
	if err != nil {
		return fmt.Errorf("block hash %d: %w", to, err)
	}

	newState := domain.IndexerState{
		ChainID:            w.cfg.ChainID,
		LastProcessedBlock: to,
		LastProcessedHash:  canonicalHash,
	}
	if err := w.repo.SaveState(ctx, newState); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	w.next = to + 1
	return nil
}

// reconcile detects a reorg at the last-processed checkpoint and, if found, rewinds
// last_processed_block by the full confirmation window and deletes indexed certificates
// above the rewound point. The normal poll flow above then naturally re-ingests them —
// Upsert is idempotent, so re-processing blocks that didn't actually reorg is harmless.
//
// O(1) extra work in the common (no-reorg) case: one CertificateRepo.GetState call and one
// EventSource.BlockHash call per poll cycle, not a rescan of the whole reorg window.
func (w *Worker) reconcile(ctx context.Context) error {
	state, err := w.repo.GetState(ctx)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}
	if state.LastProcessedHash == "" {
		return nil // cold start — nothing indexed yet, nothing to reconcile
	}

	canonicalHash, err := w.src.BlockHash(ctx, state.LastProcessedBlock)
	if err != nil {
		return fmt.Errorf("block hash %d: %w", state.LastProcessedBlock, err)
	}
	if canonicalHash == state.LastProcessedHash {
		return nil // no reorg
	}

	w.log.Warn("reorg detected",
		"last_processed_block", state.LastProcessedBlock,
		"stored_hash", state.LastProcessedHash,
		"canonical_hash", canonicalHash,
	)

	rewindTo := w.cfg.StartBlock
	if state.LastProcessedBlock > w.cfg.StartBlock+reorgWindow {
		rewindTo = state.LastProcessedBlock - reorgWindow
	}

	if err := w.repo.DeleteFromBlock(ctx, w.cfg.ChainID, rewindTo+1); err != nil {
		return fmt.Errorf("delete from block %d: %w", rewindTo+1, err)
	}

	rewoundHash, err := w.src.BlockHash(ctx, rewindTo)
	if err != nil {
		return fmt.Errorf("block hash %d: %w", rewindTo, err)
	}
	if err := w.repo.SaveState(ctx, domain.IndexerState{
		ChainID:            w.cfg.ChainID,
		LastProcessedBlock: rewindTo,
		LastProcessedHash:  rewoundHash,
	}); err != nil {
		return fmt.Errorf("save rewound state: %w", err)
	}
	w.next = rewindTo + 1
	return nil
}
```

Also remove the now-unused `lastHash` local variable and its comment block from the old `poll` body (they're superseded by the canonical-hash fetch above) — the "ponytail: LastProcessedHash = the last log's block hash..." comment is deleted since the gap it documented is exactly what this task fixes.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/indexer && go test ./internal/usecase/... -v`
Expected: PASS — all Phase-4 tests above AND every pre-existing `TestWorker_*` test (ColdStart, Resume, IdempotentReplay, EmptyRange, GetCertificateError, CtxCancel) still green, since the fake's default `BlockHash` return is a deterministic pure function of the block number, so no pre-existing test ever triggers a false reorg.

- [ ] **Step 6: Race + full verification**

Run: `cd services/indexer && go build ./... && go vet ./... && gofmt -l internal/usecase && go test ./... -race -cover`
Expected: clean build, no gofmt output, all packages PASS.

- [ ] **Step 7: Commit**

```bash
git add services/indexer/internal/usecase/worker.go services/indexer/internal/usecase/worker_test.go
git commit -m "feat(indexer): reorg reconcile — canonical checkpoint hash + bounded 12-block rewind"
```

**Anti-patterns to avoid:** don't try to find the exact reorg divergence point by scanning certificate rows — a reorg on an empty block range is invisible that way (this is the Codex-debate correction baked into the spec). Don't skip the checkpoint-hash fix and reuse the last log's hash — reconcile silently never detects anything if you do.

---

## Phase 5 — Certificate issuance trend (data metrics)

### Task 3: Migration — index on `issued_at`

**TDD: no** (migration — infra, not logic).

**Files:**
- Create: `services/indexer/migrations/00002_issued_at_index.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
CREATE INDEX idx_certificates_issued_at ON certificates (issued_at);

-- +goose Down
DROP INDEX idx_certificates_issued_at;
```

- [ ] **Step 2: Verify it applies**

Run: `make migrate-test` (spins a throwaway Postgres, applies migrations, tears down)
Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add services/indexer/migrations/00002_issued_at_index.sql
git commit -m "feat(indexer): add issued_at index for the Phase 5 trend query"
```

---

### Task 4: Proto — `GetIssuanceTrend` RPC + messages

**TDD: no** (codegen).

**Files:**
- Modify: `proto/skillpass/cert/v1/certificate.proto`

- [ ] **Step 1: Add the enum, messages, and RPC**

Edit `proto/skillpass/cert/v1/certificate.proto` — add to the `CertificateQuery` service:

```protobuf
  rpc GetIssuanceTrend(GetIssuanceTrendRequest) returns (GetIssuanceTrendResponse);
```

Append after `GetIndexerStatusResponse`:

```protobuf
enum TrendBucket {
  TREND_BUCKET_UNSPECIFIED = 0;
  TREND_BUCKET_DAY = 1;
  TREND_BUCKET_WEEK = 2;
  TREND_BUCKET_MONTH = 3;
}

message GetIssuanceTrendRequest {
  TrendBucket bucket = 1;
  string range_preset = 2; // e.g. "30d" for DAY, "12w" for WEEK, "12m" for MONTH
}

message TrendPoint {
  google.protobuf.Timestamp bucket_start = 1; // UTC
  uint64 count = 2;
}

message GetIssuanceTrendResponse {
  repeated TrendPoint points = 1;
}
```

- [ ] **Step 2: Regenerate**

Run: `make proto` (equivalently `buf generate`)
Expected: `proto/gen/go/skillpass/cert/v1/certificate.pb.go` and `certificate_grpc.pb.go` regenerate with `TrendBucket`, `GetIssuanceTrendRequest`, `TrendPoint`, `GetIssuanceTrendResponse`, and `CertificateQueryClient.GetIssuanceTrend` / `CertificateQueryServer.GetIssuanceTrend`.

- [ ] **Step 3: Verify it compiles**

Run: `go build ./proto/...`
Expected: clean (note: `CertificateQueryServer` implementations — `adapter/grpc.Server` — will NOT yet satisfy the interface until Task 7; that's expected and fixed there. If your Go toolchain compiles `proto/...` in isolation this step passes; the full-module build is re-verified at the end of Task 7.)

- [ ] **Step 4: Commit**

```bash
git add proto/skillpass/cert/v1/certificate.proto proto/gen/go/skillpass/cert/v1/certificate.pb.go proto/gen/go/skillpass/cert/v1/certificate_grpc.pb.go
git commit -m "feat(proto): GetIssuanceTrend RPC — bucketed certificate-issuance time series"
```

---

### Task 5: `usecase` trend types — bucket alignment, zero-fill, range presets, `TrendService`

**TDD: yes** — pure, input→output logic; exactly the case TDD is for.

**Files:**
- Create: `services/indexer/internal/usecase/trend.go`
- Create: `services/indexer/internal/usecase/trend_test.go`
- Modify: `services/indexer/internal/usecase/ports.go` (add `CertificateRepo.GetIssuanceTrend`, add `TrendCache` interface)

**Interfaces:**
- Produces: `usecase.TrendBucket` (`TrendBucketDay`/`Week`/`Month`), `usecase.TrendPoint{BucketStart time.Time; Count int64}`, `usecase.RangePresetToSince(bucket, preset string, now time.Time) (time.Time, error)`, `usecase.AllowedPresets() map[TrendBucket]map[string]int`, `usecase.AlignedBuckets(bucket, since, now time.Time) []time.Time`, `usecase.ZeroFillTrend(expected []time.Time, rows []TrendPoint) []TrendPoint`, `usecase.NewTrendService(repo CertificateRepo, chainID int64) *TrendService` with methods `GetTrend(ctx, bucket, since time.Time, preset string) ([]TrendPoint, error)` and `RefreshCache(ctx, bucket, since time.Time, preset string) ([]TrendPoint, error)` and `SetCache(cache TrendCache)`.
- Consumes: nothing new from other tasks yet (Task 6 provides the real `CertificateRepo.GetIssuanceTrend`; this task's tests use a fake).

- [ ] **Step 1: Add the port additions**

Edit `services/indexer/internal/usecase/ports.go` — add to `CertificateRepo`:

```go
	// GetIssuanceTrend returns raw (non-zero-filled) trend rows since the given time,
	// bucketed by day/week/month. O(certs in range) via the issued_at index.
	GetIssuanceTrend(ctx context.Context, bucket TrendBucket, since time.Time) ([]TrendPoint, error)
```

Add near the bottom of the file (new top-level interface):

```go
// TrendCache lets TrendService cache computed trend results (Phase 6 wires a Redis-backed
// implementation; TrendService is nil-safe if none is set).
type TrendCache interface {
	Get(ctx context.Context, key string) ([]TrendPoint, bool, error)
	Set(ctx context.Context, key string, points []TrendPoint) error
}
```

Add `"time"` to the file's import block if not already present.

- [ ] **Step 2: Write the failing tests**

Create `services/indexer/internal/usecase/trend_test.go`:

```go
package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

func TestRangePresetToSince(t *testing.T) {
	now := time.Date(2026, 7, 1, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		bucket  usecase.TrendBucket
		preset  string
		wantErr bool
	}{
		{"day 30d valid", usecase.TrendBucketDay, "30d", false},
		{"week 12w valid", usecase.TrendBucketWeek, "12w", false},
		{"month 12m valid", usecase.TrendBucketMonth, "12m", false},
		{"day preset invalid for week bucket", usecase.TrendBucketWeek, "30d", true},
		{"unknown bucket", usecase.TrendBucket(99), "30d", true},
		{"unknown preset", usecase.TrendBucketDay, "7d", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			since, err := usecase.RangePresetToSince(tt.bucket, tt.preset, now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !since.Before(now) {
				t.Fatalf("since (%v) should be before now (%v)", since, now)
			}
		})
	}
}

func TestAlignedBuckets_Day(t *testing.T) {
	since := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	got := usecase.AlignedBuckets(usecase.TrendBucketDay, since, now)

	want := []time.Time{
		time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	if len(got) != len(want) {
		t.Fatalf("got %d buckets, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("bucket[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestAlignedBuckets_Month(t *testing.T) {
	since := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	got := usecase.AlignedBuckets(usecase.TrendBucketMonth, since, now)

	want := []time.Time{
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	if len(got) != len(want) {
		t.Fatalf("got %d buckets, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("bucket[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestZeroFillTrend(t *testing.T) {
	d1 := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	expected := []time.Time{d1, d2, d3}

	// d2 has no rows — must be zero-filled
	rows := []usecase.TrendPoint{
		{BucketStart: d1, Count: 3},
		{BucketStart: d3, Count: 1},
	}

	got := usecase.ZeroFillTrend(expected, rows)

	if len(got) != 3 {
		t.Fatalf("got %d points, want 3", len(got))
	}
	if got[0].Count != 3 || got[1].Count != 0 || got[2].Count != 1 {
		t.Fatalf("counts = [%d,%d,%d], want [3,0,1]", got[0].Count, got[1].Count, got[2].Count)
	}
}

// fakeTrendRepo implements the subset of usecase.CertificateRepo TrendService needs.
type fakeTrendRepo struct {
	usecase.CertificateRepo // embed nil; only GetIssuanceTrend is exercised by these tests
	rows                    []usecase.TrendPoint
	err                     error
}

func (f *fakeTrendRepo) GetIssuanceTrend(_ context.Context, _ usecase.TrendBucket, _ time.Time) ([]usecase.TrendPoint, error) {
	return f.rows, f.err
}

func TestTrendService_GetTrend_ComputesAndZeroFills(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeTrendRepo{rows: []usecase.TrendPoint{{BucketStart: now, Count: 5}}}
	svc := usecase.NewTrendService(repo, 31337)

	since, _ := usecase.RangePresetToSince(usecase.TrendBucketDay, "30d", now)
	points, err := svc.GetTrend(context.Background(), usecase.TrendBucketDay, since, "30d")
	if err != nil {
		t.Fatalf("GetTrend: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected at least one zero-filled bucket")
	}
}

// fakeTrendCache implements usecase.TrendCache in-memory for tests.
type fakeTrendCache struct {
	store map[string][]usecase.TrendPoint
}

func newFakeTrendCache() *fakeTrendCache { return &fakeTrendCache{store: map[string][]usecase.TrendPoint{}} }

func (c *fakeTrendCache) Get(_ context.Context, key string) ([]usecase.TrendPoint, bool, error) {
	v, ok := c.store[key]
	return v, ok, nil
}

func (c *fakeTrendCache) Set(_ context.Context, key string, points []usecase.TrendPoint) error {
	c.store[key] = points
	return nil
}

func TestTrendService_GetTrend_CacheHit_SkipsRepo(t *testing.T) {
	repoCalled := false
	repo := &fakeTrendRepo{}
	svc := usecase.NewTrendService(repo, 31337)
	cache := newFakeTrendCache()
	svc.SetCache(cache)

	since, _ := usecase.RangePresetToSince(usecase.TrendBucketDay, "30d", time.Now())
	// prime the cache directly via RefreshCache, then verify GetTrend reads it back without
	// requiring the repo to be called again.
	if _, err := svc.RefreshCache(context.Background(), usecase.TrendBucketDay, since, "30d"); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	repo.rows = nil // if GetTrend hits the repo now, it would return an empty (not cached) result
	points, err := svc.GetTrend(context.Background(), usecase.TrendBucketDay, since, "30d")
	if err != nil {
		t.Fatalf("GetTrend: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected cached (zero-filled, non-empty) points, cache appears to have been bypassed")
	}
	_ = repoCalled
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/indexer && go test ./internal/usecase/... -run 'TestRangePresetToSince|TestAlignedBuckets|TestZeroFillTrend|TestTrendService' -v`
Expected: FAIL — compile error, none of these types/functions exist yet.

- [ ] **Step 4: Implement `trend.go`**

Create `services/indexer/internal/usecase/trend.go`:

```go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TrendBucket is the aggregation granularity for GetIssuanceTrend.
type TrendBucket int

const (
	TrendBucketDay TrendBucket = iota + 1
	TrendBucketWeek
	TrendBucketMonth
)

// TrendPoint is one bucketed count in a certificate-issuance trend.
type TrendPoint struct {
	BucketStart time.Time
	Count       int64
}

// ErrInvalidTrendRequest is returned for an unknown bucket or a preset not supported for it.
var ErrInvalidTrendRequest = errors.New("invalid trend request")

// allowedPresets maps each bucket to its supported range presets and the number of days
// each preset covers. Bounded on purpose (see the Phase 5 design spec): an unbounded
// client-supplied range can't be validated cheaply or cached in Phase 6.
var allowedPresets = map[TrendBucket]map[string]int{
	TrendBucketDay:   {"30d": 30, "90d": 90, "365d": 365},
	TrendBucketWeek:  {"12w": 12 * 7, "52w": 52 * 7},
	TrendBucketMonth: {"12m": 12 * 31, "24m": 24 * 31}, // approximate; bucket boundaries are exact via date_trunc
}

// AllowedPresets returns the full bucket -> preset -> days table (used by the Phase 6 refresh
// job to enumerate every combination to precompute).
func AllowedPresets() map[TrendBucket]map[string]int {
	return allowedPresets
}

// RangePresetToSince validates preset against bucket and returns the UTC "since" timestamp
// (now minus the preset's day count). now is injected for testability.
func RangePresetToSince(bucket TrendBucket, preset string, now time.Time) (time.Time, error) {
	presets, ok := allowedPresets[bucket]
	if !ok {
		return time.Time{}, fmt.Errorf("%w: unknown bucket %d", ErrInvalidTrendRequest, bucket)
	}
	days, ok := presets[preset]
	if !ok {
		return time.Time{}, fmt.Errorf("%w: preset %q not supported for this bucket", ErrInvalidTrendRequest, preset)
	}
	return now.UTC().AddDate(0, 0, -days), nil
}

// AlignedBuckets returns every expected bucket-start timestamp from since to now (inclusive),
// aligned to UTC day/week(Monday)/month boundaries. O(number of buckets in range) — bounded
// by the preset table above (at most 365 for the day bucket).
func AlignedBuckets(bucket TrendBucket, since, now time.Time) []time.Time {
	cur := truncateToBucket(bucket, since)
	end := truncateToBucket(bucket, now)

	var out []time.Time
	for !cur.After(end) {
		out = append(out, cur)
		cur = advanceBucket(bucket, cur)
	}
	return out
}

func truncateToBucket(bucket TrendBucket, t time.Time) time.Time {
	t = t.UTC()
	switch bucket {
	case TrendBucketWeek:
		offset := (int(t.Weekday()) + 6) % 7 // Monday = 0
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -offset)
	case TrendBucketMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	default: // TrendBucketDay
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
}

func advanceBucket(bucket TrendBucket, t time.Time) time.Time {
	switch bucket {
	case TrendBucketWeek:
		return t.AddDate(0, 0, 7)
	case TrendBucketMonth:
		return t.AddDate(0, 1, 0)
	default:
		return t.AddDate(0, 0, 1)
	}
}

// ZeroFillTrend merges DB rows (which only contain buckets with >=1 certificate) into the
// full aligned bucket list, filling zero-count for any missing bucket.
// O(len(expected) + len(rows)) time and O(len(rows)) space — a hash map keyed by bucket
// avoids an O(len(expected) * len(rows)) nested scan.
func ZeroFillTrend(expected []time.Time, rows []TrendPoint) []TrendPoint {
	byBucket := make(map[int64]int64, len(rows))
	for _, r := range rows {
		byBucket[r.BucketStart.Unix()] = r.Count
	}
	out := make([]TrendPoint, 0, len(expected))
	for _, t := range expected {
		out = append(out, TrendPoint{BucketStart: t, Count: byBucket[t.Unix()]})
	}
	return out
}

// TrendService orchestrates CertificateRepo (Postgres) and an optional TrendCache (Phase 6)
// to serve GetIssuanceTrend. Nil-safe on cache — a cold/absent cache just recomputes.
type TrendService struct {
	repo    CertificateRepo
	chainID int64
	cache   TrendCache // optional; nil-safe
}

// NewTrendService constructs a TrendService with no cache (Phase 5 default).
func NewTrendService(repo CertificateRepo, chainID int64) *TrendService {
	return &TrendService{repo: repo, chainID: chainID}
}

// SetCache wires an optional TrendCache. Call before serving traffic; safe to never call.
func (s *TrendService) SetCache(cache TrendCache) {
	s.cache = cache
}

// GetTrend returns the zero-filled trend for bucket/preset, since already validated by the
// caller (RangePresetToSince) — reads the cache first when one is set, computing on a miss.
func (s *TrendService) GetTrend(ctx context.Context, bucket TrendBucket, since time.Time, preset string) ([]TrendPoint, error) {
	if s.cache != nil {
		if points, ok, err := s.cache.Get(ctx, s.cacheKey(bucket, preset)); err == nil && ok {
			return points, nil
		}
	}
	return s.compute(ctx, bucket, since, preset)
}

// RefreshCache force-recomputes and writes to the cache, bypassing the cache-read path.
// Used by the Phase 6 asynq job, never by the read path.
func (s *TrendService) RefreshCache(ctx context.Context, bucket TrendBucket, since time.Time, preset string) ([]TrendPoint, error) {
	return s.compute(ctx, bucket, since, preset)
}

func (s *TrendService) compute(ctx context.Context, bucket TrendBucket, since time.Time, preset string) ([]TrendPoint, error) {
	rows, err := s.repo.GetIssuanceTrend(ctx, bucket, since)
	if err != nil {
		return nil, fmt.Errorf("get issuance trend: %w", err)
	}
	points := ZeroFillTrend(AlignedBuckets(bucket, since, time.Now()), rows)

	if s.cache != nil {
		// A cache-write failure must not fail the read — TrendService always returns correct
		// data regardless of cache health; the adapter logs its own errors if it wants to.
		_ = s.cache.Set(ctx, s.cacheKey(bucket, preset), points)
	}
	return points, nil
}

func (s *TrendService) cacheKey(bucket TrendBucket, preset string) string {
	return fmt.Sprintf("trend:v1:%d:%d:%s", s.chainID, bucket, preset)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/indexer && go test ./internal/usecase/... -v`
Expected: PASS — new trend tests AND all pre-existing worker tests.

- [ ] **Step 6: Verify + commit**

Run: `go build ./... 2>&1 | grep -v "does not implement"` — note: `postgres.CertificateRepo` and `adapter/grpc.Server` do not yet satisfy their updated interfaces; that's expected until Tasks 4/5. Confirm the ONLY build errors are exactly those two "missing method" errors (nothing else).

```bash
git add services/indexer/internal/usecase/ports.go services/indexer/internal/usecase/trend.go services/indexer/internal/usecase/trend_test.go
git commit -m "feat(indexer): TrendService — bucket alignment, zero-fill, range presets, optional cache"
```

**Anti-patterns to avoid:** don't interpolate the bucket into a raw SQL string (Task 6 dispatches via a Go switch instead) — that's an unnecessary dynamic-SQL smell for a fully enumerable value. Don't skip zero-filling — a frontend consumer should never have to gap-fill missing days itself.

---

### Task 6: SQL trend queries + `CertificateRepo.GetIssuanceTrend`

**TDD: yes** — integration test against real Postgres (testcontainers), the existing convention for this repo.

**Files:**
- Modify: `services/indexer/internal/db/queries.sql`
- Modify: `services/indexer/internal/adapter/postgres/certificate_repo.go`
- Modify: `services/indexer/internal/adapter/postgres/certificate_repo_test.go`

**Interfaces:**
- Consumes: `usecase.TrendBucket`, `usecase.TrendPoint` from Task 5.
- Produces: `(*CertificateRepo).GetIssuanceTrend(ctx, bucket usecase.TrendBucket, since time.Time) ([]usecase.TrendPoint, error)` satisfying the port.

- [ ] **Step 1: Add the three bucket queries**

Append to `services/indexer/internal/db/queries.sql`:

```sql
-- TrendByDay returns certificate counts bucketed by UTC day since the given timestamp.
-- name: TrendByDay :many
SELECT date_trunc('day', issued_at AT TIME ZONE 'UTC') AS bucket_start, count(*) AS cnt
FROM certificates
WHERE issued_at >= $1
GROUP BY 1
ORDER BY 1;

-- TrendByWeek returns certificate counts bucketed by UTC ISO week since the given timestamp.
-- name: TrendByWeek :many
SELECT date_trunc('week', issued_at AT TIME ZONE 'UTC') AS bucket_start, count(*) AS cnt
FROM certificates
WHERE issued_at >= $1
GROUP BY 1
ORDER BY 1;

-- TrendByMonth returns certificate counts bucketed by UTC month since the given timestamp.
-- name: TrendByMonth :many
SELECT date_trunc('month', issued_at AT TIME ZONE 'UTC') AS bucket_start, count(*) AS cnt
FROM certificates
WHERE issued_at >= $1
GROUP BY 1
ORDER BY 1;
```

- [ ] **Step 2: Regenerate sqlc**

Run: `cd services/indexer && sqlc generate`
Expected: `internal/db/queries.sql.go` gains `TrendByDay`/`TrendByWeek`/`TrendByMonth`, each `(ctx, issuedAt pgtype.Timestamptz) ([]TrendByXRow, error)` with `TrendByXRow{BucketStart pgtype.Timestamptz; Cnt int64}`.

- [ ] **Step 3: Write the failing test**

Add to `services/indexer/internal/adapter/postgres/certificate_repo_test.go`:

```go
func TestGetIssuanceTrend_BucketsByDay(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	day1 := makeCert("1")
	day1.IssuedAt = time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	day1b := makeCertOwner("2", "0xabcdef1234567890abcdef1234567890abcdef12")
	day1b.IssuedAt = time.Date(2026, 6, 29, 22, 0, 0, 0, time.UTC)
	day3 := makeCertOwner("3", "0xabcdef1234567890abcdef1234567890abcdef12")
	day3.IssuedAt = time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)

	for _, c := range []domain.Certificate{day1, day1b, day3} {
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert %s: %v", c.TokenID, err)
		}
	}

	since := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	points, err := repo.GetIssuanceTrend(ctx, usecase.TrendBucketDay, since)
	if err != nil {
		t.Fatalf("GetIssuanceTrend: %v", err)
	}

	// only buckets with >=1 row come back — day 06-30 has none and is absent (zero-fill is
	// TrendService's job, tested in Task 5, not the repo's).
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2 (06-29 and 07-01): %+v", len(points), points)
	}
	if points[0].Count != 2 {
		t.Errorf("06-29 count = %d, want 2", points[0].Count)
	}
	if points[1].Count != 1 {
		t.Errorf("07-01 count = %d, want 1", points[1].Count)
	}
}
```

Add `"github.com/oksasatya/skillpass/services/indexer/internal/usecase"` to the test file's import block if not already present.

- [ ] **Step 4: Run test to verify it fails**

Run: `cd services/indexer && go test ./internal/adapter/postgres/... -run TestGetIssuanceTrend -v`
Expected: FAIL — `repo.GetIssuanceTrend undefined`.

- [ ] **Step 5: Implement `GetIssuanceTrend`**

Edit `services/indexer/internal/adapter/postgres/certificate_repo.go` — add after `DeleteFromBlock`:

```go
// GetIssuanceTrend returns raw (non-zero-filled) trend rows since the given time, dispatched
// by bucket granularity — mirrors the dispatch() switch pattern used by List/Search above.
// O(certs in range) via the issued_at index, then a Postgres GROUP BY aggregate.
func (r *CertificateRepo) GetIssuanceTrend(ctx context.Context, bucket usecase.TrendBucket, since time.Time) ([]usecase.TrendPoint, error) {
	sinceArg := pgtype.Timestamptz{Time: since.UTC(), Valid: true}

	switch bucket {
	case usecase.TrendBucketDay:
		rows, err := r.queries.TrendByDay(ctx, sinceArg)
		if err != nil {
			return nil, fmt.Errorf("postgres.CertificateRepo.GetIssuanceTrend: %w", err)
		}
		return toTrendPoints(rows), nil
	case usecase.TrendBucketWeek:
		rows, err := r.queries.TrendByWeek(ctx, sinceArg)
		if err != nil {
			return nil, fmt.Errorf("postgres.CertificateRepo.GetIssuanceTrend: %w", err)
		}
		return toTrendPoints(rows), nil
	case usecase.TrendBucketMonth:
		rows, err := r.queries.TrendByMonth(ctx, sinceArg)
		if err != nil {
			return nil, fmt.Errorf("postgres.CertificateRepo.GetIssuanceTrend: %w", err)
		}
		return toTrendPoints(rows), nil
	default:
		return nil, fmt.Errorf("postgres.CertificateRepo.GetIssuanceTrend: unknown bucket %d", bucket)
	}
}

// trendRow is satisfied by every sqlc-generated TrendByXRow (structurally identical,
// distinct generated types — Go generics bridge them without repeating the mapping 3x).
type trendRow interface {
	db.TrendByDayRow | db.TrendByWeekRow | db.TrendByMonthRow
}

func toTrendPoints[R trendRow](rows []R) []usecase.TrendPoint {
	points := make([]usecase.TrendPoint, 0, len(rows))
	for _, row := range rows {
		switch v := any(row).(type) {
		case db.TrendByDayRow:
			points = append(points, usecase.TrendPoint{BucketStart: v.BucketStart.Time, Count: v.Cnt})
		case db.TrendByWeekRow:
			points = append(points, usecase.TrendPoint{BucketStart: v.BucketStart.Time, Count: v.Cnt})
		case db.TrendByMonthRow:
			points = append(points, usecase.TrendPoint{BucketStart: v.BucketStart.Time, Count: v.Cnt})
		}
	}
	return points
}
```

Add `"time"` to the file's import block if not already present.

- [ ] **Step 6: Run test to verify it passes**

Run: `cd services/indexer && go test ./internal/adapter/postgres/... -v`
Expected: PASS — new trend test AND all pre-existing repo tests.

- [ ] **Step 7: Full verify + commit**

Run: `go build ./... && go vet ./... && gofmt -l services/indexer && go test ./services/indexer/... -race -cover`
Expected: clean, all green. (The gRPC server still won't build — expected until Task 7.)

```bash
git add services/indexer/internal/db/queries.sql services/indexer/internal/db/queries.sql.go services/indexer/internal/adapter/postgres/certificate_repo.go services/indexer/internal/adapter/postgres/certificate_repo_test.go
git commit -m "feat(indexer): CertificateRepo.GetIssuanceTrend — day/week/month bucketed counts"
```

---

### Task 7: gRPC handler `GetIssuanceTrend` + wire `TrendService` into `Server`

**TDD: yes** for the bucket/preset validation mapping (mirrors the existing `errEmptyTokenID` pattern); the proto-mapping glue is thin but is exercised by the same tests.

**Files:**
- Modify: `services/indexer/internal/adapter/grpc/server.go`
- Modify: `services/indexer/internal/adapter/grpc/server_test.go`
- Modify: `services/indexer/cmd/indexer/main.go`

**Interfaces:**
- Consumes: `usecase.TrendService`, `usecase.RangePresetToSince`, `usecase.TrendBucket` from Task 5.
- Produces: `Server` gains a `trend *usecase.TrendService` field; `NewServer` signature grows to `NewServer(repo usecase.CertificateRepo, src usecase.EventSource, sub usecase.EventSubscriber, trend *usecase.TrendService, log *slog.Logger) *Server`.

- [ ] **Step 1: Extend `dialBufconn` and write the failing tests**

Edit `services/indexer/internal/adapter/grpc/server_test.go` — update `dialBufconn`'s signature and the `NewServer` call:

```go
func dialBufconn(t *testing.T, repo usecase.CertificateRepo, src usecase.EventSource, sub usecase.EventSubscriber, trend *usecase.TrendService) certv1.CertificateQueryClient {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { _ = lis.Close() })

	srv := grpc.NewServer()
	certv1.RegisterCertificateQueryServer(srv, grpcadapter.NewServer(repo, src, sub, trend, nil))
	t.Cleanup(srv.GracefulStop)
	// ... rest of the function body is unchanged
```

Update every existing `dialBufconn(t, ..., ...)` call site in this file to append a trailing `usecase.NewTrendService(&fakeRepo{}, 1)` argument (a throwaway TrendService — none of the existing GetCertificate/ListCertificates/GetIndexerStatus/StreamCertificateEvents tests exercise it).

Append new tests:

```go
func TestGetIssuanceTrend_Valid(t *testing.T) {
	repo := &fakeRepo{} // GetIssuanceTrend on fakeRepo returns (nil, nil) by default — add it below
	trend := usecase.NewTrendService(repo, 31337)
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), trend)

	resp, err := client.GetIssuanceTrend(context.Background(), &certv1.GetIssuanceTrendRequest{
		Bucket:      certv1.TrendBucket_TREND_BUCKET_DAY,
		RangePreset: "30d",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetPoints()) == 0 {
		t.Fatal("expected zero-filled points for a 30d range, got none")
	}
}

func TestGetIssuanceTrend_InvalidPresetForBucket(t *testing.T) {
	repo := &fakeRepo{}
	trend := usecase.NewTrendService(repo, 31337)
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), trend)

	_, err := client.GetIssuanceTrend(context.Background(), &certv1.GetIssuanceTrendRequest{
		Bucket:      certv1.TrendBucket_TREND_BUCKET_WEEK,
		RangePreset: "30d", // not a valid preset for WEEK
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want codes.InvalidArgument, got %v", status.Code(err))
	}
}

func TestGetIssuanceTrend_UnspecifiedBucket(t *testing.T) {
	repo := &fakeRepo{}
	trend := usecase.NewTrendService(repo, 31337)
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), trend)

	_, err := client.GetIssuanceTrend(context.Background(), &certv1.GetIssuanceTrendRequest{
		Bucket: certv1.TrendBucket_TREND_BUCKET_UNSPECIFIED,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want codes.InvalidArgument, got %v", status.Code(err))
	}
}
```

Add `GetIssuanceTrend` to `fakeRepo` (returns empty, no error, by default):

```go
func (f *fakeRepo) GetIssuanceTrend(_ context.Context, _ usecase.TrendBucket, _ time.Time) ([]usecase.TrendPoint, error) {
	return nil, nil
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/indexer && go test ./internal/adapter/grpc/... -run TestGetIssuanceTrend -v`
Expected: FAIL — compile error (`Server` doesn't implement `GetIssuanceTrend`; `NewServer` signature mismatch).

- [ ] **Step 3: Implement the handler and update `NewServer`**

Edit `services/indexer/internal/adapter/grpc/server.go`. Update the `Server` struct and `NewServer`:

```go
// Server implements certv1.CertificateQueryServer over the read-model ports.
type Server struct {
	repo  usecase.CertificateRepo
	src   usecase.EventSource
	sub   usecase.EventSubscriber
	trend *usecase.TrendService
	log   *slog.Logger
}

// NewServer constructs a Server. repo, src, sub, and trend must be non-nil.
func NewServer(repo usecase.CertificateRepo, src usecase.EventSource, sub usecase.EventSubscriber, trend *usecase.TrendService, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{repo: repo, src: src, sub: sub, trend: trend, log: log}
}
```

Add the handler after `GetIndexerStatus`:

```go
// GetIssuanceTrend returns a zero-filled certificate-issuance time series for the requested
// bucket granularity and range preset.
func (s *Server) GetIssuanceTrend(ctx context.Context, req *certv1.GetIssuanceTrendRequest) (*certv1.GetIssuanceTrendResponse, error) {
	bucket, err := fromProtoBucket(req.GetBucket())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	since, err := usecase.RangePresetToSince(bucket, req.GetRangePreset(), time.Now())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	points, err := s.trend.GetTrend(ctx, bucket, since, req.GetRangePreset())
	if err != nil {
		s.log.Error("GetIssuanceTrend", "err", err)
		return nil, status.Error(codes.Internal, errInternal)
	}

	protoPoints := make([]*certv1.TrendPoint, 0, len(points))
	for _, p := range points {
		protoPoints = append(protoPoints, &certv1.TrendPoint{
			BucketStart: timestamppb.New(p.BucketStart),
			Count:       uint64(p.Count), //nolint:gosec // count is always non-negative
		})
	}
	return &certv1.GetIssuanceTrendResponse{Points: protoPoints}, nil
}

// fromProtoBucket maps the proto enum to the usecase-layer type — keeps usecase framework-free.
func fromProtoBucket(b certv1.TrendBucket) (usecase.TrendBucket, error) {
	switch b {
	case certv1.TrendBucket_TREND_BUCKET_DAY:
		return usecase.TrendBucketDay, nil
	case certv1.TrendBucket_TREND_BUCKET_WEEK:
		return usecase.TrendBucketWeek, nil
	case certv1.TrendBucket_TREND_BUCKET_MONTH:
		return usecase.TrendBucketMonth, nil
	default:
		return 0, fmt.Errorf("%w: bucket must be day, week, or month", usecase.ErrInvalidTrendRequest)
	}
}
```

Add `"time"` to the file's import block if not already present.

- [ ] **Step 4: Update the `cmd/indexer/main.go` call site**

Edit `services/indexer/cmd/indexer/main.go` — in `main`, after `worker := usecase.NewWorker(...)` and before `s := buildGRPCServer(...)`, add:

```go
	trendService := usecase.NewTrendService(repo, cfg.ChainID)
```

Update the `buildGRPCServer` call and signature:

```go
	s := buildGRPCServer(repo, src, broadcaster, trendService, log)
```

```go
// buildGRPCServer wires the gRPC server with interceptors, health, and reflection.
func buildGRPCServer(repo usecase.CertificateRepo, src usecase.EventSource, sub usecase.EventSubscriber, trend *usecase.TrendService, log *slog.Logger) *grpc.Server {
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(log),
			loggingInterceptor(log),
		),
	)

	certv1.RegisterCertificateQueryServer(s, grpcadapter.NewServer(repo, src, sub, trend, log))
	// ... rest of the function body is unchanged
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/indexer && go test ./... -race -cover`
Expected: PASS across the whole indexer module — no more "missing method" build errors.

- [ ] **Step 6: Full module build + commit**

Run: `go build ./... && go vet ./... && gofmt -l services/indexer`
Expected: clean.

```bash
git add services/indexer/internal/adapter/grpc/server.go services/indexer/internal/adapter/grpc/server_test.go services/indexer/cmd/indexer/main.go
git commit -m "feat(indexer): GetIssuanceTrend gRPC handler, wire TrendService into the composition root"
```

---

### Task 8: Gateway REST — `GET /stats/trend`

**TDD: yes** for query-param validation/mapping (mirrors `certificates.go`'s `parsePageSize`/`isValidTokenID` pattern); **no** for the thin wiring, verified by build + the same test file's happy-path case.

**Files:**
- Create: `services/gateway/internal/httpapi/stats.go`
- Create: `services/gateway/internal/httpapi/stats_test.go`
- Modify: `services/gateway/internal/httpapi/router.go`

**Interfaces:**
- Consumes: `certv1.CertificateQueryClient.GetIssuanceTrend` (already on the gRPC-generated interface `Deps.Cert` once Task 7's proto regen lands).
- Produces: `GetIssuanceTrend(d Deps) http.HandlerFunc`, wired at `GET /stats/trend`.

- [ ] **Step 1: Write the failing test**

Create `services/gateway/internal/httpapi/stats_test.go`:

```go
package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// fakeTrendCertClient extends fakeCertClient (certificates_test.go) with GetIssuanceTrend.
// Embedding promotes GetCertificate/ListCertificates/StreamCertificateEvents/
// GetIndexerStatus from *fakeCertClient, so fakeTrendCertClient as a whole still satisfies
// certv1.CertificateQueryClient once this method is added.
type fakeTrendCertClient struct {
	*fakeCertClient
	trendResp *certv1.GetIssuanceTrendResponse
	trendErr  error
}

func (f *fakeTrendCertClient) GetIssuanceTrend(_ context.Context, _ *certv1.GetIssuanceTrendRequest, _ ...grpc.CallOption) (*certv1.GetIssuanceTrendResponse, error) {
	return f.trendResp, f.trendErr
}

func TestGetIssuanceTrendHandler_MissingBucket(t *testing.T) {
	client := &fakeTrendCertClient{fakeCertClient: &fakeCertClient{}}
	req := httptest.NewRequest(http.MethodGet, "/stats/trend?range=30d", nil)
	w := httptest.NewRecorder()

	GetIssuanceTrend(newDeps(client))(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetIssuanceTrendHandler_UnknownBucket(t *testing.T) {
	client := &fakeTrendCertClient{fakeCertClient: &fakeCertClient{}}
	req := httptest.NewRequest(http.MethodGet, "/stats/trend?bucket=year&range=30d", nil)
	w := httptest.NewRecorder()

	GetIssuanceTrend(newDeps(client))(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/gateway && go test ./internal/httpapi/... -run TestGetIssuanceTrendHandler -v`
Expected: FAIL — `GetIssuanceTrend` handler function doesn't exist.

- [ ] **Step 3: Implement the handler**

Create `services/gateway/internal/httpapi/stats.go`:

```go
package httpapi

import (
	"context"
	"net/http"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// bucketParams maps the REST ?bucket= value to its proto enum.
var bucketParams = map[string]certv1.TrendBucket{
	"day":   certv1.TrendBucket_TREND_BUCKET_DAY,
	"week":  certv1.TrendBucket_TREND_BUCKET_WEEK,
	"month": certv1.TrendBucket_TREND_BUCKET_MONTH,
}

// TrendPointDTO is one bucketed count in the JSON response.
type TrendPointDTO struct {
	BucketStart string `json:"bucketStart"` // RFC3339
	Count       uint64 `json:"count"`
}

// GetIssuanceTrend handles GET /stats/trend?bucket=day|week|month&range=<preset> — a
// certificate-issuance time series. Thin REST wrapper: validate, call the indexer's gRPC
// method, map to JSON, no business logic here (that's TrendService, indexer-side).
func GetIssuanceTrend(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		bucket, ok := bucketParams[q.Get("bucket")]
		if !ok {
			writeJSONError(w, http.StatusBadRequest, "bucket must be one of: day, week, month")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), d.RequestTimeout)
		defer cancel()

		resp, err := d.Cert.GetIssuanceTrend(ctx, &certv1.GetIssuanceTrendRequest{
			Bucket:      bucket,
			RangePreset: q.Get("range"),
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}

		points := make([]TrendPointDTO, 0, len(resp.GetPoints()))
		for _, p := range resp.GetPoints() {
			points = append(points, TrendPointDTO{
				BucketStart: p.GetBucketStart().AsTime().UTC().Format("2006-01-02T15:04:05Z07:00"),
				Count:       p.GetCount(),
			})
		}
		writeJSON(w, http.StatusOK, map[string][]TrendPointDTO{"points": points})
	}
}
```

- [ ] **Step 4: Wire the route**

Edit `services/gateway/internal/httpapi/router.go` — add after the `/certificates/stream` route:

```go
	mux.Handle("GET /stats/trend", GetIssuanceTrend(d))
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/gateway && go test ./... -race -cover`
Expected: PASS across the whole gateway module.

- [ ] **Step 6: Full verify + commit**

Run: `go build ./... && go vet ./... && gofmt -l services/gateway`
Expected: clean.

```bash
git add services/gateway/internal/httpapi/stats.go services/gateway/internal/httpapi/stats_test.go services/gateway/internal/httpapi/router.go
git commit -m "feat(gateway): GET /stats/trend — REST wrapper over GetIssuanceTrend"
```

**Anti-patterns to avoid:** don't add business logic (bucketing, zero-fill) in this handler — it belongs in `TrendService` (indexer-side). Don't skip the bucket-name validation — an unknown value must be a 400, not silently passed through as `TREND_BUCKET_UNSPECIFIED`.

---

## Phase 6 — Redis + asynq (issuance trend cache)

### Task 9: Add dependencies + config

**TDD: no** (dependency/config wiring).

**Files:**
- Modify: `go.mod` / `go.sum`
- Modify: `services/indexer/internal/config/config.go`

- [ ] **Step 1: Add the dependencies**

Run:
```bash
go get github.com/redis/go-redis/v9
go get github.com/hibiken/asynq
```
Expected: both added to `go.mod`/`go.sum`.

- [ ] **Step 2: Add `RedisAddr` to indexer config**

Edit `services/indexer/internal/config/config.go` — add a field to `Config`:

```go
	RedisAddr string
```

And in `Load()`, after the existing required-var checks:

```go
	if cfg.RedisAddr, err = mustenv("REDIS_ADDR"); err != nil {
		return Config{}, err
	}
```

- [ ] **Step 3: Verify + commit**

Run: `go build ./... && go vet ./...`
Expected: clean (config change alone doesn't break the build; `RedisAddr` is unused until Task 13, which is fine — Go doesn't flag unused struct fields).

```bash
git add go.mod go.sum services/indexer/internal/config/config.go
git commit -m "feat(indexer): add Redis + asynq dependencies, REDIS_ADDR config"
```

---

### Task 10: Redis-backed `TrendCache`

**TDD: yes** — cache hit/miss/marshal-error paths are easy to get subtly wrong.

**Files:**
- Create: `services/indexer/internal/adapter/cache/redis_trend_cache.go`
- Create: `services/indexer/internal/adapter/cache/redis_trend_cache_test.go`

**Interfaces:**
- Consumes: `usecase.TrendCache`, `usecase.TrendPoint` from Task 5.
- Produces: `cache.NewRedisTrendCache(client *redis.Client) *RedisTrendCache` satisfying `usecase.TrendCache`.

- [ ] **Step 1: Write the failing test**

Create `services/indexer/internal/adapter/cache/redis_trend_cache_test.go`:

```go
package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/cache"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

func newTestCache(t *testing.T) *cache.RedisTrendCache {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return cache.NewRedisTrendCache(client)
}

func TestRedisTrendCache_MissThenSetThenHit(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	_, ok, err := c.Get(ctx, "trend:v1:1:1:30d")
	if err != nil {
		t.Fatalf("Get (miss): %v", err)
	}
	if ok {
		t.Fatal("expected a miss on an empty cache")
	}

	want := []usecase.TrendPoint{{BucketStart: time.Now().UTC().Truncate(time.Second), Count: 3}}
	if err := c.Set(ctx, "trend:v1:1:1:30d", want); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok, err := c.Get(ctx, "trend:v1:1:1:30d")
	if err != nil {
		t.Fatalf("Get (hit): %v", err)
	}
	if !ok {
		t.Fatal("expected a hit after Set")
	}
	if len(got) != 1 || got[0].Count != 3 {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
```

Add `github.com/alicebob/miniredis/v2` as a test-only dependency: `go get github.com/alicebob/miniredis/v2`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/indexer && go test ./internal/adapter/cache/... -v`
Expected: FAIL — package `cache` doesn't exist yet.

- [ ] **Step 3: Implement `RedisTrendCache`**

Create `services/indexer/internal/adapter/cache/redis_trend_cache.go`:

```go
// Package cache implements usecase.TrendCache over Redis.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// cacheTTL bounds how long a cached entry lives if the refresh job ever stops running —
// the read path never depends on TTL for correctness; a miss just recomputes from Postgres.
const cacheTTL = 30 * time.Minute

var _ usecase.TrendCache = (*RedisTrendCache)(nil)

// RedisTrendCache implements usecase.TrendCache over a Redis client.
type RedisTrendCache struct {
	client *redis.Client
}

// NewRedisTrendCache constructs a RedisTrendCache over an already-connected client.
func NewRedisTrendCache(client *redis.Client) *RedisTrendCache {
	return &RedisTrendCache{client: client}
}

// Get returns the cached points for key, or (nil, false, nil) on a miss.
func (c *RedisTrendCache) Get(ctx context.Context, key string) ([]usecase.TrendPoint, bool, error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis get %s: %w", key, err)
	}

	var points []usecase.TrendPoint
	if err := json.Unmarshal(raw, &points); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached trend %s: %w", key, err)
	}
	return points, true, nil
}

// Set writes points to key with a TTL backstop.
func (c *RedisTrendCache) Set(ctx context.Context, key string, points []usecase.TrendPoint) error {
	raw, err := json.Marshal(points)
	if err != nil {
		return fmt.Errorf("marshal trend %s: %w", key, err)
	}
	if err := c.client.Set(ctx, key, raw, cacheTTL).Err(); err != nil {
		return fmt.Errorf("redis set %s: %w", key, err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/indexer && go test ./internal/adapter/cache/... -race -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/indexer/internal/adapter/cache/redis_trend_cache.go services/indexer/internal/adapter/cache/redis_trend_cache_test.go go.mod go.sum
git commit -m "feat(indexer): Redis-backed TrendCache"
```

---

### Task 11: asynq refresh job — task type, constructor, handler

**TDD: yes** — the handler's "iterate every preset combination" loop is real logic worth a fake-based test.

**Files:**
- Create: `services/indexer/internal/adapter/asynqjobs/refresh_trend.go`
- Create: `services/indexer/internal/adapter/asynqjobs/refresh_trend_test.go`
- Modify: `services/indexer/internal/usecase/ports.go` (add `TrendRefreshTaskType` const + `TaskEnqueuer` port)

**Interfaces:**
- Consumes: `usecase.TrendService`, `usecase.AllowedPresets`, `usecase.RangePresetToSince` from Phase 5.
- Produces: `usecase.TrendRefreshTaskType` (string const), `usecase.TaskEnqueuer` interface, `asynqjobs.NewRefreshTrendCacheTask() *asynq.Task`, `asynqjobs.NewRefreshTrendCacheHandler(trend *usecase.TrendService, log *slog.Logger) *RefreshTrendCacheHandler` (satisfies `asynq.Handler`).

- [ ] **Step 1: Add the port + task-type constant**

Edit `services/indexer/internal/usecase/ports.go` — add:

```go
// TrendRefreshTaskType identifies the asynq task that recomputes and caches every
// supported trend bucket/preset combination. Defined here (not in the asynq adapter) so
// Worker can reference it without importing adapter code.
const TrendRefreshTaskType = "trend:refresh"

// TaskEnqueuer lets the Worker trigger background jobs after ingest. Optional — the Worker
// is nil-safe if none is wired.
type TaskEnqueuer interface {
	// EnqueueUnique enqueues a task, deduped by taskID: a second call with the same taskID
	// while one is still pending/processing is a no-op.
	EnqueueUnique(ctx context.Context, taskType, taskID string) error
}
```

- [ ] **Step 2: Write the failing test**

Create `services/indexer/internal/adapter/asynqjobs/refresh_trend_test.go`:

```go
package asynqjobs_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/asynqjobs"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// fakeTrendRefresher records every (bucket, since, preset) combination it's asked to refresh.
type fakeTrendRefresher struct {
	calls []string
}

func (f *fakeTrendRefresher) RefreshCache(_ context.Context, bucket usecase.TrendBucket, _ time.Time, preset string) ([]usecase.TrendPoint, error) {
	f.calls = append(f.calls, preset)
	_ = bucket
	return nil, nil
}

func TestRefreshTrendCacheHandler_RefreshesEveryPreset(t *testing.T) {
	refresher := &fakeTrendRefresher{}
	h := asynqjobs.NewRefreshTrendCacheHandler(refresher, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := h.ProcessTask(context.Background(), asynqjobs.NewRefreshTrendCacheTask()); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}

	wantCount := 0
	for _, presets := range usecase.AllowedPresets() {
		wantCount += len(presets)
	}
	if len(refresher.calls) != wantCount {
		t.Fatalf("refreshed %d combinations, want %d", len(refresher.calls), wantCount)
	}
}

func TestNewRefreshTrendCacheTask_HasCorrectType(t *testing.T) {
	task := asynqjobs.NewRefreshTrendCacheTask()
	if task.Type() != usecase.TrendRefreshTaskType {
		t.Fatalf("task type = %q, want %q", task.Type(), usecase.TrendRefreshTaskType)
	}
}

var _ asynq.Handler = (*asynqjobs.RefreshTrendCacheHandler)(nil)
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd services/indexer && go test ./internal/adapter/asynqjobs/... -v`
Expected: FAIL — package doesn't exist yet.

- [ ] **Step 4: Implement the job**

Create `services/indexer/internal/adapter/asynqjobs/refresh_trend.go`:

```go
// Package asynqjobs holds the asynq task definitions and handlers for the indexer's
// background jobs (currently just the trend-cache refresh).
package asynqjobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// trendRefresher is the subset of *usecase.TrendService this handler needs — a narrow
// interface makes the handler testable without a real Postgres/Redis-backed TrendService.
type trendRefresher interface {
	RefreshCache(ctx context.Context, bucket usecase.TrendBucket, since time.Time, preset string) ([]usecase.TrendPoint, error)
}

// NewRefreshTrendCacheTask builds the (payload-less) refresh task — recompute-all needs no
// parameters, so every enqueue of this type is identical, which is exactly what makes
// asynq.Unique-based deduplication (see the asynqjobs.Enqueuer) work.
func NewRefreshTrendCacheTask() *asynq.Task {
	return asynq.NewTask(usecase.TrendRefreshTaskType, nil, asynq.MaxRetry(2))
}

// RefreshTrendCacheHandler recomputes every supported bucket/range-preset combination and
// writes each into the cache via TrendService.RefreshCache. Registered against
// usecase.TrendRefreshTaskType in the asynq ServeMux.
type RefreshTrendCacheHandler struct {
	trend trendRefresher
	log   *slog.Logger
}

// NewRefreshTrendCacheHandler constructs a RefreshTrendCacheHandler.
func NewRefreshTrendCacheHandler(trend trendRefresher, log *slog.Logger) *RefreshTrendCacheHandler {
	if log == nil {
		log = slog.Default()
	}
	return &RefreshTrendCacheHandler{trend: trend, log: log}
}

// ProcessTask satisfies asynq.Handler. O(number of bucket/preset combinations) — a small,
// fixed table (9 combinations as of Phase 5), not proportional to certificate count.
func (h *RefreshTrendCacheHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	now := time.Now()
	for bucket, presets := range usecase.AllowedPresets() {
		for preset := range presets {
			since, err := usecase.RangePresetToSince(bucket, preset, now)
			if err != nil {
				return fmt.Errorf("range preset bucket=%d preset=%s: %w", bucket, preset, err)
			}
			if _, err := h.trend.RefreshCache(ctx, bucket, since, preset); err != nil {
				h.log.Error("refresh trend cache", "bucket", bucket, "preset", preset, "err", err)
				return fmt.Errorf("refresh bucket=%d preset=%s: %w", bucket, preset, err)
			}
		}
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd services/indexer && go test ./internal/adapter/asynqjobs/... -race -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add services/indexer/internal/usecase/ports.go services/indexer/internal/adapter/asynqjobs/refresh_trend.go services/indexer/internal/adapter/asynqjobs/refresh_trend_test.go
git commit -m "feat(indexer): asynq refresh-trend-cache task + handler"
```

---

### Task 12: Wire `TaskEnqueuer` into the Worker (debounced enqueue after upsert)

**TDD: yes** — nil-safety and the "enqueue after every successful upsert" trigger are exactly the class of thing worth a regression test (mirrors the existing `EventPublisher`/`SetPublisher` test pattern from BE-2).

**Files:**
- Modify: `services/indexer/internal/usecase/worker.go`
- Modify: `services/indexer/internal/usecase/worker_test.go`
- Create: `services/indexer/internal/adapter/asynqjobs/enqueuer.go`
- Create: `services/indexer/internal/adapter/asynqjobs/enqueuer_test.go`

**Interfaces:**
- Produces: `Worker.SetEnqueuer(e usecase.TaskEnqueuer)`; `asynqjobs.NewEnqueuer(client *asynq.Client) *Enqueuer` satisfying `usecase.TaskEnqueuer`.

- [ ] **Step 1: Write the failing Worker test**

Append to `services/indexer/internal/usecase/worker_test.go`:

```go
// fakeEnqueuer implements usecase.TaskEnqueuer for tests.
type fakeEnqueuer struct {
	enqueued []string // taskType per call
}

func (f *fakeEnqueuer) EnqueueUnique(_ context.Context, taskType, _ string) error {
	f.enqueued = append(f.enqueued, taskType)
	return nil
}

func TestWorker_EnqueuesTrendRefresh_OnSuccessfulUpsert(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	enq := &fakeEnqueuer{}
	w := newWorker(src, repo)
	w.SetEnqueuer(enq)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(enq.enqueued) != 1 || enq.enqueued[0] != usecase.TrendRefreshTaskType {
		t.Fatalf("want 1 enqueue of %q, got %v", usecase.TrendRefreshTaskType, enq.enqueued)
	}
}

func TestWorker_NilEnqueuer_NoPanic(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	w := newWorker(src, repo) // SetEnqueuer never called
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/indexer && go test ./internal/usecase/... -run TestWorker_EnqueuesTrendRefresh -v`
Expected: FAIL — `w.SetEnqueuer` undefined.

- [ ] **Step 3: Wire the enqueuer into `Worker`**

Edit `services/indexer/internal/usecase/worker.go` — add a field and setter:

```go
// add to the Worker struct:
	enqueuer TaskEnqueuer // optional; nil-safe

// add method, near SetPublisher:
// SetEnqueuer wires an optional background-task enqueuer. Call before Run(); safe to never
// call — processLog no-ops the enqueue step when enqueuer is nil.
func (w *Worker) SetEnqueuer(e TaskEnqueuer) {
	w.enqueuer = e
}
```

Edit `processLog` — add after the existing `w.pub` publish block:

```go
	if w.enqueuer != nil {
		if err := w.enqueuer.EnqueueUnique(ctx, TrendRefreshTaskType, TrendRefreshTaskType); err != nil {
			// Non-fatal: ingest correctness never depends on the cache-refresh job succeeding —
			// a failed enqueue just means the trend cache stays stale until the cron backstop runs.
			w.log.Warn("enqueue trend refresh", "err", err)
		}
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/indexer && go test ./internal/usecase/... -race -v`
Expected: PASS — new tests and every pre-existing worker test.

- [ ] **Step 5: Write the failing Enqueuer test**

Create `services/indexer/internal/adapter/asynqjobs/enqueuer_test.go`:

```go
package asynqjobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/asynqjobs"
)

func TestEnqueuer_EnqueueUnique_DedupesWithinTTL(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := asynq.NewClient(asynq.RedisClientOpt{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	enq := asynqjobs.NewEnqueuer(client)

	ctx := context.Background()
	if err := enq.EnqueueUnique(ctx, "trend:refresh", "trend:refresh"); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	// second call with the same taskID must be a no-op, not an error
	if err := enq.EnqueueUnique(ctx, "trend:refresh", "trend:refresh"); err != nil {
		t.Fatalf("second (duplicate) enqueue should be absorbed, got: %v", err)
	}
	_ = time.Second // no sleep needed — dedup is immediate within the TTL window
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `cd services/indexer && go test ./internal/adapter/asynqjobs/... -run TestEnqueuer -v`
Expected: FAIL — `asynqjobs.NewEnqueuer` doesn't exist.

- [ ] **Step 7: Implement the Enqueuer**

Create `services/indexer/internal/adapter/asynqjobs/enqueuer.go`:

```go
package asynqjobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// uniqueTTL bounds the dedup window — matches the cron backstop cadence order-of-magnitude
// (Phase 6 design: 15-minute backstop), so a burst of enqueues collapses into one task
// without silently dropping a legitimately-later refresh.
const uniqueTTL = 5 * time.Minute

var _ usecase.TaskEnqueuer = (*Enqueuer)(nil)

// Enqueuer implements usecase.TaskEnqueuer over an asynq.Client.
type Enqueuer struct {
	client *asynq.Client
}

// NewEnqueuer constructs an Enqueuer over an already-configured asynq.Client.
func NewEnqueuer(client *asynq.Client) *Enqueuer {
	return &Enqueuer{client: client}
}

// EnqueueUnique enqueues taskType, deduped by taskID within uniqueTTL — a duplicate call
// while one is pending is absorbed as a no-op, not an error.
func (e *Enqueuer) EnqueueUnique(ctx context.Context, taskType, taskID string) error {
	task := asynq.NewTask(taskType, nil)
	_, err := e.client.EnqueueContext(ctx, task, asynq.TaskID(taskID), asynq.Unique(uniqueTTL))
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", taskType, err)
	}
	return nil
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `cd services/indexer && go test ./internal/adapter/asynqjobs/... -race -v`
Expected: PASS

- [ ] **Step 9: Full verify + commit**

Run: `go build ./... && go vet ./... && gofmt -l services/indexer && go test ./services/indexer/... -race -cover`
Expected: clean, all green.

```bash
git add services/indexer/internal/usecase/worker.go services/indexer/internal/usecase/worker_test.go services/indexer/internal/adapter/asynqjobs/enqueuer.go services/indexer/internal/adapter/asynqjobs/enqueuer_test.go go.mod go.sum
git commit -m "feat(indexer): debounced trend-refresh enqueue from the ingest Worker"
```

---

### Task 13: Composition root — Redis, asynq server/scheduler, cache-wired `TrendService`

**TDD: no** (composition-root wiring — verify by running + the existing test suite continuing to pass).

**Files:**
- Modify: `services/indexer/cmd/indexer/main.go`

**Interfaces:**
- Consumes: everything from Tasks 1-4 (`cache.NewRedisTrendCache`, `asynqjobs.NewEnqueuer`, `asynqjobs.NewRefreshTrendCacheHandler`, `asynqjobs.NewRefreshTrendCacheTask`, `Worker.SetEnqueuer`, `TrendService.SetCache`).

- [ ] **Step 1: Wire Redis + the cache into `TrendService`**

Edit `services/indexer/cmd/indexer/main.go` — add imports for `"github.com/hibiken/asynq"`, `"github.com/redis/go-redis/v9"`, `"github.com/oksasatya/skillpass/services/indexer/internal/adapter/asynqjobs"`, `"github.com/oksasatya/skillpass/services/indexer/internal/adapter/cache"`.

After `trendService := usecase.NewTrendService(repo, cfg.ChainID)`, add:

```go
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close() //nolint:errcheck // best-effort close on process exit

	trendService.SetCache(cache.NewRedisTrendCache(redisClient))

	asynqRedisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}
	asynqClient := asynq.NewClient(asynqRedisOpt)
	defer asynqClient.Close() //nolint:errcheck // best-effort close on process exit

	worker.SetEnqueuer(asynqjobs.NewEnqueuer(asynqClient))
```

(`worker.SetEnqueuer(...)` must be added right after the existing `worker.SetPublisher(broadcaster)` line.)

- [ ] **Step 2: Build the asynq server + scheduler, and bundle the supervised services**

`runConcurrently` already sits at the Sonar-preferred 5-param ceiling (`ctx, s, worker, addr, log`); adding the asynq server/scheduler would push it to 7. Introduce a small struct instead of growing the param list further.

Add a new helper function and a bundling struct, and call the helper from `main` before `runConcurrently`:

```go
	asynqServer, asynqMux, scheduler := buildAsynqRuntime(asynqRedisOpt, trendService, log)
```

```go
// buildAsynqRuntime wires the asynq processing server (handles enqueued refresh tasks) and
// scheduler (15-minute cron backstop, in case an event-triggered enqueue is ever missed).
// Returns the mux alongside the server since Run(mux) needs the exact same instance the
// handler was registered on.
func buildAsynqRuntime(redisOpt asynq.RedisClientOpt, trend *usecase.TrendService, log *slog.Logger) (*asynq.Server, *asynq.ServeMux, *asynq.Scheduler) {
	server := asynq.NewServer(redisOpt, asynq.Config{Concurrency: 5})

	mux := asynq.NewServeMux()
	mux.Handle(usecase.TrendRefreshTaskType, asynqjobs.NewRefreshTrendCacheHandler(trend, log))

	scheduler := asynq.NewScheduler(redisOpt, nil)
	if _, err := scheduler.Register("*/15 * * * *", asynqjobs.NewRefreshTrendCacheTask()); err != nil {
		log.Error("register trend-refresh cron", "err", err)
	}

	return server, mux, scheduler
}

// runtimeServices bundles the long-running components runConcurrently supervises —
// introduced once adding the asynq server/scheduler would have pushed runConcurrently
// past the Sonar-preferred 5-param ceiling.
type runtimeServices struct {
	grpcServer  *grpc.Server
	worker      *usecase.Worker
	asynqServer *asynq.Server
	asynqMux    *asynq.ServeMux
	scheduler   *asynq.Scheduler
}
```

Update the `main()` call site:

```go
	svc := runtimeServices{
		grpcServer:  s,
		worker:      worker,
		asynqServer: asynqServer,
		asynqMux:    asynqMux,
		scheduler:   scheduler,
	}
	if err := runConcurrently(ctx, svc, cfg.GRPCAddr, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
```

Replace `runConcurrently`'s definition (4 params now: `ctx, svc, addr, log`):

```go
// runConcurrently starts the worker, gRPC server, asynq processing server, and asynq
// scheduler; monitors ctx for graceful shutdown of all four.
func runConcurrently(ctx context.Context, svc runtimeServices, addr string, log *slog.Logger) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return svc.worker.Run(gCtx)
	})

	g.Go(func() error {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		log.Info("gRPC server listening", "addr", addr)
		return svc.grpcServer.Serve(lis)
	})

	g.Go(func() error {
		return svc.asynqServer.Run(svc.asynqMux)
	})

	g.Go(func() error {
		return svc.scheduler.Run()
	})

	g.Go(func() error {
		<-gCtx.Done()
		svc.grpcServer.GracefulStop()
		svc.asynqServer.Shutdown()
		svc.scheduler.Shutdown()
		return nil
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	}
	return nil
}
```

`asynq.Server.Run` and `asynq.Scheduler.Run` are both **blocking** calls (they don't return until `Shutdown` is called from another goroutine) — the same shape the existing `svc.grpcServer.Serve(lis)` goroutine already has, so both drop cleanly into the same errgroup pattern.

- [ ] **Step 2: Build to confirm the composition root compiles**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Full-module verification**

Run: `go build ./... && go vet ./... && gofmt -l services/indexer && go test ./services/indexer/... -race -cover`
Expected: clean, all green (this doesn't require a live Redis — nothing in the existing unit/integration test suite exercises the composition root directly; `cmd/indexer` has 0% test coverage already, consistent with the rest of this codebase's `cmd/` packages).

- [ ] **Step 4: Manual smoke test against the dev stack (see Task 14 for docker-compose)**

Deferred to Task 14, once Redis is in `docker-compose.yml` — running `cmd/indexer` locally right now would fail fast on `config.Load()`'s new required `REDIS_ADDR` check, which is expected.

- [ ] **Step 5: Commit**

```bash
git add services/indexer/cmd/indexer/main.go
git commit -m "feat(indexer): wire Redis + asynq server/scheduler into the composition root"
```

---

### Task 14: docker-compose + Makefile Redis wiring

**TDD: no** (infra).

**Files:**
- Modify: `deploy/docker-compose.yml`
- Modify: `Makefile`

- [ ] **Step 1: Add the Redis service**

Edit `deploy/docker-compose.yml` — add a new service before `indexer`:

```yaml
  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - skillpass
```

Update the `indexer` service's `depends_on` and `environment`:

```yaml
    depends_on:
      postgres:
        condition: service_healthy
      anvil:
        condition: service_started
      redis:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/skillpass?sslmode=disable
      ETH_RPC_URL: http://anvil:8545
      REDIS_ADDR: "redis:6379"
      # Deterministic address: account[0] nonce=0 CREATE on a fresh anvil
      CONTRACT_ADDRESS: "0x5FbDB2315678afecb367f032d93F642f64180aa3"
      CHAIN_ID: "31337"
      START_BLOCK: "0"
      GRPC_ADDR: ":50051"
      POLL_INTERVAL: "2s"
      BATCH_SIZE: "2000"
```

- [ ] **Step 2: Update the `run-indexer` Makefile doc comment**

Edit `Makefile` — update the comment above `run-indexer`:

```makefile
# Run the indexer locally (requires env vars — see services/indexer/internal/config/config.go)
# Required: DATABASE_URL, ETH_RPC_URL, CONTRACT_ADDRESS, CHAIN_ID, REDIS_ADDR
# Optional: GRPC_ADDR (":50051"), START_BLOCK ("0"), BATCH_SIZE ("2000"), POLL_INTERVAL ("5s")
run-indexer:
	go run ./services/indexer/cmd/indexer
```

- [ ] **Step 3: Verify the compose file**

Run: `docker compose -f deploy/docker-compose.yml config`
Expected: exits 0, no YAML errors.

- [ ] **Step 4: Full dev-stack smoke test**

Run: `make dev-up && make dev-seed`
Expected: all 5 containers (`postgres`, `anvil`, `redis`, `indexer`, `gateway`) start; seed succeeds as before.

Run: `curl -sf http://localhost:8080/stats/trend?bucket=day&range=30d`
Expected: a 200 response with a `points` array covering the last 30 UTC days, non-zero counts on the days seeded certificates were issued.

Run: `make dev-down`
Expected: clean teardown.

- [ ] **Step 5: Commit**

```bash
git add deploy/docker-compose.yml Makefile
git commit -m "feat(infra): add redis to the dev docker-compose stack"
```

---

## Plan-wide verification (run once, after all three phases)

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./services/... -race -cover
```
Expected: clean build, no gofmt output, every package green.

Per the project's standing code-review discipline (§18): dispatch `superpowers:requesting-code-review` against the full diff (`git rev-parse HEAD~<N>..HEAD` spanning every commit in this plan) before considering Phase 4-6 done — algorithmic-complexity + go-review + Sonar-Go + a critical-thinking pass, same gate BE-2 went through.
