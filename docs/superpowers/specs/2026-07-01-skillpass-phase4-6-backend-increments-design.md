# SkillPass Phase 4–6 — Backend Increments Design

> Backend-only design doc. Frontend follows in a later increment only if one of these phases requires a UI change (none currently do — Phase 5's trend data is new surface, not a required one). Phases 7 (notification/metadata service) and 8 (SIWE auth) are explicitly out of scope here — deliberately split off as independent subsystems, each to get its own brainstorm + spec when picked up.

## Context

BE-1 (indexer) and BE-2 (gateway BFF + SSE) are done and merged to `dev`. This spec covers the next three backend increments, chosen and scoped via user Q&A + a Claude↔Codex (GPT-5.5) cross-model design debate. Each phase closes a gap explicitly deferred from an earlier phase:

- **Phase 4 — Reorg reconcile.** BE-1 stores `block_hash` / `last_processed_hash` per checkpoint but never uses them (see `services/indexer/internal/usecase/worker.go` comment: *"Phase 4 reorg reconcile must fetch the canonical hash of `to` (HeaderByNumber) before trusting it"*). This phase closes that gap.
- **Phase 5 — Data metrics.** Explicitly NOT server/infra observability (Prometheus/OTel rejected) — this is a certificate-issuance **time-series trend**, i.e. metrics about the domain data, not the servers.
- **Phase 6 — Redis + asynq.** The MVP spec deferred Redis until "the gateway is scaled horizontally" or "a job queue appears for async work" (`2026-06-30-skillpass-mvp-design.md` line 135). This phase is that job queue's first real appearance, paired with the one concrete job Phase 5's trend query needs to scale.

## Cross-model review note

This design went through a Claude↔Codex (GPT-5.5, high reasoning) adversarial debate before being written up. Codex's review surfaced three real corrections that are incorporated below (not left as open questions):

1. `last_processed_hash` in the current code is the **last log's** block hash, not the canonical header hash of `last_processed_block` — reconcile cannot work until this is fixed first.
2. A reorg on a block range with **no certificates in it** can't be detected by scanning certificate rows — detection must be based on the checkpoint's own hash chain, not on cert presence.
3. The original draft put the Phase 6 Redis cache behind the **gateway**. This breaks the "gateway only ever talks to the indexer over gRPC" boundary established in BE-2. The cache belongs behind the **indexer**; the gateway's contract is unchanged.

One genuine disagreement did surface — whether the indexer should keep indexing at the chain head (current behavior, SSE stays instant) or switch to confirmed-only indexing (delay all indexing/SSE by the 12-block confirmation window, trading latency for the guarantee that a certificate is never shown before it's final). The user chose to **keep indexing at head** — this preserves the already-shipped, already-verified BE-2 "live" SSE behavior; a certificate silently disappearing from the read model after a deep (>12 block) reorg is accepted as a rare, documented limitation for this pilot's scale rather than solved outright.

---

## Phase 4 — Reorg reconcile

### Problem

The ingest worker (`services/indexer/internal/usecase/worker.go`) advances `last_processed_block` after every batch, but never re-verifies that the blocks it already processed are still on the canonical chain. If Base Sepolia (or a local anvil restart) reorgs blocks the indexer already ingested, the read model silently drifts from the chain and never self-corrects.

### Design

**1. Fix checkpoint semantics first.** `IndexerState.LastProcessedHash` must always be the canonical header hash of `LastProcessedBlock`, fetched via `ethclient.HeaderByNumber`, regardless of whether that block contained any `CertificateIssued` logs. This replaces the current "last log's block hash, or empty if no logs" behavior. This is a pure correctness fix to `Worker.poll` and the `chain.EventSource` port — no schema change (the columns already exist from BE-1).

**2. Cheap detection on every poll.** After the existing poll-and-advance logic runs, fetch the canonical header hash of `last_processed_block` again and compare it to the stored `LastProcessedHash`:
- **Match** → no reorg, nothing else to do. This costs exactly one extra `HeaderByNumber` RPC call per poll cycle (not a full 12-block re-check every cycle, which would be wasteful at `PollInterval` values as low as 2s in dev — the one-block check is the cheap common case).
- **Mismatch** → a reorg happened somewhere at or before `last_processed_block`. Proceed to reconcile.

**3. Reconcile by rewinding the full confirmation window, not by searching for the exact divergence point.** On a detected mismatch:
- Compute `rewindTo := max(StartBlock, last_processed_block - 12)` (12 = the confirmation depth chosen for this project).
- Delete all indexed certificates with `block_number > rewindTo`.
- Set `last_processed_block = rewindTo` (and refetch+store the canonical hash for that block).
- Let the normal poll loop re-ingest forward from there on its next cycle — no new ingest code path. `Upsert` is already idempotent (`UNIQUE(chain_id, tx_hash, log_index)` from BE-1), so re-processing blocks that *didn't* actually reorg is safe and just re-writes the same rows.

This is deliberately the simpler of the two options Codex raised (rewind-the-whole-window vs. persist a per-block hash chain and find the exact common ancestor). Because certificate rows only exist where certificates were issued, hunting for "the first mismatched certificate's block" can miss a reorg that happened in an empty stretch of blocks — rewinding the full window sidesteps that without adding a new `indexed_blocks` table. The cost is bounded (at most 12 blocks' worth of certificates ever get deleted-and-replayed per reorg event), which is proportionate for a bounded 12-block confirmation window.

**4. Accepted limitation (explicit, not silent).** A certificate that was indexed and already pushed over the live SSE feed to a connected frontend, then reorg'd away before reaching 12 confirmations, is deleted from the read model with **no explicit "removed" event** sent to the client. The frontend will simply stop seeing it on the next `ListCertificates`/`GetCertificate` refresh. Given deep reorgs are rare on Base Sepolia and this is a pilot-scale learning project, this gap is accepted rather than solved with a retraction protocol. (Re-publishing the same certificate as an "issued" SSE event again after a shallow reorg + natural replay is harmless — the frontend's SSE handler already just calls `invalidateQueries()`, not an append, so a duplicate event is a no-op re-fetch, not a duplicate list entry.)

### Testing

TDD: **yes** — this is exactly the class of bug-prone, input→output-contract logic the project's TDD-fit gate calls for (a state-machine-shaped correctness fix with a reproducible failure mode). Extend `services/indexer/internal/usecase/worker_test.go`'s existing `fakeEventSource` with a way to return a different canonical hash for a given block number (simulating a reorg), then assert: (a) a matching hash is a no-op, (b) a mismatched hash triggers the rewind-and-delete, (c) certificates within the rewound range are gone from the fake repo afterward, (d) the next `Poll()` call re-ingests them via the existing idempotent path.

---

## Phase 5 — Certificate issuance trend (data metrics)

### Problem

There's currently no way to see certificate-issuance trends over time — only point lookups (`GetCertificate`) and a flat paginated list (`ListCertificates`). This phase adds a time-series aggregate, explicitly scoped to *domain data* insight (not server/infra telemetry, which was explicitly rejected).

### Design

**New gRPC method** on `CertificateQuery` (indexer):

```protobuf
enum TrendBucket {
  TREND_BUCKET_UNSPECIFIED = 0;
  TREND_BUCKET_DAY = 1;
  TREND_BUCKET_WEEK = 2;
  TREND_BUCKET_MONTH = 3;
}

message GetIssuanceTrendRequest {
  TrendBucket bucket = 1;
  string range_preset = 2; // one of the bounded presets below for this bucket
}

message TrendPoint {
  google.protobuf.Timestamp bucket_start = 1; // UTC
  uint64 count = 2;
}

message GetIssuanceTrendResponse {
  repeated TrendPoint points = 1;
}
```

**Bounded range presets** (per bucket, so Phase 6's cache can enumerate exactly what to precompute — an unbounded client-supplied date range can't be usefully cached and can't be validated cheaply):

| Bucket | Supported `range_preset` values |
|---|---|
| `DAY` | `30d`, `90d`, `365d` |
| `WEEK` | `12w`, `52w` |
| `MONTH` | `12m`, `24m` |

An invalid bucket/range_preset combination is rejected with `codes.InvalidArgument` at the gRPC layer (mirrors the existing `errEmptyTokenID` pattern in `adapter/grpc/server.go`).

**Query implementation:** bucket selects between three separate sqlc queries (one per `date_trunc` granularity) chosen by a Go `switch` on the validated enum — **not** a dynamically interpolated `date_trunc($1, issued_at)` string, which would be an unnecessary dynamic-SQL smell for a value that's fully enumerable. All three queries operate in UTC: `date_trunc('day', issued_at AT TIME ZONE 'UTC')` (etc.), so a deployment in any server timezone still buckets consistently.

**Zero-filled buckets:** Postgres `GROUP BY` only returns buckets that have ≥1 row. The usecase layer fills in every bucket in the requested range with `count: 0` where the query returned nothing, so the frontend (whenever it's wired up) never has to do gap-filling itself. This is `O(number of buckets in the range)` — bounded by the preset table above (max 365 for `30d`/`90d`/`365d` at the day granularity), trivial at any realistic scale.

**Migration:** add `CREATE INDEX idx_certificates_issued_at ON certificates (issued_at);` — supports the `WHERE issued_at >= $1` range filter each trend query applies before aggregating. (goose migration, sqlc regenerate, per the existing BE-1 pattern.)

**Gateway:** thin REST wrapper, `GET /stats/trend?bucket=day&range=30d`, following the exact shape of the existing `GET /certificates` handler — validate query params, call the indexer's gRPC method, map to a JSON DTO, no business logic in the gateway.

**Complexity (Bahasa Indonesia, per project convention for data-manipulation work):** Query-nya `O(cert dalam range)` waktu (index di `issued_at` bikin filter range-nya cepat, lalu Postgres agregasi/`GROUP BY` sisanya), lalu zero-fill di Go `O(jumlah bucket)` — dua-duanya dibatasi preset (max 365 hari/bucket), jadi biayanya kecil dan predictable di skala pilot ini (ratusan-ribuan sertifikat). Ga ada nested loop atau N+1 — satu query per request, satu pass buat isi bucket kosong.

### Testing

TDD: **yes** for the pure zero-fill/bucket-alignment helper (input: raw DB rows + a requested range → output: a complete, gap-filled slice of `TrendPoint`) — table-driven, matches the project's existing `toDTO`/`parsePageSize` pure-helper test style. TDD: **no** for the thin gRPC/REST wiring (verify by build + an integration test against a seeded Postgres, mirroring the existing `adapter/postgres` test suite's testcontainers pattern) — add a regression test confirming the normal path (a few seeded certs across known dates) returns the expected bucketed counts.

---

## Phase 6 — Redis + asynq (issuance trend cache)

### Problem

Phase 5's trend query is cheap today but will re-scan+re-aggregate on every request as the certificate count grows. This phase introduces Redis + asynq — deliberately paired with Phase 5's query as the first concrete job, rather than shipping bare infrastructure with nothing to run (a bare "infra-only" Redis addition was explicitly rejected).

### Design

**Boundary (the corrected part, post-Codex-debate): Redis lives behind the indexer only.** The gateway's contract is completely unchanged — it still calls `GetIssuanceTrend` over gRPC exactly as Phase 5 defined it. The indexer's *implementation* of that gRPC method becomes cache-aware internally:

1. On `GetIssuanceTrend(bucket, range_preset)`, check Redis for key `trend:v1:{chain_id}:{bucket}:{range_preset}`.
2. **Hit** → return the cached `GetIssuanceTrendResponse` (deserialized from the stored JSON/protobuf bytes).
3. **Miss** → run the Phase 5 live query, write the result to Redis (so correctness never depends on the background job having run yet — a cold cache is a slow-but-correct response, never an error), then return it.

**The one concrete job:** reuses the existing `services/indexer/internal/platform/broadcast` publish path (the same one BE-2's SSE feature already wired the Worker's `processLog` into). After every successful certificate upsert, additionally enqueue an asynq task `RefreshTrendCacheTask` with a **fixed, unique task ID** (asynq's unique-task option) — so if ten certificates land within the same debounce window, only one refresh task is actually queued/run, not ten. The task recomputes all nine bucket/range-preset combinations from the table in Phase 5 and writes them all to Redis. A low-frequency cron backstop (asynq's `Scheduler`, e.g. every 15 minutes) also enqueues the same task as cheap insurance against a missed or failed event-triggered enqueue.

This gives asynq two genuinely different, well-motivated exercises (event-triggered unique task + periodic scheduled task) rather than one arbitrary cron with nothing forcing it to exist.

**New config (indexer):** `REDIS_ADDR` (required once this phase ships), asynq concurrency (small default, e.g. 5 — this is a single background job type, not a general-purpose queue yet).

**docker-compose:** add a `redis` service (standard `redis:7-alpine` image) to `deploy/docker-compose.yml`; indexer's `depends_on` gains `redis: condition: service_healthy`.

### Testing

TDD: **yes** for the cache-key construction and the debounce/unique-task-ID logic (pure, input→output, easy to get subtly wrong) and for the cache-hit/cache-miss branching in the gRPC handler (fake a Redis client interface, mirror the existing `usecase.CertificateRepo` fake-based test style). TDD: **no** for the asynq server/scheduler bootstrap and Redis client wiring in `cmd/indexer/main.go` (infra wiring — verify by running, add a normal-path regression test).

---

## Cross-cutting

- **Architecture invariants preserved:** gateway still only talks to the indexer over gRPC (no new gateway dependency); `domain` packages stay framework-free; single Go module; hexagonal-lite layering unchanged. Phase 6 is the first phase where the indexer itself gains a new external dependency (Redis) beyond Postgres + the chain RPC — this is the deliberate, explicitly-decided introduction point per the original MVP spec's own deferred-Redis note.
- **New dependencies:** a Redis client (`github.com/redis/go-redis/v9` — the current idiomatic choice) and `github.com/hibiken/asynq`, both added to `go.mod` in Phase 6 only.
- **Migrations:** one new migration in Phase 5 (`idx_certificates_issued_at`). Phase 4 and Phase 6 need no schema changes.
- **Sonar-Go / golang-expert discipline:** carries forward unchanged from BE-1/BE-2 — will be enumerated in full (with the verbatim guardrails block) in each phase's implementation plan, per the project's standing per-UC briefing convention.
- **Out of scope for this spec:** Phase 7 (notification/metadata service) and Phase 8 (SIWE auth) — deliberately split off in the scoping discussion as independent subsystems (a new domain service vs. a cross-cutting authentication concern), each to be brainstormed and spec'd on its own when picked up next.

## Open items

None — every fork raised during the cross-model debate was either resolved by adopting Codex's correction (checkpoint-hash fix, cert-row-independent rewind, cache-behind-indexer boundary, bounded presets) or explicitly decided by the user (keep indexing at head; accept the rare silent-removal gap on deep reorg).
