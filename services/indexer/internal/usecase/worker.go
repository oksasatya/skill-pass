package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
)

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
	src  EventSource
	repo CertificateRepo
	cfg  WorkerConfig
	log  *slog.Logger
	next uint64 // next block to process; resolved on first poll
}

// NewWorker constructs a Worker with its dependencies.
func NewWorker(src EventSource, repo CertificateRepo, cfg WorkerConfig, log *slog.Logger) *Worker {
	if log == nil {
		log = slog.Default()
	}
	return &Worker{src: src, repo: repo, cfg: cfg, log: log}
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
	var lastHash string
	for _, l := range logs {
		if err := w.processLog(ctx, l); err != nil {
			return err // state not advanced; batch re-tried next poll
		}
		lastHash = l.BlockHash
	}

	// ponytail: LastProcessedHash = the last log's block hash (empty if the range had no logs),
	// not necessarily block `to`'s canonical hash. It is stored but unused in BE-1; Phase 4 reorg
	// reconcile must fetch the canonical hash of `to` (HeaderByNumber) before trusting it.
	newState := domain.IndexerState{
		ChainID:            w.cfg.ChainID,
		LastProcessedBlock: to,
		LastProcessedHash:  lastHash,
	}
	if err := w.repo.SaveState(ctx, newState); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	w.next = to + 1
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
	return nil
}
