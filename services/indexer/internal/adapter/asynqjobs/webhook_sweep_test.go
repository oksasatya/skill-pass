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
	maxRetries  []int  // parallel to enqueuedIDs
	failFor     string // taskID that should fail, if set
}

func (f *fakeSweepEnqueuer) EnqueueUnique(_ context.Context, _, taskID string, _ []byte, maxRetry int) error {
	if taskID == f.failFor {
		return errors.New("fake: enqueue failed")
	}
	f.enqueuedIDs = append(f.enqueuedIDs, taskID)
	f.maxRetries = append(f.maxRetries, maxRetry)
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
	// Final-review finding: a swept re-enqueue must use the same usecase.WebhookMaxRetry
	// budget as the original fast-path enqueue, not asynq's default (25).
	for i, mr := range enq.maxRetries {
		if mr != usecase.WebhookMaxRetry {
			t.Fatalf("enqueuedIDs[%d] maxRetry = %d, want %d", i, mr, usecase.WebhookMaxRetry)
		}
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
