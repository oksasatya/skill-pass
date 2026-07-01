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
