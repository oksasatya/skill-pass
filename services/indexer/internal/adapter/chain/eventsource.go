// Package chain implements usecase.EventSource via go-ethereum ethclient.
package chain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/chain/binding"
	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// Compile-time assertion: EventSource satisfies the port.
var _ usecase.EventSource = (*EventSource)(nil)

// EventSource adapts go-ethereum/ethclient to the usecase.EventSource port.
// It only reads from the chain — no signing, no transactions.
type EventSource struct {
	client   *ethclient.Client
	contract *binding.SkillPassCertificate
	addr     common.Address
}

// NewEventSource dials the RPC endpoint and binds the contract at contractAddr.
func NewEventSource(ctx context.Context, rpcURL, contractAddr string) (*EventSource, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", rpcURL, err)
	}
	addr := common.HexToAddress(contractAddr)
	contract, err := binding.NewSkillPassCertificate(addr, client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("bind contract at %s: %w", contractAddr, err)
	}
	return &EventSource{client: client, contract: contract, addr: addr}, nil
}

// Close releases the underlying RPC connection.
func (e *EventSource) Close() {
	e.client.Close()
}

// HeadBlock returns the current chain head block number.
func (e *EventSource) HeadBlock(ctx context.Context) (uint64, error) {
	n, err := e.client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("block number: %w", err)
	}
	return n, nil
}

// BlockHash returns the canonical header hash of the given block number.
func (e *EventSource) BlockHash(ctx context.Context, blockNumber uint64) (string, error) {
	header, err := e.client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return "", fmt.Errorf("header by number %d: %w", blockNumber, err)
	}
	return header.Hash().Hex(), nil
}

// IssuedLogs returns CertificateIssued event logs in [fromBlock, toBlock] inclusive.
// Only tokenId + provenance fields are extracted; the full cert is fetched via GetCertificate.
func (e *EventSource) IssuedLogs(ctx context.Context, fromBlock, toBlock uint64) ([]domain.IssuedLog, error) {
	opts := &bind.FilterOpts{
		Start:   fromBlock,
		End:     &toBlock,
		Context: ctx,
	}
	iter, err := e.contract.FilterCertificateIssued(opts, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("filter CertificateIssued [%d,%d]: %w", fromBlock, toBlock, err)
	}
	defer iter.Close()

	var logs []domain.IssuedLog
	for iter.Next() {
		ev := iter.Event
		logs = append(logs, domain.IssuedLog{
			TokenID:     ev.TokenId.String(),
			BlockNumber: ev.Raw.BlockNumber,
			BlockHash:   ev.Raw.BlockHash.Hex(),
			TxHash:      ev.Raw.TxHash.Hex(),
			LogIndex:    uint(ev.Raw.Index),
		})
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterate CertificateIssued logs: %w", err)
	}
	return logs, nil
}

// GetCertificate calls getCertificate(tokenId) on-chain and maps the result to domain.OnchainCertificate.
// tokenID must be a valid decimal string representation of a uint256.
func (e *EventSource) GetCertificate(ctx context.Context, tokenID string) (domain.OnchainCertificate, error) {
	id, ok := new(big.Int).SetString(tokenID, 10)
	if !ok {
		return domain.OnchainCertificate{}, fmt.Errorf("parse tokenID %q: %w", tokenID, domain.ErrInvalidTokenID)
	}

	result, err := e.contract.GetCertificate(&bind.CallOpts{Context: ctx}, id)
	if err != nil {
		return domain.OnchainCertificate{}, fmt.Errorf("getCertificate(%s): %w", tokenID, err)
	}

	owner, err := domain.NewAddress(result.Recipient.Hex())
	if err != nil {
		return domain.OnchainCertificate{}, fmt.Errorf("normalize owner address for token %s: %w", tokenID, err)
	}

	return domain.OnchainCertificate{
		TokenID:       tokenID,
		Owner:         owner,
		Title:         result.Cert.Title,
		RecipientName: result.Cert.RecipientName,
		IssuerName:    result.Cert.IssuerName,
		Description:   result.Cert.Description,
		MetadataURI:   result.Cert.MetadataURI,
		IssuedAt:      time.Unix(result.Cert.IssuedAt.Int64(), 0).UTC(),
	}, nil
}
