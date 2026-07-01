package usecase

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
)

// WebhookDeliverTaskType identifies the asynq task that delivers one signed webhook.
// Defined here (not in the adapter) so Worker can reference it without importing adapter
// code. NOTE: services/notify cannot import this constant directly -- Go's internal/
// package visibility rules mean only code under services/indexer/ may import
// services/indexer/internal/usecase. notify carries its own copy of this exact string;
// the two are two independent deployables that only agree on the wire-level string.
const WebhookDeliverTaskType = "webhook:deliver"

// WebhookSweepTaskType identifies the asynq task (Task 5) that scans webhook_outbox for
// rows not yet enqueued and retries them -- the durable backstop for a missed enqueue.
const WebhookSweepTaskType = "webhook:sweep"

// webhookCertificateIssuedEvent is the "event" field value for every webhook payload this
// codebase currently emits -- a single fixed value, not yet an enum, since there is only
// one event type today.
const webhookCertificateIssuedEvent = "certificate.issued"

// WebhookEvent is the JSON envelope delivered to the configured webhook consumer when a
// certificate is issued. Field names/shape are a stable external contract -- changing them
// is a breaking change for any real consumer.
type WebhookEvent struct {
	Event string           `json:"event"`
	Data  WebhookEventData `json:"data"`
}

// WebhookEventData is the certificate payload nested inside WebhookEvent.
type WebhookEventData struct {
	TokenID       string `json:"tokenId"`
	OwnerAddress  string `json:"ownerAddress"`
	Title         string `json:"title"`
	RecipientName string `json:"recipientName"`
	IssuerName    string `json:"issuerName"`
	Description   string `json:"description"`
	IssuedAt      string `json:"issuedAt"` // RFC3339
	ChainID       int64  `json:"chainId"`
	TxHash        string `json:"txHash"`
}

// NewWebhookEvent builds the webhook JSON payload for a newly indexed certificate.
// Pure function -- O(1).
func NewWebhookEvent(c domain.Certificate) ([]byte, error) {
	evt := WebhookEvent{
		Event: webhookCertificateIssuedEvent,
		Data: WebhookEventData{
			TokenID:       c.TokenID,
			OwnerAddress:  c.Owner.String(),
			Title:         c.Title,
			RecipientName: c.RecipientName,
			IssuerName:    c.IssuerName,
			Description:   c.Description,
			IssuedAt:      c.IssuedAt.UTC().Format(time.RFC3339),
			ChainID:       c.ChainID,
			TxHash:        c.TxHash,
		},
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("usecase.NewWebhookEvent: %w", err)
	}
	return payload, nil
}
