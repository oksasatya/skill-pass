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
