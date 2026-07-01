package usecase

import (
	"context"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
)

// CertificatePage is one keyset page of results.
type CertificatePage struct {
	Items      []domain.Certificate
	NextCursor string // token_id of the last item; "" when no more
	HasMore    bool
}

// ListParams drives ListCertificates.
// Empty Owner = all; empty Query = no text search.
type ListParams struct {
	Owner  string // optional owner_address filter (normalized by adapter)
	Query  string // optional ILIKE search over title/issuer/recipient
	Cursor string // keyset cursor (token_id); "" = first page
	Limit  int    // page size; <=0 → adapter applies a sane default
}

// CertificateRepo is the read-model + state store (Postgres adapter implements in T4).
type CertificateRepo interface {
	// Upsert inserts or updates a certificate; idempotent on token_id.
	Upsert(ctx context.Context, c domain.Certificate) error

	// GetByTokenID returns a certificate by token_id, or domain.ErrNotFound.
	GetByTokenID(ctx context.Context, tokenID string) (domain.Certificate, error)

	// List returns a keyset page — O(log n + limit) via index scan.
	List(ctx context.Context, p ListParams) (CertificatePage, error)

	// Count returns the total number of indexed certificates.
	Count(ctx context.Context) (int64, error)

	// GetState retrieves the singleton indexer checkpoint.
	GetState(ctx context.Context) (domain.IndexerState, error)

	// SaveState persists the indexer checkpoint.
	SaveState(ctx context.Context, s domain.IndexerState) error

	// DeleteFromBlock removes all certificates at or above blockNumber for chainID —
	// used by reorg reconcile to roll back the confirmation window.
	DeleteFromBlock(ctx context.Context, chainID int64, blockNumber uint64) error

	// GetIssuanceTrend returns raw (non-zero-filled) trend rows since the given time,
	// bucketed by day/week/month. O(certs in range) via the issued_at index.
	GetIssuanceTrend(ctx context.Context, bucket TrendBucket, since time.Time) ([]TrendPoint, error)
}

// EventSource is the chain read side (ethclient adapter implements in T5).
type EventSource interface {
	// HeadBlock returns the current chain head block number.
	HeadBlock(ctx context.Context) (uint64, error)

	// BlockHash returns the canonical header hash of the given block number.
	BlockHash(ctx context.Context, blockNumber uint64) (string, error)

	// IssuedLogs returns CertificateIssued logs in [fromBlock, toBlock] inclusive.
	IssuedLogs(ctx context.Context, fromBlock, toBlock uint64) ([]domain.IssuedLog, error)

	// GetCertificate backfills the full on-chain certificate struct via eth_call getCertificate(tokenId).
	GetCertificate(ctx context.Context, tokenID string) (domain.OnchainCertificate, error)
}

// EventPublisher notifies live subscribers when a certificate is indexed. Optional — the
// Worker is nil-safe if none is wired.
type EventPublisher interface {
	Publish(c domain.Certificate)
}

// EventSubscriber lets the gRPC adapter subscribe to live indexed-certificate events
// (implemented by the in-process broadcaster in platform/broadcast).
type EventSubscriber interface {
	// Subscribe registers a new subscriber and returns its channel plus an unsubscribe func.
	// Callers MUST call unsubscribe (e.g. via defer) to avoid a goroutine/channel leak.
	Subscribe() (<-chan domain.Certificate, func())
}

// TrendCache lets TrendService cache computed trend results (Phase 6 wires a Redis-backed
// implementation; TrendService is nil-safe if none is set).
type TrendCache interface {
	Get(ctx context.Context, key string) ([]TrendPoint, bool, error)
	Set(ctx context.Context, key string, points []TrendPoint) error
}

// TrendRefreshTaskType identifies the asynq task that recomputes and caches every
// supported trend bucket/preset combination. Defined here (not in the asynq adapter) so
// Worker can reference it without importing adapter code.
const TrendRefreshTaskType = "trend:refresh"

// TaskEnqueuer lets the Worker trigger background jobs after ingest. Optional — the Worker
// is nil-safe if none is wired.
type TaskEnqueuer interface {
	// EnqueueUnique enqueues a task, deduped by taskID: a second call with the same taskID
	// while one is still pending/processing is a no-op.
	EnqueueUnique(ctx context.Context, taskType, taskID string) error
}
