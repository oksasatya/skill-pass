# SkillPass Phase 7 — Notification & Metadata Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a gateway metadata endpoint (ERC-721 JSON for wallets/marketplaces) and a reorg-safe, crash-safe webhook notification pipeline (new `services/notify` + indexer-owned outbox/sweep) for certificate-issuance events.

**Architecture:** Metadata is a pure reshape added to the existing `services/gateway` (no new gRPC). Webhook delivery is a new Postgres-free `services/notify` service that consumes `webhook:deliver` asynq tasks; the indexer (which already owns Postgres) durably records "we owe a webhook" via a new `webhook_outbox` table keyed by `(chain_id, tx_hash, token_id)` — stable across reorg re-mining, unlike the block-relative `log_index` — with a fast-path enqueue plus a `webhook:sweep` cron backstop (mirroring the existing Phase 6 trend-refresh scheduler/handler split) so a crash or transient failure never silently drops a notification.

**Tech Stack:** Go 1.26, pgx/v5 + sqlc (indexer Postgres), asynq + Redis (shared queue, indexer produces, notify consumes), net/http (gateway + notify), goose migrations.

**Design doc:** `docs/superpowers/specs/2026-07-01-skillpass-phase7-notification-metadata-design.md` — read this first for the full reasoning behind the outbox/sweep design (it was adversarially reviewed in two rounds and found two real defects in earlier drafts; the sections below implement the converged design, not the rejected ones).

## Global Constraints

- **Single Go module** `github.com/oksasatya/skillpass` at repo root — `services/notify/` is a new directory under the same module; no new `go.mod`.
- **Hexagonal-lite (HARD):** `internal/usecase` packages (both indexer's and notify's) import nothing from `internal/adapter`, third-party HTTP/asynq/pgx packages, or each other. Adapters implement `usecase`-defined port interfaces.
- **Gateway boundary (HARD):** the gateway's Go code gains **zero** new imports of Redis/asynq/Postgres in this plan — the metadata endpoint only ever calls the existing `GetCertificate` gRPC method.
- **Notify boundary (HARD):** `services/notify` never imports Postgres/pgx — all durable webhook state lives in the indexer, which already owns Postgres.
- **Dedup key:** `(chain_id, tx_hash, token_id)` for the webhook outbox — deliberately NOT `log_index` (block-relative, defeated by reorg re-mining) and a deliberately different key shape than the existing `certificates` table's `UNIQUE (chain_id, tx_hash, log_index)` constraint (different question: "same log entry" vs. "already notified about this certificate").
- **`InsertWebhookOutbox` failure is FATAL** in `Worker.processLog` (propagates, so the batch retries next poll) — this is the specific mechanism that closes the crash-window gap found in review. The subsequent enqueue-to-asynq step stays best-effort/logged, same as the existing trend-refresh pattern.
- **Sonar-Go from first commit** (paste into every task):

```
# Sonar-Go guardrails — write compliant from the first commit
- go:S107 — ≤7 params (≤5 preferred; past that a Deps/Opts struct).
- go:S3776 — cognitive complexity ≤15 → extract helpers; t.Run subtests.
- go:S1192 — const for any string literal duplicated 3+ times.
- errcheck (handle every error), gosec, govulncheck. Wrap with %w; sentinel errors + errors.Is/As.
```

- **TDD verdicts are stated per task below** — honor them exactly; a task with `TDD: yes` writes the failing test FIRST.
- **Go/TS/backend implementer briefs**: invoke `golang-expert` first (hub skill — auto-chains go-patterns/go-review/go-test/go-error-handling/go-concurrency-patterns + algorithmic-complexity + relevant superpowers; follow its Auto-chain section) plus `senior-backend`, before writing any code in a Go task.

---

## Task 1: Migration — `webhook_outbox` table

**TDD: no** (pure SQL DDL — verified by applying it against a throwaway Postgres, same discipline as the Phase 4-6 plan's migration task).

**Files:**
- Create: `services/indexer/migrations/00003_webhook_outbox.sql`

- [ ] **Step 1: Write the migration**

Create `services/indexer/migrations/00003_webhook_outbox.sql`:

```sql
-- +goose Up
CREATE TABLE webhook_outbox (
    id           BIGSERIAL PRIMARY KEY,
    chain_id     BIGINT NOT NULL,
    tx_hash      TEXT NOT NULL,
    token_id     TEXT NOT NULL,
    payload      JSONB NOT NULL,
    enqueued_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chain_id, tx_hash, token_id)
);

-- Partial index: the sweep query (WHERE enqueued_at IS NULL) only ever touches unenqueued
-- rows, which stay a small fraction of the table as it grows — O(log k) via this index,
-- not an O(n) scan of the whole outbox history.
CREATE INDEX idx_webhook_outbox_unenqueued ON webhook_outbox (id) WHERE enqueued_at IS NULL;

-- +goose Down
DROP TABLE webhook_outbox;
```

- [ ] **Step 2: Verify it applies cleanly against a throwaway Postgres**

Run:
```bash
docker run --rm -d --name skillpass-migrate-test-p7 -e POSTGRES_PASSWORD=pg -p 55433:5432 postgres:17
sleep 3
DATABASE_URL="postgres://postgres:pg@localhost:55433/postgres?sslmode=disable" \
  go run -C services/indexer -tags migrate_verify ./cmd/indexer 2>&1 | head -5 || true
```

Since there's no standalone migration-verify binary (the Makefile's `migrate-test` target already references a removed `cmd/verify-migrate` binary — a pre-existing gap, not this task's scope), verify directly with `goose`:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir services/indexer/migrations postgres \
  "postgres://postgres:pg@localhost:55433/postgres?sslmode=disable" up
```

Expected: exits 0, output ends with `OK   00003_webhook_outbox.sql`.

Then verify the Down migration too:
```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir services/indexer/migrations postgres \
  "postgres://postgres:pg@localhost:55433/postgres?sslmode=disable" down
```

Expected: exits 0, `webhook_outbox` table dropped.

Clean up:
```bash
docker stop skillpass-migrate-test-p7
```

- [ ] **Step 3: Commit**

```bash
git add services/indexer/migrations/00003_webhook_outbox.sql
git commit -m "feat(indexer): add webhook_outbox migration for reorg-safe webhook dedup"
```

---

## Task 2: `TaskEnqueuer.EnqueueUnique` gains a payload parameter

**TDD: yes** (the payload-passthrough behavior is exactly what the plan needs to prove works, and it's cheap to verify with a failing test first).

**Files:**
- Modify: `services/indexer/internal/usecase/ports.go`
- Modify: `services/indexer/internal/usecase/worker.go`
- Modify: `services/indexer/internal/adapter/asynqjobs/enqueuer.go`
- Modify: `services/indexer/internal/usecase/worker_test.go`
- Test: `services/indexer/internal/adapter/asynqjobs/enqueuer_test.go`

**Interfaces:**
- Produces: `usecase.TaskEnqueuer.EnqueueUnique(ctx context.Context, taskType, taskID string, payload []byte) error` — the existing method gains a fourth parameter; all three existing call sites (real adapter, worker.go's trend-refresh call, the test fake) must be updated in this same task.

- [ ] **Step 1: Write the failing test** (asserts payload bytes reach the enqueued asynq task)

Add to `services/indexer/internal/adapter/asynqjobs/enqueuer_test.go` (append after the existing `TestEnqueuer_EnqueueUnique_DedupesWithinTTL`):

```go
func TestEnqueuer_EnqueueUnique_PayloadReachesTask(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := asynq.NewClient(asynq.RedisClientOpt{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	enq := asynqjobs.NewEnqueuer(client)

	ctx := context.Background()
	wantPayload := []byte(`{"tokenId":"1"}`)
	if err := enq.EnqueueUnique(ctx, "webhook:deliver", "webhook:deliver:1", wantPayload); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: mr.Addr()})
	t.Cleanup(func() { _ = inspector.Close() })
	tasks, err := inspector.ListPendingTasks("default")
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d pending tasks, want 1", len(tasks))
	}
	if string(tasks[0].Payload) != string(wantPayload) {
		t.Errorf("payload = %s, want %s", tasks[0].Payload, wantPayload)
	}
}
```

Also update the existing `TestEnqueuer_EnqueueUnique_DedupesWithinTTL` test's two calls (they currently call `EnqueueUnique` with 3 args) — add a `nil` payload argument to both:

```go
	if err := enq.EnqueueUnique(ctx, "trend:refresh", "trend:refresh", nil); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	// second call with the same taskID must be a no-op, not an error
	if err := enq.EnqueueUnique(ctx, "trend:refresh", "trend:refresh", nil); err != nil {
		t.Fatalf("second (duplicate) enqueue should be absorbed, got: %v", err)
	}
