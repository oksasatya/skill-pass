// Package webhook implements the asynq handler that signs and delivers webhook payloads
// produced by the indexer's webhook:deliver task.
package webhook

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/notify/internal/usecase"
)

// httpTimeout bounds a single delivery attempt -- asynq's own retry/backoff handles
// retrying a slow or unreachable endpoint across separate attempts.
const httpTimeout = 10 * time.Second

// DeliverTaskType must match usecase.WebhookDeliverTaskType on the indexer side --
// duplicated here (not imported) because Go's internal/ package visibility means notify
// cannot import services/indexer/internal/usecase at all; the two deployables only agree
// on this wire-level string.
const DeliverTaskType = "webhook:deliver"

// Handler is an asynq.Handler that signs and POSTs the task's payload to the configured
// webhook URL.
type Handler struct {
	url    string
	secret string
	client *http.Client
}

// NewHandler constructs a Handler.
func NewHandler(url, secret string) *Handler {
	return &Handler{url: url, secret: secret, client: &http.Client{Timeout: httpTimeout}}
}

// ProcessTask satisfies asynq.Handler. A non-2xx response or request error returns an
// error, which asynq retries per its own MaxRetry/backoff configuration on the task.
func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	body := t.Payload()
	sig := usecase.SignPayload(h.secret, body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SkillPass-Signature", "sha256="+sig)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("deliver webhook: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable here

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook delivery failed: status %d", resp.StatusCode)
	}
	return nil
}
