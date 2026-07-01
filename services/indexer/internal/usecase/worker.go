package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
)

// reorgWindow is the confirmation depth: a reorg is only ever reconciled within this many
// blocks of the checkpoint. Fixed, not configurable — matches the project's chosen finality
// assumption (see the Phase 4 design spec).
const reorgWindow = 12

// WebhookMaxRetry bounds delivery attempts for webhook:deliver tasks -- both the fast-path
// enqueue here and the webhook:sweep re-enqueue (asynqjobs/webhook_sweep.go) use this same
// exported const, so a swept retry doesn't silently get a different retry budget than the
// original enqueue. Per the Phase 7 design spec: spans several hours of retry attempts for
// an external endpoint that's temporarily down, without asynq's default (25) keeping a
// permanently-dead endpoint's task alive for days.
const WebhookMaxRetry = 8

// WorkerConfig holds tunable parameters for the indexer worker.
type WorkerConfig struct {
	ChainID      int64
	StartBlock   uint64        // cold-start block (contract deploy block; 0 for anvil)
	BatchSize    uint64        // blocks per FilterLogs call, e.g. 2000
	PollInterval time.Duration // e.g. 5s
}

// Worker is the application service that polls the chain and upserts into the read model.
// It is resumable (persists state after each batch) and idempotent (Upsert is keyed on token_id).
type Worker struct {
	src      EventSource
	repo     CertificateRepo
	cfg      WorkerConfig
	log      *slog.Logger
	pub      EventPublisher // optional; nil-safe (see SetPublisher)
	enqueuer TaskEnqueuer   // optional; nil-safe (see SetEnqueuer)
	next     uint64         // next block to process; resolved on first poll
}

// NewWorker constructs a Worker with its dependencies.
func NewWorker(src EventSource, repo CertificateRepo, cfg WorkerConfig, log *slog.Logger) *Worker {
	if log == nil {
		log = slog.Default()
	}
	return &Worker{src: src, repo: repo, cfg: cfg, log: log}
}

// SetPublisher wires an optional live-event publisher. Call before Run(); safe to never
// call — processLog no-ops the publish step when pub is nil.
func (w *Worker) SetPublisher(pub EventPublisher) {
	w.pub = pub
}

// SetEnqueuer wires an optional background-task enqueuer. Call before Run(); safe to never
// call — processLog no-ops the enqueue step when enqueuer is nil.
func (w *Worker) SetEnqueuer(e TaskEnqueuer) {
	w.enqueuer = e
}

// Run is the long-lived poll loop. It stops when ctx is cancelled.
// Transient poll errors are logged and retried on the next tick — they never kill the worker.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.initNext(ctx); err != nil {
		return fmt.Errorf("worker init: %w", err)
	}

	// Poll once immediately so startup isn't delayed by one full interval.
	if err := w.poll(ctx); err != nil {
		w.log.Error("poll", "err", err)
	}

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.poll(ctx); err != nil {
				w.log.Error("poll", "err", err)
			}
		}
	}
}

// Poll is exported so tests can drive it directly without the ticker.
func (w *Worker) Poll(ctx context.Context) error {
	if err := w.initNext(ctx); err != nil {
		return fmt.Errorf("worker init: %w", err)
	}
	return w.poll(ctx)
}

// initNext resolves the resume point from persisted state (idempotent; no-op after first call).
func (w *Worker) initNext(ctx context.Context) error {
	if w.next != 0 {
		return nil // already initialized
	}
	state, err := w.repo.GetState(ctx)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}
	if state.LastProcessedBlock == 0 {
		w.next = w.cfg.StartBlock
	} else {
		w.next = state.LastProcessedBlock + 1
	}
	return nil
}

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

	// Note: this does not proactively invalidate the trend cache (Phase 6). If the
	// canonical replacement blocks re-add the same certificates, processLog's own
	// EnqueueUnique call refreshes it; if a reorg net-removes certificates with no
	// replacement re-ingest, the cache stays stale until the 15-min cron backstop or
	// the cache's own TTL heals it -- an accepted tradeoff per the Phase 4 design spec.
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
		if err := w.enqueuer.EnqueueUnique(ctx, TrendRefreshTaskType, TrendRefreshTaskType, nil, 0); err != nil {
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
	if err := w.enqueuer.EnqueueUnique(ctx, WebhookDeliverTaskType, taskID, payload, WebhookMaxRetry); err != nil {
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