```

- [ ] **Step 2: Run tests to verify they fail (compile error — signature mismatch)**

Run: `go test ./services/indexer/internal/adapter/asynqjobs/... -run TestEnqueuer -v`
Expected: FAIL — compile error, `not enough arguments in call to enq.EnqueueUnique` (interface/impl still has the old 3-arg signature).

- [ ] **Step 3: Update the port interface**

Edit `services/indexer/internal/usecase/ports.go` — replace the `TaskEnqueuer` interface (lines 96–102):

```go
// TaskEnqueuer lets the Worker trigger background jobs after ingest. Optional — the Worker
// is nil-safe if none is wired.
type TaskEnqueuer interface {
	// EnqueueUnique enqueues a task, deduped by taskID: a second call with the same taskID
	// while one is still pending/processing is a no-op. payload may be nil.
	EnqueueUnique(ctx context.Context, taskType, taskID string, payload []byte) error
}
```

- [ ] **Step 4: Update the real adapter**

Edit `services/indexer/internal/adapter/asynqjobs/enqueuer.go` — replace `EnqueueUnique` (lines 30–42):

```go
// EnqueueUnique enqueues taskType, deduped by taskID within uniqueTTL — a duplicate call
// while one is pending is absorbed as a no-op, not an error. payload may be nil.
func (e *Enqueuer) EnqueueUnique(ctx context.Context, taskType, taskID string, payload []byte) error {
	task := asynq.NewTask(taskType, payload)
	_, err := e.client.EnqueueContext(ctx, task, asynq.TaskID(taskID), asynq.Unique(uniqueTTL))
	// Both indicate an equivalent task is already pending/active: the common case is
	// ErrDuplicateTask (the Unique key is still held); ErrTaskIDConflict is the narrow
	// window where the unique key expired but the fixed TaskID row is still present.
	// Either way there's nothing new for the caller to do -- absorb both as a no-op.
	if errors.Is(err, asynq.ErrDuplicateTask) || errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", taskType, err)
	}
	return nil
}
```

- [ ] **Step 5: Update `worker.go`'s trend-refresh call site**

Edit `services/indexer/internal/usecase/worker.go` — in `processLog` (around line 230), change:

```go
	if w.enqueuer != nil {
		if err := w.enqueuer.EnqueueUnique(ctx, TrendRefreshTaskType, TrendRefreshTaskType); err != nil {
```

to:

```go
	if w.enqueuer != nil {
		if err := w.enqueuer.EnqueueUnique(ctx, TrendRefreshTaskType, TrendRefreshTaskType, nil); err != nil {
```

- [ ] **Step 6: Update the test fake**

Edit `services/indexer/internal/usecase/worker_test.go` — replace the `fakeEnqueuer` type and its method (around line 325–331):

```go
// fakeEnqueuer implements usecase.TaskEnqueuer for tests.
type fakeEnqueuer struct {
	enqueued []string // taskType per call
	payloads [][]byte // payload per call, parallel to enqueued
}

func (f *fakeEnqueuer) EnqueueUnique(_ context.Context, taskType, _ string, payload []byte) error {
	f.enqueued = append(f.enqueued, taskType)
	f.payloads = append(f.payloads, payload)
	return nil
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go build ./... && go test ./services/indexer/... -v -run 'TestEnqueuer|TestWorker'`
Expected: PASS — all tests green, including the new `TestEnqueuer_EnqueueUnique_PayloadReachesTask`.

- [ ] **Step 8: Commit**

```bash
git add services/indexer/internal/usecase/ports.go services/indexer/internal/usecase/worker.go \
        services/indexer/internal/adapter/asynqjobs/enqueuer.go \
        services/indexer/internal/usecase/worker_test.go \
        services/indexer/internal/adapter/asynqjobs/enqueuer_test.go
git commit -m "feat(indexer): TaskEnqueuer.EnqueueUnique gains a payload parameter"
```

---

## Task 3: `CertificateRepo` webhook outbox methods

**TDD: yes** (the dedup behavior is exactly what the whole reorg-safety argument rests on — must be red-green tested against a real Postgres, not assumed).

**Files:**
- Modify: `services/indexer/internal/usecase/ports.go`
- Modify: `services/indexer/internal/db/queries.sql`
- Modify: `services/indexer/internal/adapter/postgres/certificate_repo.go`
- Test: `services/indexer/internal/adapter/postgres/certificate_repo_test.go`

**Interfaces:**
- Consumes: Task 1's `webhook_outbox` table.
- Produces: `usecase.WebhookOutboxEntry{ID int64; Payload []byte}`; `usecase.CertificateRepo.InsertWebhookOutbox(ctx, chainID int64, txHash, tokenID string, payload []byte) (id int64, isNew bool, err error)`; `.ListUnenqueuedWebhookOutbox(ctx, limit int) ([]usecase.WebhookOutboxEntry, error)`; `.MarkWebhookOutboxEnqueued(ctx, id int64) error`.

- [ ] **Step 1: Write the failing tests**

Add to `services/indexer/internal/adapter/postgres/certificate_repo_test.go` (append at the end of the file):

```go
func TestInsertWebhookOutbox_DedupesByChainTxToken(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	payload := []byte(`{"tokenId":"1"}`)

	id1, isNew1, err := repo.InsertWebhookOutbox(ctx, 31337, "0xabc", "1", payload)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !isNew1 {
		t.Fatal("first insert should be new")
	}
	if id1 == 0 {
		t.Fatal("expected a non-zero id")
	}

	id2, isNew2, err := repo.InsertWebhookOutbox(ctx, 31337, "0xabc", "1", payload)
	if err != nil {
		t.Fatalf("second (duplicate) insert: %v", err)
	}
	if isNew2 {
		t.Fatal("second insert of the same (chain_id, tx_hash, token_id) must not be new")
	}
	if id2 != 0 {
		t.Fatal("duplicate insert should return a zero id (nothing was returned)")
	}

	// A DIFFERENT token_id sharing the same tx_hash (hypothetical batch-mint-via-multicall
	// in one transaction) must still be treated as a genuinely new event, not collapsed.
	id3, isNew3, err := repo.InsertWebhookOutbox(ctx, 31337, "0xabc", "2", payload)
	if err != nil {
		t.Fatalf("different token_id insert: %v", err)
	}
	if !isNew3 {
		t.Fatal("a different token_id sharing the same tx_hash must be treated as new")
	}
	if id3 == id1 {
		t.Fatal("expected a distinct id for a distinct token_id")
	}
}

func TestListUnenqueuedWebhookOutbox_ReturnsOnlyUnmarked(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	id1, _, err := repo.InsertWebhookOutbox(ctx, 31337, "0xabc", "1", []byte(`{}`))
	if err != nil {
		t.Fatalf("insert 1: %v", err)
	}
	if _, _, err := repo.InsertWebhookOutbox(ctx, 31337, "0xdef", "2", []byte(`{}`)); err != nil {
		t.Fatalf("insert 2: %v", err)
	}

	if err := repo.MarkWebhookOutboxEnqueued(ctx, id1); err != nil {
		t.Fatalf("mark enqueued: %v", err)
	}

	entries, err := repo.ListUnenqueuedWebhookOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("list unenqueued: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d unenqueued entries, want 1 (id1 was marked enqueued)", len(entries))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./services/indexer/internal/adapter/postgres/... -run 'TestInsertWebhookOutbox|TestListUnenqueuedWebhookOutbox' -v`
Expected: FAIL — compile error, `repo.InsertWebhookOutbox undefined` (method doesn't exist yet).

- [ ] **Step 3: Add the port interface methods and type**

Edit `services/indexer/internal/usecase/ports.go` — add after the `CertificatePage`/`ListParams` types (around line 15, before the `CertificateRepo` interface):

```go
// WebhookOutboxEntry is one durable "we owe a webhook" record, read back for delivery.
type WebhookOutboxEntry struct {
	ID      int64
	Payload []byte
}
```

Then add three methods to the `CertificateRepo` interface (after `GetIssuanceTrend`, before the closing `}` at line 53):

```go

	// InsertWebhookOutbox durably records that chainID/txHash/tokenID owes a webhook
	// delivery, deduped by that triple — the identity of the on-chain event itself, stable
	// across a reorg re-mining the same transaction into a different block (unlike
	// log_index, which is block-relative). isNew is true iff this call created the row;
	// false means a webhook was already recorded for this exact event and the caller must
	// not enqueue again. id is only meaningful when isNew is true.
	InsertWebhookOutbox(ctx context.Context, chainID int64, txHash, tokenID string, payload []byte) (id int64, isNew bool, err error)

	// ListUnenqueuedWebhookOutbox returns up to limit outbox rows not yet marked enqueued,
	// oldest first — used by both the fast-path enqueue and the webhook:sweep backstop.
	ListUnenqueuedWebhookOutbox(ctx context.Context, limit int) ([]WebhookOutboxEntry, error)

	// MarkWebhookOutboxEnqueued marks an outbox row as handed to the task queue.
	MarkWebhookOutboxEnqueued(ctx context.Context, id int64) error
```

- [ ] **Step 4: Add the SQL queries**

Edit `services/indexer/internal/db/queries.sql` — append at the end of the file:

```sql

-- InsertWebhookOutbox durably records a certificate-issued event as owing a webhook
-- delivery, deduped by (chain_id, tx_hash, token_id) -- ON CONFLICT DO NOTHING means a
-- duplicate call returns zero rows (:one then surfaces pgx.ErrNoRows, which the repo
-- method translates to isNew=false, not an error).
-- name: InsertWebhookOutbox :one
INSERT INTO webhook_outbox (chain_id, tx_hash, token_id, payload)
VALUES ($1, $2, $3, $4)
ON CONFLICT (chain_id, tx_hash, token_id) DO NOTHING
RETURNING id;

-- ListUnenqueuedWebhookOutbox returns outbox rows not yet handed to the task queue,
-- oldest first -- used by both the fast-path enqueue and the webhook:sweep backstop.
-- name: ListUnenqueuedWebhookOutbox :many
SELECT id, payload FROM webhook_outbox WHERE enqueued_at IS NULL ORDER BY id LIMIT $1;

-- MarkWebhookOutboxEnqueued marks an outbox row as handed to the task queue.
-- name: MarkWebhookOutboxEnqueued :exec
UPDATE webhook_outbox SET enqueued_at = now() WHERE id = $1;
```

- [ ] **Step 5: Regenerate sqlc**

Run: `cd services/indexer && sqlc generate`
Expected: exits 0. `git diff --stat services/indexer/internal/db/queries.sql.go` shows new generated code for `InsertWebhookOutbox`, `ListUnenqueuedWebhookOutbox`, `MarkWebhookOutboxEnqueued`.

Verify the generated `Payload` field type:
```bash
grep -n "Payload" services/indexer/internal/db/queries.sql.go
```
Expected: `Payload []byte` on both `InsertWebhookOutboxParams` and `ListUnenqueuedWebhookOutboxRow`. If sqlc instead generated a different type for the `jsonb` column (e.g. a wrapped type), adjust the repo method in Step 6 to match — do not force a cast that doesn't compile.

- [ ] **Step 6: Implement the repo methods**

Edit `services/indexer/internal/adapter/postgres/certificate_repo.go` — add after `GetIssuanceTrend` (after line 164, before the `trendRow` type block):

```go

// InsertWebhookOutbox durably records chainID/txHash/tokenID as owing a webhook delivery.
// O(1) via the (chain_id, tx_hash, token_id) unique index.
func (r *CertificateRepo) InsertWebhookOutbox(ctx context.Context, chainID int64, txHash, tokenID string, payload []byte) (int64, bool, error) {
	id, err := r.queries.InsertWebhookOutbox(ctx, db.InsertWebhookOutboxParams{
		ChainID: chainID,
		TxHash:  txHash,
		TokenID: tokenID,
		Payload: payload,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil // already recorded for this exact event -- not an error
		}
		return 0, false, fmt.Errorf("postgres.CertificateRepo.InsertWebhookOutbox: %w", err)
	}
	return id, true, nil
}

// ListUnenqueuedWebhookOutbox returns up to limit outbox rows not yet enqueued.
// O(limit) via idx_webhook_outbox_unenqueued.
func (r *CertificateRepo) ListUnenqueuedWebhookOutbox(ctx context.Context, limit int) ([]usecase.WebhookOutboxEntry, error) {
	rows, err := r.queries.ListUnenqueuedWebhookOutbox(ctx, int32(limit)) //nolint:gosec // limit is a small internal constant
	if err != nil {
		return nil, fmt.Errorf("postgres.CertificateRepo.ListUnenqueuedWebhookOutbox: %w", err)
	}
	entries := make([]usecase.WebhookOutboxEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, usecase.WebhookOutboxEntry{ID: row.ID, Payload: row.Payload})
	}
	return entries, nil
}

// MarkWebhookOutboxEnqueued marks an outbox row as handed to the task queue.
func (r *CertificateRepo) MarkWebhookOutboxEnqueued(ctx context.Context, id int64) error {
	if err := r.queries.MarkWebhookOutboxEnqueued(ctx, id); err != nil {
		return fmt.Errorf("postgres.CertificateRepo.MarkWebhookOutboxEnqueued: %w", err)
	}
	return nil
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go build ./... && go test ./services/indexer/internal/adapter/postgres/... -v -run 'TestInsertWebhookOutbox|TestListUnenqueuedWebhookOutbox'`
Expected: PASS. Also run the FULL postgres package suite as a regression check (this file's other tests must stay green): `go test ./services/indexer/internal/adapter/postgres/... -v`

- [ ] **Step 8: Commit**

```bash
git add services/indexer/internal/usecase/ports.go services/indexer/internal/db/queries.sql \
        services/indexer/internal/db/queries.sql.go \
        services/indexer/internal/adapter/postgres/certificate_repo.go \
        services/indexer/internal/adapter/postgres/certificate_repo_test.go
git commit -m "feat(indexer): CertificateRepo webhook outbox methods (insert/list/mark)"
```

---

## Task 4: `Worker.processLog()` webhook outbox flow

**TDD: yes** (the fatal-on-outbox-error control-flow change and the dedup-under-replay behavior are both correctness-critical and easy to get subtly wrong — must be red-green tested, not eyeballed).

**Files:**
- Create: `services/indexer/internal/usecase/webhook_event.go`
- Modify: `services/indexer/internal/usecase/worker.go`
- Modify: `services/indexer/internal/usecase/worker_test.go`

**Interfaces:**
- Consumes: Task 2's `TaskEnqueuer.EnqueueUnique(ctx, taskType, taskID string, payload []byte) error`; Task 3's `CertificateRepo.InsertWebhookOutbox`/`MarkWebhookOutboxEnqueued`.
- Produces: `usecase.WebhookDeliverTaskType = "webhook:deliver"`; `usecase.WebhookSweepTaskType = "webhook:sweep"` (consumed by Task 5); `usecase.NewWebhookEvent(c domain.Certificate) ([]byte, error)`.

- [ ] **Step 1: Write the failing tests**

Add to `services/indexer/internal/usecase/worker_test.go` — first, extend `fakeRepo` to satisfy the growing `CertificateRepo` interface. Replace the `fakeRepo` struct definition (lines 52–55):

```go
// fakeRepo implements usecase.CertificateRepo for tests.
type fakeRepo struct {
	certs     map[string]domain.Certificate
	state     domain.IndexerState
	outbox    map[string]int64 // "txHash:tokenID" -> assigned id
	outboxSeq int64
}
```

Add these methods after the existing `GetIssuanceTrend` stub (after line 96):

```go

// InsertWebhookOutbox fakes the (chain_id, tx_hash, token_id) dedup for tests.
func (r *fakeRepo) InsertWebhookOutbox(_ context.Context, _ int64, txHash, tokenID string, _ []byte) (int64, bool, error) {
	if r.outbox == nil {
		r.outbox = make(map[string]int64)
	}
	key := txHash + ":" + tokenID
	if _, exists := r.outbox[key]; exists {
		return 0, false, nil
	}
	r.outboxSeq++
	r.outbox[key] = r.outboxSeq
	return r.outboxSeq, true, nil
}

// ListUnenqueuedWebhookOutbox is unused by worker tests; stubbed only to satisfy the port.
func (r *fakeRepo) ListUnenqueuedWebhookOutbox(_ context.Context, _ int) ([]usecase.WebhookOutboxEntry, error) {
	return nil, nil
}

// MarkWebhookOutboxEnqueued is unused by worker tests; stubbed only to satisfy the port.
func (r *fakeRepo) MarkWebhookOutboxEnqueued(_ context.Context, _ int64) error {
	return nil
}
```

Now update the existing `fakeEnqueuer` usage in `TestWorker_EnqueuesTrendRefresh_OnSuccessfulUpsert` (around line 344) — the assertion `len(enq.enqueued) != 1` will break once webhook enqueue also fires. Replace the whole test body:

```go
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
	if countTaskType(enq.enqueued, usecase.TrendRefreshTaskType) != 1 {
		t.Fatalf("want 1 enqueue of %q, got %v", usecase.TrendRefreshTaskType, enq.enqueued)
	}
}

// countTaskType counts occurrences of taskType in enqueued -- Task 4 adds a second
// enqueue call (webhook:deliver) alongside trend:refresh, so exact-length assertions on
// the whole slice are no longer meaningful; count the specific type instead.
func countTaskType(enqueued []string, taskType string) int {
	n := 0
	for _, t := range enqueued {
		if t == taskType {
			n++
		}
	}
	return n
}
```

Now add three new tests after it (still in `worker_test.go`):

```go

func TestWorker_EnqueuesWebhookDeliver_OnNewCertificate(t *testing.T) {
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
	if countTaskType(enq.enqueued, usecase.WebhookDeliverTaskType) != 1 {
		t.Fatalf("want 1 enqueue of %q, got %v", usecase.WebhookDeliverTaskType, enq.enqueued)
	}
}

func TestWorker_DoesNotReenqueueWebhook_OnIdempotentReplay(t *testing.T) {
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
		t.Fatalf("first poll: %v", err)
	}
	if got := countTaskType(enq.enqueued, usecase.WebhookDeliverTaskType); got != 1 {
		t.Fatalf("want 1 webhook enqueue after first poll, got %d", got)
	}

	// Reset the checkpoint (simulating a reorg replay of the same certificate) but keep
	// the SAME repo/outbox map -- the outbox dedup must prevent a second webhook enqueue.
	repo.state = domain.IndexerState{}
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("second (replay) poll: %v", err)
	}
	if got := countTaskType(enq.enqueued, usecase.WebhookDeliverTaskType); got != 1 {
		t.Fatalf("want still only 1 webhook enqueue after replay (outbox dedup), got %d", got)
	}
}

// fakeFailingOutboxRepo wraps fakeRepo but makes InsertWebhookOutbox always fail --
// verifies processLog propagates the error (fatal) rather than swallowing it.
type fakeFailingOutboxRepo struct {
	*fakeRepo
}

func (r *fakeFailingOutboxRepo) InsertWebhookOutbox(_ context.Context, _ int64, _, _ string, _ []byte) (int64, bool, error) {
	return 0, false, errors.New("fake: outbox insert failed")
}

func TestWorker_InsertWebhookOutboxError_IsFatal_StateNotAdvanced(t *testing.T) {
	repo := &fakeFailingOutboxRepo{fakeRepo: newFakeRepo()}
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	w := newWorker(src, repo)
	w.SetEnqueuer(&fakeEnqueuer{})

	err := w.Poll(t.Context())
	if err == nil {
		t.Fatal("want an error when InsertWebhookOutbox fails")
	}
	if repo.state.LastProcessedBlock != 0 {
		t.Fatalf("state must not advance when outbox insert fails, got %d", repo.state.LastProcessedBlock)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./services/indexer/internal/usecase/... -v -run TestWorker`
Expected: FAIL — compile errors (`usecase.WebhookDeliverTaskType undefined`, `fakeRepo does not implement usecase.CertificateRepo` until Step 3/4 land).

- [ ] **Step 3: Create the webhook event type + task-type consts**

Create `services/indexer/internal/usecase/webhook_event.go`:

```go
package usecase

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
)

// WebhookDeliverTaskType identifies the asynq task that delivers one signed webhook.
// Defined here (not in the adapter) so Worker can reference it without importing adapter
// code. NOTE: services/notify cannot import this constant directly -- Go's internal/
// package visibility rules mean only code under services/indexer/ may import
// services/indexer/internal/usecase. notify carries its own copy of this exact string;
// the two are two independent deployables that only agree on the wire-level string.
const WebhookDeliverTaskType = "webhook:deliver"

// WebhookSweepTaskType identifies the asynq task (Task 5) that scans webhook_outbox for
// rows not yet enqueued and retries them -- the durable backstop for a missed enqueue.
const WebhookSweepTaskType = "webhook:sweep"

// webhookCertificateIssuedEvent is the "event" field value for every webhook payload this
// codebase currently emits -- a single fixed value, not yet an enum, since there is only
// one event type today.
const webhookCertificateIssuedEvent = "certificate.issued"

// WebhookEvent is the JSON envelope delivered to the configured webhook consumer when a
// certificate is issued. Field names/shape are a stable external contract -- changing them
// is a breaking change for any real consumer.
type WebhookEvent struct {
	Event string           `json:"event"`
	Data  WebhookEventData `json:"data"`
}

// WebhookEventData is the certificate payload nested inside WebhookEvent.
type WebhookEventData struct {
	TokenID       string `json:"tokenId"`
	OwnerAddress  string `json:"ownerAddress"`
	Title         string `json:"title"`
	RecipientName string `json:"recipientName"`
	IssuerName    string `json:"issuerName"`
	Description   string `json:"description"`
	IssuedAt      string `json:"issuedAt"` // RFC3339
	ChainID       int64  `json:"chainId"`
	TxHash        string `json:"txHash"`
}

// NewWebhookEvent builds the webhook JSON payload for a newly indexed certificate.
// Pure function -- O(1).
func NewWebhookEvent(c domain.Certificate) ([]byte, error) {
	evt := WebhookEvent{
		Event: webhookCertificateIssuedEvent,
		Data: WebhookEventData{
			TokenID:       c.TokenID,
			OwnerAddress:  c.Owner.String(),
			Title:         c.Title,
			RecipientName: c.RecipientName,
			IssuerName:    c.IssuerName,
			Description:   c.Description,
			IssuedAt:      c.IssuedAt.UTC().Format(time.RFC3339),
			ChainID:       c.ChainID,
			TxHash:        c.TxHash,
		},
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("usecase.NewWebhookEvent: %w", err)
	}
	return payload, nil
}
```

- [ ] **Step 4: Wire the webhook flow into `processLog`**

Edit `services/indexer/internal/usecase/worker.go` — replace `processLog` (lines 213–237) in full:

```go
// processLog backfills full cert data for a single event log and upserts it.
func (w *Worker) processLog(ctx context.Context, l domain.IssuedLog) error {
	data, err := w.src.GetCertificate(ctx, l.TokenID)
	if err != nil {
		return fmt.Errorf("get certificate %s: %w", l.TokenID, err)
	}
	cert, err := domain.NewIndexedCertificate(l, data, w.cfg.ChainID)
	if err != nil {
		return fmt.Errorf("build certificate %s: %w", l.TokenID, err)
	}
	if err := w.repo.Upsert(ctx, cert); err != nil {
		return fmt.Errorf("upsert %s: %w", l.TokenID, err)
	}
	if w.pub != nil {
		w.pub.Publish(cert)
	}
	if w.enqueuer != nil {
		if err := w.enqueuer.EnqueueUnique(ctx, TrendRefreshTaskType, TrendRefreshTaskType, nil); err != nil {
			// Non-fatal: ingest correctness never depends on the cache-refresh job succeeding —
			// a failed enqueue just means the trend cache stays stale until the cron backstop runs.
			w.log.Warn("enqueue trend refresh", "err", err)
		}
		if err := w.enqueueWebhook(ctx, cert); err != nil {
			return fmt.Errorf("webhook outbox %s: %w", l.TokenID, err)
		}
	}
	return nil
}

// enqueueWebhook durably records that cert owes a webhook delivery, then best-effort
// enqueues it. InsertWebhookOutbox failing is FATAL -- it propagates so poll() does not
// advance the checkpoint and retries the whole batch next cycle (both Upsert and
// InsertWebhookOutbox are idempotent under retry via their own ON CONFLICT clauses). The
// enqueue-to-asynq step below stays best-effort/logged: webhook:sweep (Task 5) is its
// backstop.
func (w *Worker) enqueueWebhook(ctx context.Context, cert domain.Certificate) error {
	payload, err := NewWebhookEvent(cert)
	if err != nil {
		return err
	}
	id, isNew, err := w.repo.InsertWebhookOutbox(ctx, w.cfg.ChainID, cert.TxHash, cert.TokenID, payload)
	if err != nil {
		return err
	}
	if !isNew {
		return nil // already recorded for this exact on-chain event -- nothing to do
	}
	taskID := fmt.Sprintf("%s:%d", WebhookDeliverTaskType, id)
	if err := w.enqueuer.EnqueueUnique(ctx, WebhookDeliverTaskType, taskID, payload); err != nil {
		// A failed enqueue here is fully recovered by webhook:sweep, since the outbox row
		// stays enqueued_at IS NULL until something successfully enqueues it.
		w.log.Warn("enqueue webhook deliver", "err", err)
		return nil
	}
	if err := w.repo.MarkWebhookOutboxEnqueued(ctx, id); err != nil {
		// webhook: enqueue succeeded but marking it failed -- realistic trigger is
		// shutdown/context cancellation (worker runs on a cancelable ctx; see main.go's
		// shutdown sequencing), not just a generic DB hiccup. The row stays unenqueued
		// and webhook:sweep will retry it. Full elimination needs 2PC between Postgres
		// and Redis, disproportionate at this project's stage -- accepted, bounded
		// tradeoff: see design doc §2.
		w.log.Warn("mark webhook outbox enqueued", "err", err)
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go build ./... && go test ./services/indexer/internal/usecase/... -v -run TestWorker`
Expected: PASS — all Worker tests green, including the 3 new tests. This must include the FULL existing suite (12+ tests as of Phase 6) as a regression check, since this is the 4th extension of `processLog`/`worker.go` across the project's history.

- [ ] **Step 6: Commit**

```bash
git add services/indexer/internal/usecase/webhook_event.go \
        services/indexer/internal/usecase/worker.go \
        services/indexer/internal/usecase/worker_test.go
git commit -m "feat(indexer): Worker enqueues reorg-safe webhook outbox on certificate issuance"
```

---

## Task 5: `webhook:sweep` handler + composition-root wiring

**TDD: no** (thin wiring over `ListUnenqueuedWebhookOutbox` + `EnqueueUnique`, both already tested in Tasks 2-3), but add tests verifying the re-enqueue and one-bad-row-doesn't-block-the-rest behavior.

**Files:**
- Create: `services/indexer/internal/adapter/asynqjobs/webhook_sweep.go`
- Test: `services/indexer/internal/adapter/asynqjobs/webhook_sweep_test.go`
- Modify: `services/indexer/cmd/indexer/main.go`

**Interfaces:**
- Consumes: Task 3's `CertificateRepo.ListUnenqueuedWebhookOutbox`/`MarkWebhookOutboxEnqueued`; Task 2's `TaskEnqueuer.EnqueueUnique`; Task 4's `usecase.WebhookSweepTaskType`/`usecase.WebhookDeliverTaskType`.
- Produces: `asynqjobs.NewWebhookSweepTask() *asynq.Task`; `asynqjobs.NewWebhookSweepHandler(repo, enqueuer, log) *WebhookSweepHandler` (satisfies `asynq.Handler`).

- [ ] **Step 1: Create the handler**

Create `services/indexer/internal/adapter/asynqjobs/webhook_sweep.go`:

```go
package asynqjobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// sweepLimit bounds how many stale outbox rows one sweep run re-attempts -- keeps each
// sweep O(sweepLimit), not O(size of the whole outbox table).
const sweepLimit = 100

// webhookOutboxLister is the subset of usecase.CertificateRepo this handler needs -- a
// narrow interface makes it testable without a real Postgres-backed repo.
type webhookOutboxLister interface {
	ListUnenqueuedWebhookOutbox(ctx context.Context, limit int) ([]usecase.WebhookOutboxEntry, error)
	MarkWebhookOutboxEnqueued(ctx context.Context, id int64) error
}

// NewWebhookSweepTask builds the (payload-less) sweep trigger task -- the scan itself
// needs no parameters, every enqueue of this type is identical.
func NewWebhookSweepTask() *asynq.Task {
	return asynq.NewTask(usecase.WebhookSweepTaskType, nil, asynq.MaxRetry(2))
}

// WebhookSweepHandler re-attempts enqueueing any outbox row not yet handed to the task
// queue -- the durable backstop for Worker.processLog's best-effort fast-path enqueue.
// Registered against usecase.WebhookSweepTaskType in the indexer's asynq ServeMux.
type WebhookSweepHandler struct {
	repo     webhookOutboxLister
	enqueuer usecase.TaskEnqueuer
	log      *slog.Logger
}

// NewWebhookSweepHandler constructs a WebhookSweepHandler.
func NewWebhookSweepHandler(repo webhookOutboxLister, enqueuer usecase.TaskEnqueuer, log *slog.Logger) *WebhookSweepHandler {
	if log == nil {
		log = slog.Default()
	}
	return &WebhookSweepHandler{repo: repo, enqueuer: enqueuer, log: log}
}

// ProcessTask satisfies asynq.Handler. O(sweepLimit) per run, not proportional to the
// certificate count or the outbox table's total size (idx_webhook_outbox_unenqueued keeps
// the underlying query itself O(log k) too).
func (h *WebhookSweepHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	entries, err := h.repo.ListUnenqueuedWebhookOutbox(ctx, sweepLimit)
	if err != nil {
		return fmt.Errorf("list unenqueued webhook outbox: %w", err)
	}
	for _, e := range entries {
		taskID := fmt.Sprintf("%s:%d", usecase.WebhookDeliverTaskType, e.ID)
		if err := h.enqueuer.EnqueueUnique(ctx, usecase.WebhookDeliverTaskType, taskID, e.Payload); err != nil {
			h.log.Warn("sweep: enqueue webhook deliver", "id", e.ID, "err", err)
			continue // don't let one bad row block the rest of the sweep
		}
		if err := h.repo.MarkWebhookOutboxEnqueued(ctx, e.ID); err != nil {
			h.log.Warn("sweep: mark webhook outbox enqueued", "id", e.ID, "err", err)
		}
	}
	return nil
}
```

- [ ] **Step 2: Write tests**

Create `services/indexer/internal/adapter/asynqjobs/webhook_sweep_test.go`:

```go
package asynqjobs_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/asynqjobs"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

type fakeOutboxLister struct {
	entries []usecase.WebhookOutboxEntry
	marked  []int64
}

func (f *fakeOutboxLister) ListUnenqueuedWebhookOutbox(_ context.Context, _ int) ([]usecase.WebhookOutboxEntry, error) {
	return f.entries, nil
}

func (f *fakeOutboxLister) MarkWebhookOutboxEnqueued(_ context.Context, id int64) error {
	f.marked = append(f.marked, id)
	return nil
}

type fakeSweepEnqueuer struct {
	enqueuedIDs []string
	failFor     string // taskID that should fail, if set
}

func (f *fakeSweepEnqueuer) EnqueueUnique(_ context.Context, _, taskID string, _ []byte) error {
	if taskID == f.failFor {
		return errors.New("fake: enqueue failed")
	}
	f.enqueuedIDs = append(f.enqueuedIDs, taskID)
	return nil
}

func TestWebhookSweepHandler_ReenqueuesAllUnenqueuedRows(t *testing.T) {
	lister := &fakeOutboxLister{entries: []usecase.WebhookOutboxEntry{
		{ID: 1, Payload: []byte(`{"tokenId":"1"}`)},
		{ID: 2, Payload: []byte(`{"tokenId":"2"}`)},
	}}
	enq := &fakeSweepEnqueuer{}
	h := asynqjobs.NewWebhookSweepHandler(lister, enq, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := h.ProcessTask(context.Background(), asynqjobs.NewWebhookSweepTask()); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}
	if len(enq.enqueuedIDs) != 2 {
		t.Fatalf("got %d enqueues, want 2", len(enq.enqueuedIDs))
	}
	if len(lister.marked) != 2 {
		t.Fatalf("got %d marked-enqueued calls, want 2", len(lister.marked))
	}
}

func TestWebhookSweepHandler_OneRowFailing_DoesNotBlockTheRest(t *testing.T) {
	lister := &fakeOutboxLister{entries: []usecase.WebhookOutboxEntry{
		{ID: 1, Payload: []byte(`{}`)},
		{ID: 2, Payload: []byte(`{}`)},
	}}
	enq := &fakeSweepEnqueuer{failFor: "webhook:deliver:1"}
	h := asynqjobs.NewWebhookSweepHandler(lister, enq, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := h.ProcessTask(context.Background(), asynqjobs.NewWebhookSweepTask()); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}
	if len(enq.enqueuedIDs) != 1 || enq.enqueuedIDs[0] != "webhook:deliver:2" {
		t.Fatalf("want only id=2 enqueued after id=1 fails, got %v", enq.enqueuedIDs)
	}
	if len(lister.marked) != 1 || lister.marked[0] != 2 {
		t.Fatalf("want only id=2 marked, got %v", lister.marked)
	}
}

var _ asynq.Handler = (*asynqjobs.WebhookSweepHandler)(nil)
```

- [ ] **Step 3: Run tests**

Run: `go build ./... && go test ./services/indexer/internal/adapter/asynqjobs/... -v`
Expected: PASS — all tests green, including the 2 new sweep tests plus the existing trend-refresh and enqueuer tests (regression).

- [ ] **Step 4: Wire the sweep handler + cron into the composition root**

Edit `services/indexer/cmd/indexer/main.go`:

First, change the `enqueuer` construction (around line 76-77) to keep a named reference instead of an inline call:

```go
	asynqRedisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}
	asynqClient := asynq.NewClient(asynqRedisOpt)
	defer asynqClient.Close() //nolint:errcheck // best-effort close on process exit

	enqueuer := asynqjobs.NewEnqueuer(asynqClient)
	worker.SetEnqueuer(enqueuer)

	s := buildGRPCServer(repo, src, broadcaster, trendService, log)

	asynqServer, asynqMux, scheduler := buildAsynqRuntime(asynqRedisOpt, trendService, repo, enqueuer, log)
```

(This replaces the previous `worker.SetEnqueuer(asynqjobs.NewEnqueuer(asynqClient))` and `asynqServer, asynqMux, scheduler := buildAsynqRuntime(asynqRedisOpt, trendService, log)` lines with the above.)

Then replace `buildAsynqRuntime` in full:

```go
// buildAsynqRuntime wires the asynq processing server (handles enqueued refresh/sweep
// tasks) and scheduler (cron backstops: 15-min trend-refresh, 5-min webhook-outbox sweep).
// Returns the mux alongside the server since Run(mux) needs the exact same instance the
// handler was registered on.
func buildAsynqRuntime(redisOpt asynq.RedisClientOpt, trend *usecase.TrendService, repo usecase.CertificateRepo, enqueuer usecase.TaskEnqueuer, log *slog.Logger) (*asynq.Server, *asynq.ServeMux, *asynq.Scheduler) {
	server := asynq.NewServer(redisOpt, asynq.Config{Concurrency: 5})

	mux := asynq.NewServeMux()
	mux.Handle(usecase.TrendRefreshTaskType, asynqjobs.NewRefreshTrendCacheHandler(trend, log))
	mux.Handle(usecase.WebhookSweepTaskType, asynqjobs.NewWebhookSweepHandler(repo, enqueuer, log))

	scheduler := asynq.NewScheduler(redisOpt, nil)
	if _, err := scheduler.Register("*/15 * * * *", asynqjobs.NewRefreshTrendCacheTask()); err != nil {
		log.Error("register trend-refresh cron", "err", err)
	}
	if _, err := scheduler.Register("*/5 * * * *", asynqjobs.NewWebhookSweepTask()); err != nil {
		log.Error("register webhook-sweep cron", "err", err)
	}

	return server, mux, scheduler
}
```

- [ ] **Step 5: Full build + test regression**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./services/indexer/... -race -cover`
Expected: clean build, no vet/gofmt output, every package green.

- [ ] **Step 6: Commit**

```bash
git add services/indexer/internal/adapter/asynqjobs/webhook_sweep.go \
        services/indexer/internal/adapter/asynqjobs/webhook_sweep_test.go \
        services/indexer/cmd/indexer/main.go
git commit -m "feat(indexer): webhook:sweep handler + cron backstop wired into composition root"
```

---

## Task 6: notify service — `SignPayload` (pure HMAC signing)

**TDD: yes** (pure function, HMAC correctness is easy to get subtly wrong — wrong encoding, wrong header format — and cheap to verify with a failing test first).

**Files:**
- Create: `services/notify/internal/usecase/sign.go`
- Test: `services/notify/internal/usecase/sign_test.go`

**Interfaces:**
- Produces: `usecase.SignPayload(secret string, body []byte) string` — hex-encoded HMAC-SHA256, consumed by Task 7's webhook handler.

- [ ] **Step 1: Write the failing tests**

Create `services/notify/internal/usecase/sign_test.go`:

```go
package usecase_test

import (
	"testing"

	"github.com/oksasatya/skillpass/services/notify/internal/usecase"
)

func TestSignPayload_MatchesKnownHMAC(t *testing.T) {
	// Ground truth computed independently:
	//   printf '%s' '{"event":"certificate.issued"}' | openssl dgst -sha256 -hmac "test-secret"
	got := usecase.SignPayload("test-secret", []byte(`{"event":"certificate.issued"}`))
	want := "9eafe280c2c64ecd9cb6342b5e415dfde6aa92b787bcaab89440b1f9bc2d532f"
	if got != want {
		t.Fatalf("SignPayload = %q, want %q", got, want)
	}
}

func TestSignPayload_DifferentSecrets_ProduceDifferentSignatures(t *testing.T) {
	body := []byte(`{"event":"certificate.issued"}`)
	sig1 := usecase.SignPayload("secret-a", body)
	sig2 := usecase.SignPayload("secret-b", body)
	if sig1 == sig2 {
		t.Fatal("different secrets must produce different signatures")
	}
}

func TestSignPayload_DifferentBodies_ProduceDifferentSignatures(t *testing.T) {
	sig1 := usecase.SignPayload("secret", []byte(`{"a":1}`))
	sig2 := usecase.SignPayload("secret", []byte(`{"a":2}`))
	if sig1 == sig2 {
		t.Fatal("different bodies must produce different signatures")
	}
}

func TestSignPayload_SameInput_IsDeterministic(t *testing.T) {
	body := []byte(`{"event":"certificate.issued"}`)
	sig1 := usecase.SignPayload("secret", body)
	sig2 := usecase.SignPayload("secret", body)
	if sig1 != sig2 {
		t.Fatal("same secret+body must always produce the same signature")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./services/notify/... -v`
Expected: FAIL — `no Go files in services/notify/internal/usecase` or `usecase.SignPayload undefined` (package doesn't exist yet).

- [ ] **Step 3: Implement `SignPayload`**

Create `services/notify/internal/usecase/sign.go`:

```go
// Package usecase holds notify's pure business logic -- signing webhook payloads. It
// imports nothing from adapter or third-party HTTP/asynq packages (hexagonal-lite).
package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// SignPayload returns the hex-encoded HMAC-SHA256 of body using secret -- the value sent
// in the X-SkillPass-Signature header (as "sha256=<hex>"), mirroring the GitHub/Stripe
// webhook-signing convention. Pure function -- O(len(body)).
func SignPayload(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body) //nolint:errcheck // hash.Hash.Write never returns an error
	return hex.EncodeToString(mac.Sum(nil))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./services/notify/... -v`
Expected: PASS — all 4 tests green.

- [ ] **Step 5: Commit**

```bash
git add services/notify/internal/usecase/sign.go services/notify/internal/usecase/sign_test.go
git commit -m "feat(notify): pure HMAC-SHA256 payload signing"
```

---

## Task 7: notify service — webhook delivery handler + config + composition root

**TDD: no** (thin wiring over Task 6's `SignPayload` + `net/http`), but add `httptest.Server`-based tests covering happy path, non-2xx, and unreachable-endpoint.

**Files:**
- Create: `services/notify/internal/config/config.go`
- Create: `services/notify/internal/adapter/webhook/handler.go`
- Test: `services/notify/internal/adapter/webhook/handler_test.go`
- Create: `services/notify/cmd/notify/main.go`
- Create: `services/notify/Dockerfile`

**Interfaces:**
- Consumes: Task 6's `usecase.SignPayload`.
- Produces: `webhook.DeliverTaskType = "webhook:deliver"` (must match `usecase.WebhookDeliverTaskType` from Task 4 — see the comment in Task 4's `webhook_event.go` on why this is a separate literal, not a shared import); `webhook.NewHandler(url, secret string) *Handler` (satisfies `asynq.Handler`).

- [ ] **Step 1: Config**

Create `services/notify/internal/config/config.go`:

```go
// Package config loads notify configuration from environment variables with fail-fast validation.
package config

import (
	"fmt"
	"os"
)

// Config holds all tunable parameters for the notify service.
type Config struct {
	RedisAddr     string
	WebhookURL    string
	WebhookSecret string
	HTTPAddr      string
}

// Load reads configuration from environment variables and returns a Config or an error
// naming the first missing required variable.
func Load() (Config, error) {
	var cfg Config
	var err error

	if cfg.RedisAddr, err = mustenv("REDIS_ADDR"); err != nil {
		return Config{}, err
	}
	if cfg.WebhookURL, err = mustenv("WEBHOOK_URL"); err != nil {
		return Config{}, err
	}
	if cfg.WebhookSecret, err = mustenv("WEBHOOK_SECRET"); err != nil {
		return Config{}, err
	}
	cfg.HTTPAddr = getenv("HTTP_ADDR", ":8090")

	return cfg, nil
}

// mustenv returns the value of an env var or an error if it is empty / unset.
func mustenv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required env var %s is not set", key)
	}
	return v, nil
}

// getenv returns the env var value or a default when the var is empty / unset.
func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
```

- [ ] **Step 2: Webhook delivery handler + tests**

Create `services/notify/internal/adapter/webhook/handler.go`:

```go
// Package webhook implements the asynq handler that signs and delivers webhook payloads
// produced by the indexer's webhook:deliver task.
package webhook

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/notify/internal/usecase"
)

// httpTimeout bounds a single delivery attempt -- asynq's own retry/backoff handles
// retrying a slow or unreachable endpoint across separate attempts.
const httpTimeout = 10 * time.Second

// DeliverTaskType must match usecase.WebhookDeliverTaskType on the indexer side --
// duplicated here (not imported) because Go's internal/ package visibility means notify
// cannot import services/indexer/internal/usecase at all; the two deployables only agree
// on this wire-level string.
const DeliverTaskType = "webhook:deliver"

// Handler is an asynq.Handler that signs and POSTs the task's payload to the configured
// webhook URL.
type Handler struct {
	url    string
	secret string
	client *http.Client
}

// NewHandler constructs a Handler.
func NewHandler(url, secret string) *Handler {
	return &Handler{url: url, secret: secret, client: &http.Client{Timeout: httpTimeout}}
}

// ProcessTask satisfies asynq.Handler. A non-2xx response or request error returns an
// error, which asynq retries per its own MaxRetry/backoff configuration on the task.
func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	body := t.Payload()
	sig := usecase.SignPayload(h.secret, body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SkillPass-Signature", "sha256="+sig)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("deliver webhook: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable here

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook delivery failed: status %d", resp.StatusCode)
	}
	return nil
}
```

Create `services/notify/internal/adapter/webhook/handler_test.go`:

```go
package webhook_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/notify/internal/adapter/webhook"
	"github.com/oksasatya/skillpass/services/notify/internal/usecase"
)

func TestHandler_ProcessTask_Success(t *testing.T) {
	var gotBody []byte
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotSig = r.Header.Get("X-SkillPass-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	h := webhook.NewHandler(srv.URL, "test-secret")
	payload := []byte(`{"event":"certificate.issued","data":{"tokenId":"1"}}`)
	task := asynq.NewTask(webhook.DeliverTaskType, payload)

	if err := h.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}
	if string(gotBody) != string(payload) {
		t.Errorf("body = %s, want %s", gotBody, payload)
	}
	wantSig := "sha256=" + usecase.SignPayload("test-secret", payload)
	if gotSig != wantSig {
		t.Errorf("signature header = %q, want %q", gotSig, wantSig)
	}
}

func TestHandler_ProcessTask_NonSuccessStatus_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	h := webhook.NewHandler(srv.URL, "test-secret")
	task := asynq.NewTask(webhook.DeliverTaskType, []byte(`{}`))

	if err := h.ProcessTask(context.Background(), task); err == nil {
		t.Fatal("want an error on a 500 response (asynq must retry)")
	}
}

func TestHandler_ProcessTask_UnreachableURL_ReturnsError(t *testing.T) {
	h := webhook.NewHandler("http://127.0.0.1:1", "test-secret") // nothing listens on port 1
	task := asynq.NewTask(webhook.DeliverTaskType, []byte(`{}`))

	if err := h.ProcessTask(context.Background(), task); err == nil {
		t.Fatal("want an error when the endpoint is unreachable")
	}
}

var _ asynq.Handler = (*webhook.Handler)(nil)
```

- [ ] **Step 3: Run tests**

Run: `go build ./... && go test ./services/notify/... -v`
Expected: PASS — all tests green.

- [ ] **Step 4: Composition root**

Create `services/notify/cmd/notify/main.go`:

```go
// Command notify is the SkillPass webhook delivery service. It consumes webhook:deliver
// tasks from the shared Redis/asynq queue and POSTs signed payloads to one configured
// webhook URL. It never touches Postgres -- the indexer owns all durable webhook state
// (the outbox table + the sweep backstop).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"golang.org/x/sync/errgroup"

	"github.com/oksasatya/skillpass/services/notify/internal/adapter/webhook"
	"github.com/oksasatya/skillpass/services/notify/internal/config"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}
	server := asynq.NewServer(redisOpt, asynq.Config{Concurrency: 5})

	mux := asynq.NewServeMux()
	mux.Handle(webhook.DeliverTaskType, webhook.NewHandler(cfg.WebhookURL, cfg.WebhookSecret))

	healthSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: healthzMux()}

	if err := runConcurrently(ctx, server, mux, healthSrv, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// healthzMux serves a minimal liveness endpoint -- notify has no other public HTTP API.
func healthzMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

// runConcurrently starts the asynq consumer and the /healthz server; shuts both down
// gracefully on ctx cancellation.
func runConcurrently(ctx context.Context, server *asynq.Server, mux *asynq.ServeMux, healthSrv *http.Server, log *slog.Logger) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return server.Run(mux)
	})

	g.Go(func() error {
		log.Info("healthz server listening", "addr", healthSrv.Addr)
		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		server.Shutdown()
		return healthSrv.Close()
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}
```

- [ ] **Step 5: Dockerfile**

Create `services/notify/Dockerfile`:

```dockerfile
# syntax=docker/dockerfile:1
# Mirrors services/indexer/Dockerfile and services/gateway/Dockerfile: static binary,
# distroless/nonroot runtime.

FROM golang:1.26 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /notify \
    ./services/notify/cmd/notify

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /notify /notify

ENTRYPOINT ["/notify"]
```

- [ ] **Step 6: Build check**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean build, no vet/gofmt output (this confirms `cmd/notify/main.go` compiles against the real `asynq`/`errgroup` packages, not just the unit-tested handler in isolation).

- [ ] **Step 7: Commit**

```bash
git add services/notify/internal/config/config.go \
        services/notify/internal/adapter/webhook/handler.go \
        services/notify/internal/adapter/webhook/handler_test.go \
        services/notify/cmd/notify/main.go \
        services/notify/Dockerfile
git commit -m "feat(notify): webhook delivery handler + composition root + Dockerfile"
```

---

## Task 8: Gateway metadata endpoint

**TDD: no** (thin REST→gRPC reshape, same pattern as the existing `GetCertificate` handler), but add tests: happy path, invalid token ID, not-found — matching `certificates_test.go`'s existing style.

**Files:**
- Create: `services/gateway/internal/httpapi/metadata.go`
- Test: `services/gateway/internal/httpapi/metadata_test.go`
- Modify: `services/gateway/internal/httpapi/router.go`

**Interfaces:**
- Consumes: the existing `Deps.Cert.GetCertificate` gRPC call (no new proto).
- Produces: `httpapi.GetCertificateMetadata(d Deps) http.HandlerFunc`.

- [ ] **Step 1: Handler + DTO**

Create `services/gateway/internal/httpapi/metadata.go`:

```go
package httpapi

import (
	"context"
	"net/http"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// MetadataAttributeDTO is one ERC-721-style attribute entry.
type MetadataAttributeDTO struct {
	TraitType string `json:"trait_type"`
	Value     string `json:"value"`
}

// MetadataDTO is the ERC-721 metadata JSON shape served at GET /certificates/{tokenId}/metadata.
// image is deliberately omitted -- no image asset exists in this project.
type MetadataDTO struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Attributes  []MetadataAttributeDTO `json:"attributes"`
}

// GetCertificateMetadata handles GET /certificates/{tokenId}/metadata -- reshapes the
// existing GetCertificate gRPC response into ERC-721-style JSON so wallets/marketplaces
// can render a certificate. No new gRPC call; zero business logic beyond the reshape.
func GetCertificateMetadata(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenID := r.PathValue("tokenId")
		if !isValidTokenID(tokenID) {
			writeJSONError(w, http.StatusBadRequest, "token_id must be a non-empty decimal digit string")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), d.RequestTimeout)
		defer cancel()

		resp, err := d.Cert.GetCertificate(ctx, &certv1.GetCertificateRequest{TokenId: tokenID})
		if err != nil {
			writeGRPCError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, toMetadataDTO(resp.GetCertificate()))
	}
}

// toMetadataDTO maps a proto Certificate to its ERC-721 metadata JSON representation.
// Pure function -- O(1).
func toMetadataDTO(c *certv1.Certificate) MetadataDTO {
	return MetadataDTO{
		Name:        c.GetTitle(),
		Description: c.GetDescription(),
		Attributes: []MetadataAttributeDTO{
			{TraitType: "Recipient", Value: c.GetRecipientName()},
			{TraitType: "Issuer", Value: c.GetIssuerName()},
			{TraitType: "Issued At", Value: c.GetIssuedAt().AsTime().UTC().Format("2006-01-02T15:04:05Z07:00")},
		},
	}
}
```

- [ ] **Step 2: Tests**

Create `services/gateway/internal/httpapi/metadata_test.go`:

```go
package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

func TestGetCertificateMetadataHandler_InvalidTokenID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/certificates/1abc/metadata", nil)
	req.SetPathValue("tokenId", "1; DROP TABLE certificates;")
	w := httptest.NewRecorder()

	GetCertificateMetadata(newDeps(&fakeCertClient{}))(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetCertificateMetadataHandler_Success(t *testing.T) {
	client := &fakeCertClient{getResp: &certv1.GetCertificateResponse{Certificate: sampleProtoCert("1")}}
	req := httptest.NewRequest(http.MethodGet, "/certificates/1/metadata", nil)
	req.SetPathValue("tokenId", "1")
	w := httptest.NewRecorder()

	GetCertificateMetadata(newDeps(client))(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body MetadataDTO
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "Go Expert" {
		t.Errorf("name = %q, want %q", body.Name, "Go Expert")
	}
	if body.Description != "Backend cert" {
		t.Errorf("description = %q, want %q", body.Description, "Backend cert")
	}
	if len(body.Attributes) != 3 {
		t.Fatalf("got %d attributes, want 3", len(body.Attributes))
	}
	if body.Attributes[0].TraitType != "Recipient" || body.Attributes[0].Value != "Alice" {
		t.Errorf("attributes[0] = %+v, want Recipient=Alice", body.Attributes[0])
	}
	if body.Attributes[1].TraitType != "Issuer" || body.Attributes[1].Value != "Skillpass" {
		t.Errorf("attributes[1] = %+v, want Issuer=Skillpass", body.Attributes[1])
	}
}

func TestGetCertificateMetadataHandler_NotFound(t *testing.T) {
	client := &fakeCertClient{getErr: status.Error(codes.NotFound, "certificate not found")}
	req := httptest.NewRequest(http.MethodGet, "/certificates/999/metadata", nil)
	req.SetPathValue("tokenId", "999")
	w := httptest.NewRecorder()

	GetCertificateMetadata(newDeps(client))(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
```

- [ ] **Step 3: Wire the route**

Edit `services/gateway/internal/httpapi/router.go` — add one line after the existing `GET /certificates/{tokenId}` registration (line 33):

```go
	mux.Handle("GET /certificates/{tokenId}", GetCertificate(d))
	mux.Handle("GET /certificates/{tokenId}/metadata", GetCertificateMetadata(d))
```

- [ ] **Step 4: Run tests**

Run: `go build ./... && go test ./services/gateway/... -v -run 'TestGetCertificateMetadata|TestGetCertificate'`
Expected: PASS — new metadata tests green, existing `GetCertificate` tests unaffected (regression check, Go 1.22+'s ServeMux disambiguates `/certificates/{tokenId}` vs `/certificates/{tokenId}/metadata` by pattern specificity, not registration order).

- [ ] **Step 5: Commit**

```bash
git add services/gateway/internal/httpapi/metadata.go \
        services/gateway/internal/httpapi/metadata_test.go \
        services/gateway/internal/httpapi/router.go
git commit -m "feat(gateway): GET /certificates/{tokenId}/metadata ERC-721 endpoint"
```

---

## Task 9: docker-compose wiring + `seed.sh`/`Makefile` fixes + full smoke test

**TDD: no** (infra + shell script), verified by an end-to-end smoke test that actually captures a delivered webhook, not just a plausible-looking config.

**Files:**
- Modify: `deploy/docker-compose.yml`
- Modify: `deploy/seed.sh`
- Modify: `Makefile`

- [ ] **Step 1: Add the `notify` service to docker-compose**

Edit `deploy/docker-compose.yml` — add a new `notify` service after `indexer` (before `gateway`):

```yaml
  notify:
    build:
      context: ..
      dockerfile: services/notify/Dockerfile
    depends_on:
      redis:
        condition: service_healthy
    environment:
      REDIS_ADDR: "redis:6379"
      WEBHOOK_URL: "http://host.docker.internal:9000/webhook"
      WEBHOOK_SECRET: "dev-secret-change-me"
      HTTP_ADDR: ":8090"
    networks:
      - skillpass
```

- [ ] **Step 2: Update `deploy/seed.sh`** to compute the next token ID and point `metadataURI` at the new gateway endpoint

Replace the entire file `deploy/seed.sh`:

```bash
#!/usr/bin/env bash
# Deploy SkillPassCertificate to local anvil and issue 2 test certificates.
# Run after `make dev-up` from the repo root.
set -euo pipefail

export PATH="$HOME/.foundry/bin:$PATH"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RPC_URL="http://localhost:8545"
GATEWAY_URL="http://localhost:8080"
OWNER_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
CONTRACT="0x5FbDB2315678afecb367f032d93F642f64180aa3"

echo "==> Deploying SkillPassCertificate (account[0] nonce=0)..."
cd "$REPO_ROOT/contracts"
PRIVATE_KEY="$OWNER_KEY" forge script script/Deploy.s.sol:Deploy \
  --rpc-url "$RPC_URL" \
  --broadcast \
  --quiet

echo "==> Contract should be at: $CONTRACT"
CODE=$(cast code "$CONTRACT" --rpc-url "$RPC_URL")
if [ "$CODE" = "0x" ]; then
  echo "ERROR: no code at $CONTRACT — deploy failed or address mismatch" >&2
  exit 1
fi
echo "    Code confirmed at $CONTRACT"

# NOTE: totalSupply()+1 prediction assumes this script is the sole writer against a fresh
# anvil instance. It is NOT safe against a concurrent minting flow (e.g. the web app's
# useIssueCertificate hook running against the same anvil at the same time) -- acceptable
# here since this script only ever targets a throwaway local dev chain.
SUPPLY=$(cast call "$CONTRACT" "totalSupply()(uint256)" --rpc-url "$RPC_URL")
NEXT_ID=$((SUPPLY + 1))
METADATA_URI_1="${GATEWAY_URL}/certificates/${NEXT_ID}/metadata"

echo "==> Issuing certificate #1 (recipient: account[1])..."
cast send "$CONTRACT" \
  "issueCertificate(address,string,string,string,string,string)" \
  "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" \
  "Full Stack Web3" \
  "Oksa Satya" \
  "SkillPass Academy" \
  "Completed the Full Stack Web3 program" \
  "$METADATA_URI_1" \
  --rpc-url "$RPC_URL" \
  --private-key "$OWNER_KEY" \
  --quiet

NEXT_ID=$((NEXT_ID + 1))
METADATA_URI_2="${GATEWAY_URL}/certificates/${NEXT_ID}/metadata"

echo "==> Issuing certificate #2 (recipient: account[2])..."
cast send "$CONTRACT" \
  "issueCertificate(address,string,string,string,string,string)" \
  "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC" \
  "Smart Contract Security" \
  "Budi Santoso" \
  "SkillPass Academy" \
  "Completed the Smart Contract Security audit course" \
  "$METADATA_URI_2" \
  --rpc-url "$RPC_URL" \
  --private-key "$OWNER_KEY" \
  --quiet

echo "==> Seed complete — 2 certificates issued."
```

- [ ] **Step 3: Fix the pre-existing `dev-verify` off-by-one**

Edit `Makefile` — in the `dev-verify` target, change:

```makefile
	echo "--- GetCertificate token_id=0 ---"; \
	$$GRPCURL -plaintext -d '{"token_id":"0"}' localhost:50051 skillpass.cert.v1.CertificateQuery/GetCertificate; \
```

to:

```makefile
	echo "--- GetCertificate token_id=1 ---"; \
	$$GRPCURL -plaintext -d '{"token_id":"1"}' localhost:50051 skillpass.cert.v1.CertificateQuery/GetCertificate; \
```

(Pre-existing bug, found during this plan's review — the contract's first token ID is 1, not 0, since `_nextTokenId` starts at 1.)

- [ ] **Step 4: Verify the compose file**

Run: `docker compose -f deploy/docker-compose.yml config`
Expected: exits 0, no YAML errors; output includes the new `notify` service resolved correctly with `depends_on.redis.condition: service_healthy`.

- [ ] **Step 5: Full dev-stack smoke test — including a real captured webhook delivery**

Start a throwaway local listener on port 9000 to capture the webhook the `notify` container will POST to (via `host.docker.internal`, Docker Desktop's standard hostname for reaching the host machine from inside a container):

```bash
rm -f /tmp/webhook-capture.txt
nc -l 9000 > /tmp/webhook-capture.txt &
NC_PID=$!
```

Bring up the stack and seed:

```bash
make dev-up
make dev-seed
```

Expected: all 6 containers (`postgres`, `anvil`, `redis`, `indexer`, `notify`, `gateway`) start; `dev-seed` issues 2 certificates as before.

Give the async pipeline (fast-path enqueue → asynq processing → HTTP POST) a few seconds, then check the capture:

```bash
sleep 8
kill "$NC_PID" 2>/dev/null || true
cat /tmp/webhook-capture.txt
```

Expected: the captured request contains a `POST /webhook` line, an `X-SkillPass-Signature: sha256=...` header, and a JSON body containing `"event":"certificate.issued"` and the seeded certificate's `title`/`recipientName` (e.g. `"Full Stack Web3"`, `"Oksa Satya"`).

Then verify the new metadata endpoint against real seeded data:

```bash
curl -sf -w "\nHTTP_STATUS:%{http_code}\n" http://localhost:8080/certificates/1/metadata
```

Expected: HTTP 200, JSON body with `"name":"Full Stack Web3"`, `"description":"Completed the Full Stack Web3 program"`, and an `attributes` array with `Recipient`/`Issuer`/`Issued At` entries matching the seeded certificate.

Tear down:

```bash
make dev-down
```

Expected: clean teardown, `docker compose -f deploy/docker-compose.yml ps -a` empty afterward.

- [ ] **Step 6: Plan-wide verification**

Run:

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./services/... -race -cover
```

Expected: clean build, no gofmt output, every package green.

- [ ] **Step 7: Commit**

```bash
git add deploy/docker-compose.yml deploy/seed.sh Makefile
git commit -m "feat(infra): wire notify into the dev stack, fix seed.sh metadataURI + dev-verify token_id"
```

---

## Plan-wide verification (run once, after all nine tasks)

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./services/... -race -cover
```
Expected: clean build, no gofmt output, every package green.

Per the project's standing code-review discipline (§18): dispatch `superpowers:requesting-code-review` against the full diff (`git merge-base main HEAD` or the commit before Task 1 of this plan, whichever scopes tighter) before considering Phase 7 done — algorithmic-complexity + go-review + Sonar-Go + a critical-thinking pass, same gate the Phase 4-6 plan went through (which found and fixed a real timezone-dependent bug at that stage — do not skip this step).
