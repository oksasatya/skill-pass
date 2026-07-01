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
