package domain

import (
	"fmt"
	"time"
)

// Certificate is the canonical indexed certificate (read model + verification proof).
type Certificate struct {
	// Core identity
	TokenID       string
	Owner         Address
	Title         string
	RecipientName string // optional on-chain
	IssuerName    string
	Description   string // optional on-chain
	MetadataURI   string // optional on-chain
	IssuedAt      time.Time

	// On-chain provenance
	ChainID     int64
	TxHash      string
	LogIndex    int64
	BlockNumber int64
	BlockHash   string
}

// Validate enforces Certificate invariants.
// Returns ErrInvalidCertificate (wrapping) on any violation.
func (c Certificate) Validate() error {
	if err := ValidateTokenID(c.TokenID); err != nil {
		return fmt.Errorf("%w: token_id: %v", ErrInvalidCertificate, err)
	}
	if c.Owner.IsZero() {
		return fmt.Errorf("%w: owner must not be zero address", ErrInvalidCertificate)
	}
	if c.Title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidCertificate)
	}
	if c.IssuerName == "" {
		return fmt.Errorf("%w: issuer_name is required", ErrInvalidCertificate)
	}
	if c.IssuedAt.IsZero() {
		return fmt.Errorf("%w: issued_at must not be zero", ErrInvalidCertificate)
	}
	if c.TxHash == "" {
		return fmt.Errorf("%w: tx_hash is required", ErrInvalidCertificate)
	}
	if c.BlockHash == "" {
		return fmt.Errorf("%w: block_hash is required", ErrInvalidCertificate)
	}
	if c.BlockNumber < 0 {
		return fmt.Errorf("%w: block_number must be >= 0", ErrInvalidCertificate)
	}
	return nil
}

// IssuedLog is decoded from the CertificateIssued event log (chain adapter produces this).
type IssuedLog struct {
	TokenID     string
	BlockNumber uint64
	BlockHash   string
	TxHash      string
	LogIndex    uint
}

// OnchainCertificate is returned by getCertificate(tokenId) eth_call backfill.
type OnchainCertificate struct {
	TokenID       string
	Owner         Address
	Title         string
	RecipientName string
	IssuerName    string
	Description   string
	MetadataURI   string
	IssuedAt      time.Time
}

// NewIndexedCertificate merges event provenance (log) with on-chain data (data)
// into a validated Certificate. Pure function — O(1).
func NewIndexedCertificate(log IssuedLog, data OnchainCertificate, chainID int64) (Certificate, error) {
	c := Certificate{
		TokenID:       log.TokenID,
		Owner:         data.Owner,
		Title:         data.Title,
		RecipientName: data.RecipientName,
		IssuerName:    data.IssuerName,
		Description:   data.Description,
		MetadataURI:   data.MetadataURI,
		IssuedAt:      data.IssuedAt,
		ChainID:       chainID,
		TxHash:        log.TxHash,
		LogIndex:      int64(log.LogIndex),
		BlockNumber:   int64(log.BlockNumber),
		BlockHash:     log.BlockHash,
	}
	if err := c.Validate(); err != nil {
		return Certificate{}, err
	}
	return c, nil
}

// IndexerState records the indexer's progress checkpoint.
type IndexerState struct {
	ChainID            int64
	LastProcessedBlock uint64
	LastProcessedHash  string
}
