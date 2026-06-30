package usecase

import (
	"context"

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
}

// EventSource is the chain read side (ethclient adapter implements in T5).
type EventSource interface {
	// HeadBlock returns the current chain head block number.
	HeadBlock(ctx context.Context) (uint64, error)

	// IssuedLogs returns CertificateIssued logs in [fromBlock, toBlock] inclusive.
	IssuedLogs(ctx context.Context, fromBlock, toBlock uint64) ([]domain.IssuedLog, error)

	// GetCertificate backfills the full on-chain certificate struct via eth_call getCertificate(tokenId).
	GetCertificate(ctx context.Context, tokenID string) (domain.OnchainCertificate, error)
}
