# SkillPass Phase 7 — Notification & Metadata Design

**Status:** Approved (design). Adversarially reviewed against the codebase in two rounds via a Claude↔Codex (GPT-5.5) cross-model debate; both rounds' findings are folded into this document, not summarized away.

## Overview

Phase 7 adds two independent capabilities, deliberately split from the original "notification/metadata service" phase name into their actual architectural shapes:

1. **A metadata endpoint** — `GET /certificates/{tokenId}/metadata` on the existing `services/gateway`, serving ERC-721-style JSON so wallets/marketplaces can render a certificate. Not a new service: it is a pure reshape of data the gateway already fetches via its existing `GetCertificate` gRPC call.
2. **A notification service** (`services/notify/`) — a new, Postgres-free deployable service that delivers a signed webhook to one configured external consumer whenever a certificate is issued, with durable, reorg-safe, crash-safe delivery.

Both were scoped through direct user decisions (recorded below) and stress-tested via two rounds of adversarial review that found and fixed two real defects in the first draft (see "Reorg-safe webhook dedup" below) — this is not a rubber-stamped design.

## Scope decisions (already made, do not relitigate)

- Metadata = HTTP API generating JSON from existing Postgres-backed data, not IPFS pinning (no external pinning-service dependency; IPFS is deferred to a later phase if ever needed).
- Notification = webhook to an external consumer (not email — no email capture mechanism exists; not FE toast — SSE already covers that via Phase 3).
- Webhook registration = config-driven, admin-only (`WEBHOOK_URL`/`WEBHOOK_SECRET` env vars, both `mustenv`-required) — no public self-service subscription API, because no auth system exists yet (SIWE is a later, separate phase) and a public registration endpoint would be an SSRF vector.
- One webhook target only — no multi-subscriber config; genuinely no second consumer exists yet, and flat env vars match the existing `config.go` pattern used by indexer/gateway.
- Delivery history = structured `slog` logs only, no queryable Postgres table for it — add one later only if a real need to query history appears.
- `deploy/seed.sh` (the only current certificate-issuance mechanism) is updated in-scope, so the new metadata endpoint is actually reachable from real on-chain data, not just testable by hand.

## Architecture

```
                                    ┌──────────────────┐
                                    │  webhook target   │
                                    │ (config: 1 URL)    │
                                    └────────▲──────────┘
                                             │ HTTP POST + HMAC-SHA256 sig
                                    ┌────────┴──────────┐
                                    │  services/notify   │◄── asynq consumer only
                                    │  (no Postgres)      │    (shared Redis)
                                    └────────▲──────────┘
                                             │ webhook:deliver task
┌──────────┐   gRPC    ┌───────────┐  Upsert +  ┌─────────────┐
│ gateway  │──────────►│  indexer  │───────────►│   Worker.    │
│ (BFF)    │           │  (owns    │  outbox    │  processLog  │
│          │           │  Postgres)│  insert    └──────┬──────┘
│ GET /certificates/    │           │                   │ webhook:sweep (cron, ~5min)
│  {id}/metadata ───────┘           │◄──────────────────┘ backstop for missed enqueues
└──────────┘                        └───────────┘
```

