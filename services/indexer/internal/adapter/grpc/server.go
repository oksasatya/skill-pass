// Package grpc is the inbound adapter: maps usecase ports to the certv1 gRPC API.
package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// Compile-time assertion: Server must satisfy the interface.
var _ certv1.CertificateQueryServer = (*Server)(nil)

const (
	errEmptyTokenID = "token_id must not be empty"
	errNotFound     = "certificate not found"
	errInternal     = "internal server error"
)

// indexLagSaneLimit is the max lag before healthy=false (MVP threshold).
// ponytail: hard-coded threshold; expose as config param if tuning is needed
const indexLagSaneLimit uint64 = 500

// Server implements certv1.CertificateQueryServer over the read-model ports.
type Server struct {
	repo usecase.CertificateRepo
	src  usecase.EventSource
	log  *slog.Logger
}

// NewServer constructs a Server. Both repo and src must be non-nil.
func NewServer(repo usecase.CertificateRepo, src usecase.EventSource, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{repo: repo, src: src, log: log}
}

// GetCertificate returns a single certificate by token_id.
func (s *Server) GetCertificate(ctx context.Context, req *certv1.GetCertificateRequest) (*certv1.GetCertificateResponse, error) {
	if req.GetTokenId() == "" {
		return nil, status.Error(codes.InvalidArgument, errEmptyTokenID)
	}

	cert, err := s.repo.GetByTokenID(ctx, req.GetTokenId())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, errNotFound)
		}
		s.log.Error("GetCertificate: repo", "token_id", req.GetTokenId(), "err", err)
		return nil, status.Error(codes.Internal, errInternal)
	}

	return &certv1.GetCertificateResponse{Certificate: toProto(cert)}, nil
}

// ListCertificates returns a keyset-paginated list of certificates.
func (s *Server) ListCertificates(ctx context.Context, req *certv1.ListCertificatesRequest) (*certv1.ListCertificatesResponse, error) {
	page, err := s.repo.List(ctx, usecase.ListParams{
		Owner:  req.GetOwnerAddress(),
		Query:  req.GetQuery(),
		Cursor: req.GetCursor(),
		Limit:  int(req.GetPageSize()),
	})
	if err != nil {
		s.log.Error("ListCertificates: repo", "err", err)
		return nil, status.Error(codes.Internal, errInternal)
	}

	protos := make([]*certv1.Certificate, 0, len(page.Items))
	for _, c := range page.Items {
		protos = append(protos, toProto(c))
	}

	return &certv1.ListCertificatesResponse{
		Certificates: protos,
		NextCursor:   page.NextCursor,
		HasMore:      page.HasMore,
	}, nil
}

// GetIndexerStatus returns the current indexer progress and health.
// It degrades gracefully when the chain is unreachable: healthy=false, chain_head_block=0.
func (s *Server) GetIndexerStatus(ctx context.Context, _ *certv1.GetIndexerStatusRequest) (*certv1.GetIndexerStatusResponse, error) {
	state, err := s.repo.GetState(ctx)
	if err != nil {
		s.log.Error("GetIndexerStatus: GetState", "err", err)
		return nil, status.Error(codes.Internal, errInternal)
	}

	count, err := s.repo.Count(ctx)
	if err != nil {
		s.log.Error("GetIndexerStatus: Count", "err", err)
		return nil, status.Error(codes.Internal, errInternal)
	}

	// ponytail: SELECT count(*) is O(n); reltuples approximation if this becomes a hot path
	head, headErr := s.src.HeadBlock(ctx)
	chainReachable := headErr == nil
	if headErr != nil {
		s.log.Warn("GetIndexerStatus: HeadBlock degraded", "err", headErr)
	}

	lag, healthy := computeStatus(state.LastProcessedBlock, head, chainReachable)

	return &certv1.GetIndexerStatusResponse{
		LastProcessedBlock: state.LastProcessedBlock,
		ChainHeadBlock:     head,
		IndexLag:           lag,
		Healthy:            healthy,
		TotalCertificates:  uint64(count), //nolint:gosec // count is always non-negative
	}, nil
}

// StreamCertificateEvents is a BE-2 seam — real server-streaming lands in BE-2 Task 3.
// ponytail: stub returns Unimplemented; the gateway SSE bridge is wired in BE-2
func (s *Server) StreamCertificateEvents(_ *certv1.StreamCertificateEventsRequest, _ grpc.ServerStreamingServer[certv1.CertificateEvent]) error {
	return status.Error(codes.Unimplemented, "streaming lands in BE-2")
}

// --- helpers ---

// toProto maps a domain.Certificate to its proto representation.
func toProto(c domain.Certificate) *certv1.Certificate {
	return &certv1.Certificate{
		TokenId:       c.TokenID,
		OwnerAddress:  c.Owner.String(),
		Title:         c.Title,
		RecipientName: c.RecipientName,
		IssuerName:    c.IssuerName,
		Description:   c.Description,
		MetadataUri:   c.MetadataURI,
		IssuedAt:      timestamppb.New(c.IssuedAt),
		TxHash:        c.TxHash,
		BlockNumber:   uint64(c.BlockNumber), //nolint:gosec // BlockNumber >= 0 enforced by domain.Validate
	}
}

// computeStatus derives lag and healthy from the indexer state and chain head.
// Extracted to keep GetIndexerStatus under cognitive-complexity budget.
func computeStatus(lastProcessed, head uint64, chainReachable bool) (lag uint64, healthy bool) {
	if !chainReachable {
		return 0, false
	}
	if lastProcessed > head {
		lag = 0 // guard underflow (can happen on resets)
	} else {
		lag = head - lastProcessed
	}
	healthy = lag < indexLagSaneLimit
	return lag, healthy
}
