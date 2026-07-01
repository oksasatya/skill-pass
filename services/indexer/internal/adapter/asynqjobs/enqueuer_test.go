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
	if err := enq.EnqueueUnique(ctx, "trend:refresh", "trend:refresh", nil, 0); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	// second call with the same taskID must be a no-op, not an error
	if err := enq.EnqueueUnique(ctx, "trend:refresh", "trend:refresh", nil, 0); err != nil {
		t.Fatalf("second (duplicate) enqueue should be absorbed, got: %v", err)
	}
	_ = time.Second // no sleep needed — dedup is immediate within the TTL window
}

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
	const wantMaxRetry = 8
	if err := enq.EnqueueUnique(ctx, "webhook:deliver", "webhook:deliver:1", wantPayload, wantMaxRetry); err != nil {
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
	// Final-review finding: EnqueueUnique's maxRetry param must actually reach asynq's
	// TaskInfo.MaxRetry, not just be silently ignored -- this is the strongest possible
	// check, against real asynq state, not a fake's recorded value.
	if tasks[0].MaxRetry != wantMaxRetry {
		t.Errorf("MaxRetry = %d, want %d", tasks[0].MaxRetry, wantMaxRetry)
	}
}

func TestEnqueuer_EnqueueUnique_ZeroMaxRetry_LeavesAsynqDefault(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := asynq.NewClient(asynq.RedisClientOpt{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	enq := asynqjobs.NewEnqueuer(client)

	ctx := context.Background()
	if err := enq.EnqueueUnique(ctx, "trend:refresh", "trend:refresh:zero-test", nil, 0); err != nil {
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
	// maxRetry=0 must NOT be sent to asynq as MaxRetry(0) -- it must leave asynq's own
	// default (25) untouched, confirming the "maxRetry > 0" guard in EnqueueUnique works.
	const asynqDefaultMaxRetry = 25
	if tasks[0].MaxRetry != asynqDefaultMaxRetry {
		t.Errorf("MaxRetry = %d, want asynq's default %d (maxRetry=0 must not override it)", tasks[0].MaxRetry, asynqDefaultMaxRetry)
	}
}