- **Metadata**: gateway-only change, no new gRPC/proto.
- **Notify**: new service, Redis-only, zero Postgres, zero public API surface beyond `/healthz`.
- **Outbox + sweep**: lives in the **indexer** (it already owns Postgres and already runs an asynq server+scheduler for Phase 6's trend-refresh cron) — not in notify, which stays a pure, stateless-beyond-Redis delivery worker.

## 1. Metadata endpoint (`services/gateway`)

**Route** (`router.go`): `mux.Handle("GET /certificates/{tokenId}/metadata", GetCertificateMetadata(d))`.

**Handler**: reuses `d.Cert.GetCertificate` (existing gRPC call, same as `GetCertificate` handler) — no new proto. 404 via the existing `writeGRPCError` on `NotFound`, same pattern as `GetCertificate`.

**JSON shape** (ERC-721 metadata convention, `image` deliberately omitted — no image asset exists in this project; omitting is honest, faking one is not):

```json
{
  "name": "Full Stack Web3",
  "description": "Completed the Full Stack Web3 program",
  "attributes": [
    {"trait_type": "Recipient", "value": "Oksa Satya"},
    {"trait_type": "Issuer", "value": "SkillPass Academy"},
    {"trait_type": "Issued At", "value": "2026-07-01T08:00:00Z"}
  ]
}
```

## 2. Reorg-safe webhook dedup (indexer)

### The problem an earlier draft got wrong

A first draft proposed deduping webhook-enqueues the same way Phase 6 debounces trend-refresh: `enqueuer.EnqueueUnique(ctx, WebhookDeliverTaskType, fmt.Sprintf("webhook:deliver:%s", cert.TokenID), payload)`, relying on asynq's own `Unique(uniqueTTL)` (5 minutes). **Rejected** — verified against actual code:

- `Worker.reconcile()`'s `DeleteFromBlock` genuinely deletes certificate rows on a detected reorg; the normal poll loop then re-ingests whatever is still present in the new canonical chain, commonly the *exact same transaction* re-mined into a later block.
- If that re-ingest happens more than 5 minutes (the existing `uniqueTTL` const) after the original webhook fired — very plausible, reorgs are not detected instantly — asynq's dedup window has already expired, and the "same" certificate would fire a second, false "new certificate" webhook.
- Extending `uniqueTTL` only shrinks the probability window; it does not close it, and it overloads one const with two unrelated meanings (debounce-window for trend-refresh vs. correctness-guarantee for webhook dedup).
- A first fix attempt (a Postgres marker table keyed `(chain_id, tx_hash, log_index)`, checked via `INSERT ... ON CONFLICT DO NOTHING` before enqueueing) was itself found wrong by adversarial review: `LogIndex` (`ev.Raw.Index` in `eventsource.go`) is **block-relative** — the same transaction re-mined into a different block can land at a different log index, defeating the dedup. It also had a silent-loss window: if the Postgres insert succeeded but the subsequent enqueue failed or the process crashed, the marker was permanently "used up" with no webhook ever sent and no backstop.

### The converged design

**Dedup key:** `(chain_id, tx_hash, token_id)` — not `log_index`. `token_id` is stable across re-mining of the same transaction (assigned deterministically by contract execution order, not block position) and correctly does *not* collapse two genuinely distinct certificates that might one day share a `tx_hash` under a hypothetical batch-mint-via-multicall (each gets its own `token_id`). This is deliberately a different key shape than the existing `certificates` table's `UNIQUE (chain_id, tx_hash, log_index)` constraint (migration `00001_init.sql`) — that constraint answers "is this the same *log entry*", which `log_index` correctly disambiguates for; this one answers "have we already told an external consumer about this *certificate*", which needs `token_id`. Different questions, deliberately different keys.

**Outbox pattern** (migration `00003_webhook_outbox.sql`):

```sql
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
```

`CertificateRepo` gains three methods (added to the existing interface, not a new port — see "Port shape" below):

- `InsertWebhookOutbox(ctx, chainID int64, txHash, tokenID string, payload []byte) (isNew bool, error)` — `INSERT ... ON CONFLICT (chain_id, tx_hash, token_id) DO NOTHING RETURNING id`; `isNew` true iff a row came back.
- `ListUnenqueuedWebhookOutbox(ctx, limit int) ([]WebhookOutboxEntry, error)` — `SELECT id, payload FROM webhook_outbox WHERE enqueued_at IS NULL ORDER BY id LIMIT $1`.
- `MarkWebhookOutboxEnqueued(ctx, id int64) error` — `UPDATE webhook_outbox SET enqueued_at = now() WHERE id = $1`.

**`Worker.processLog()` flow**, after the existing `Upsert` succeeds:

```go
isNew, err := w.repo.InsertWebhookOutbox(ctx, w.cfg.ChainID, cert.TxHash, cert.TokenID, payload)
if err != nil {
    return fmt.Errorf("insert webhook outbox %s: %w", cert.TokenID, err) // FATAL, not logged-and-continued
}
if isNew && w.enqueuer != nil {
    // Fast path: most webhooks go out within one poll cycle. Best-effort — the
    // webhook:sweep cron (below) is the durable backstop if this fails.
    if err := w.enqueuer.EnqueueUnique(ctx, WebhookDeliverTaskType, fmt.Sprintf("webhook:deliver:%d", outboxID), payload); err == nil {
        _ = w.repo.MarkWebhookOutboxEnqueued(ctx, outboxID) // best-effort; sweep retries if this fails
    } else {
        w.log.Warn("enqueue webhook deliver", "err", err)
    }
}
```

**Why `InsertWebhookOutbox`'s error must be fatal (propagated), unlike the enqueue step below it:** confirmed in round 2 of the adversarial review against the actual `poll()`/`processLog()` code — `poll()` only calls `SaveState` (advances the checkpoint) after *every* log in the batch's `processLog` call returns without error (`worker.go`). If `InsertWebhookOutbox` fails and that error propagates, the whole batch — including the just-succeeded `Upsert` — is retried on the next poll cycle (both `Upsert` and `InsertWebhookOutbox` are idempotent under retry via their own `ON CONFLICT` clauses, so re-running the batch is harmless). This closes the crash-window gap entirely: nothing between "certificate upserted" and "outbox row durably recorded" can be silently lost, because the checkpoint simply doesn't move until both succeed. The *enqueue* step (asynq call) stays best-effort/logged, same as the existing trend-refresh pattern, because the outbox row is now the durable source of truth and the sweep cron is its backstop — the enqueue step no longer needs to be the safety mechanism.

**`webhook:sweep` — a scheduled cron *trigger* + a separate mux *handler*, mirroring the existing trend-refresh split exactly (not "just a cron entry" — asynq's `Scheduler` can only periodically enqueue a fixed task; the actual DB query has to run in a handler registered on the mux):**

- Indexer's existing `asynq.Scheduler` (already running for the 15-min trend-refresh cron) gets a second registration: every ~5 minutes, enqueue a payload-less `webhook:sweep` trigger task.
- Indexer's existing `asynq.ServeMux` (already running the `RefreshTrendCacheHandler`) gets a second handler, `WebhookSweepHandler`, registered on `WebhookSweepTaskType`. On invocation: `ListUnenqueuedWebhookOutbox(ctx, limit)`, and for each row found, re-attempt the same `EnqueueUnique` + `MarkWebhookOutboxEnqueued` sequence the fast path uses.
- This lives in the **indexer** (which has Postgres access), not in `services/notify` (which does not) — the notify service only ever registers a `webhook:deliver` handler on its own separate asynq mux, connected to the same Redis, and never queries Postgres.

**Residual accepted edge case (documented, not silently ignored):** if `EnqueueUnique` succeeds but the subsequent `MarkWebhookOutboxEnqueued` UPDATE fails, the row stays unmarked and the sweep will later re-attempt a delivery that already went out. Round 2 of the review confirmed the realistic trigger for this is graceful shutdown / context cancellation (the worker runs on a cancelable `gCtx`; shutdown cancels the context then stops asynq) more often than a generic "DB hiccup." This narrow window is bounded by asynq's own per-task uniqueness only if the sweep happens to run within asynq's dedup TTL of the original enqueue; outside that window a rare duplicate webhook could still go out. Full elimination needs two-phase-commit semantics between Postgres and Redis — judged disproportionate for this project's stage. The code carries a comment naming this ceiling explicitly (`// webhook: <ceiling>, see design doc §2`), not a silent gap.

**Port shape:** the three outbox methods are added to the existing `CertificateRepo` interface, not a new port. Reasoning, grounded in this codebase's actual precedent (`ports.go`): `CertificateRepo` already accumulates every *Postgres*-backed concern served by the same adapter/pool — certificate CRUD, indexer checkpoint (`GetState`/`SaveState`), and Phase 5's trend reads (`GetIssuanceTrend`) all live there despite being conceptually distinct. Phase 6 gave genuinely *new infrastructure* (Redis caching, task queueing) their own small ports (`TrendCache`, `TaskEnqueuer`) specifically because those are a different backing store. The webhook outbox is Postgres-backed like everything else on `CertificateRepo`, so by the codebase's own convention it belongs there, not behind a new interface invented for this feature alone.

**`TaskEnqueuer.EnqueueUnique` gains a payload parameter** — one method, not two:

```go
EnqueueUnique(ctx context.Context, taskType, taskID string, payload []byte) error
```

The existing trend-refresh call site passes `nil`. Confirmed in round 2: the port is already generic by `taskType`/`taskID`, the adapter currently hardcodes `nil`, and the blast radius is exactly the three known call sites (`worker.go`'s trend-refresh call, the real `Enqueuer` adapter, the `fakeEnqueuer` test double) — small and mechanical. A second, near-identical method was considered and rejected (ambiguity about which to call, no real benefit over one method with an optional payload).

## 3. Notification service (`services/notify/`)

```
services/notify/
  cmd/notify/main.go                    — composition root: asynq consumer (webhook:deliver only) + /healthz
  internal/config/config.go             — mustenv: REDIS_ADDR, WEBHOOK_URL, WEBHOOK_SECRET
  internal/usecase/sign.go              — pure: SignPayload(secret, body []byte) string (hex HMAC-SHA256)
  internal/adapter/webhook/handler.go   — asynq.Handler: unmarshal payload, sign, POST (net/http, 10s timeout)
```

No Postgres anywhere in this service. Config mirrors the existing `mustenv`/`getenv` pattern in `services/indexer/internal/config/config.go` exactly.

**Webhook envelope:**

```json
{
  "event": "certificate.issued",
  "data": {
    "tokenId": "1", "ownerAddress": "0x...", "title": "...", "recipientName": "...",
    "issuerName": "...", "description": "...", "issuedAt": "2026-07-01T08:00:00Z",
    "chainId": 31337, "txHash": "0x..."
  }
}
```

**Signature:** `X-SkillPass-Signature: sha256=<hex HMAC-SHA256 over the raw JSON body>`, using `WEBHOOK_SECRET` — mirrors the GitHub/Stripe convention.

**Retry:** `asynq.MaxRetry(8)` with asynq's own default exponential-backoff `RetryDelayFunc` (library default, not custom) — spans several hours of retry attempts for an external endpoint that's temporarily down. HTTP client timeout: 10s. Any non-2xx response, or a request error, is treated as a failed delivery (asynq retries automatically); 2xx is success.

**Delivery outcome:** `slog` structured logs only (chosen scope — no DB table for delivery history).

**`docker-compose.yml`:** new `notify` service, `depends_on: redis: condition: service_healthy` only (no dependency on `indexer` or `postgres` — notify is a pure Redis consumer; tasks queue durably regardless of startup order). Env: `REDIS_ADDR`, `WEBHOOK_URL`, `WEBHOOK_SECRET`.

## 4. `deploy/seed.sh` + `Makefile` changes

- Before each `cast send ... issueCertificate`, compute the next token ID: `NEXT_ID=$(($(cast call "$CONTRACT" "totalSupply()(uint256)" --rpc-url "$RPC_URL") + 1))`, then `METADATA_URI="http://localhost:8080/certificates/${NEXT_ID}/metadata"`, replacing the current hardcoded `"ipfs://demo1"` / `"ipfs://demo2"` placeholders.
- **Documented caveat** (code comment, not silently assumed): this prediction is safe only because `deploy/seed.sh` is the sole writer against a fresh anvil instance in this flow. Verified (round 2 of review) that `apps/web/src/hooks/useIssueCertificate.ts` does *not* predict a token ID at all — it parses the actual `CertificateIssued` event from the transaction receipt after mining — so there is no competing prediction race with this script today; the caveat exists only against a hypothetical future concurrent-minting flow, and is stated honestly rather than assumed away.
- Cosmetic fix: seed.sh's echo labels ("Issuing certificate #0" / "#1") are renamed to "#1"/"#2" to match the certificates' *actual* assigned token IDs (contract's `_nextTokenId` starts at 1, not 0).
- **Drive-by fix** (pre-existing bug, found during this review, cheap to fix in the same touched file): `Makefile`'s `dev-verify` target queries `token_id: "0"`, which never exists — changed to `"1"`.

## Constraints

- Single Go module at repo root — `services/notify/` is a new directory under the same module, no new `go.mod`.
- Hexagonal-lite: `services/notify/internal/usecase` (the pure signing function) imports nothing from `internal/adapter` or third-party HTTP/asynq packages.
- Sonar-Go guardrails (write compliant from the first commit):

```
# Sonar guardrails — write compliant code from start
Go:
- go:S107 — ≤7 params (project preference ≤5; 6+ = smell → Deps/Opts struct from the start).
- go:S3776 — cognitive complexity ≤15 → extract helpers; tests use t.Run subtests to reset the budget.
- go:S1192 — const for any string literal duplicated 3+ times.
```

## Error handling & testing

- **Metadata endpoint** — `TDD: no` (thin REST→gRPC reshape, same pattern as the existing `GetCertificate` handler). Add tests after: happy path + 404-on-not-found regression, matching `certificates_test.go`'s existing style.
- **`SignPayload`** — `TDD: yes` (pure function, HMAC correctness is easy to get subtly wrong — wrong encoding, wrong header format — and cheap to verify with a failing test first).
- **Notify's asynq handler** — `TDD: no` (thin wiring over signing + HTTP), but add tests with an `httptest.Server` covering: happy path (2xx → success), non-2xx (→ returns error, asynq retries), signature header present and correct.
- **`InsertWebhookOutbox`/`ListUnenqueuedWebhookOutbox`/`MarkWebhookOutboxEnqueued`** — `TDD: yes` for the dedup behavior specifically (a testcontainers integration test asserting: first insert of a given `(chain_id, tx_hash, token_id)` returns `isNew=true`; a second insert of the *same* key returns `isNew=false` and does not create a second row — this is the exact behavior the whole reorg-safety argument rests on, so it must be red-green tested, not assumed).
- **`EnqueueUnique` payload param** — add a unit test asserting non-nil payload bytes actually reach the created `asynq.Task` (round-2 reviewer's specific suggestion).
- **`Worker.processLog()`'s new fatal-on-`InsertWebhookOutbox`-error behavior** — `TDD: yes` for this specific control-flow change: a test asserting that when the fake outbox-insert returns an error, `processLog` returns an error too (not swallowed), and the existing worker test suite (12 tests as of Phase 6) must be re-run in full as a regression check since this touches the same function every prior Worker test already exercises.
- **`webhook:sweep` handler** — `TDD: no` (thin wiring over `ListUnenqueuedWebhookOutbox` + `EnqueueUnique`), but add a test verifying it re-enqueues exactly the unenqueued rows and skips already-enqueued ones.
- **`deploy/seed.sh`** — `TDD: no` (shell script, verified by running the full dev-stack smoke test, same discipline as Phase 6 Task 14: `make dev-up && make dev-seed`, then curl the new metadata endpoint expecting the real seeded title/recipient/issuer, confirming the wiring is actually correct end-to-end, not just plausible on paper).
