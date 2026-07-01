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
