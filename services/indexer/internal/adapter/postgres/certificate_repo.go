// Package postgres implements usecase.CertificateRepo over Postgres via sqlc + pgxpool.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oksasatya/skillpass/services/indexer/internal/db"
	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// compile-time port check
var _ usecase.CertificateRepo = (*CertificateRepo)(nil)

const (
	defaultLimit = 20
	maxLimit     = 100

	errGetIssuanceTrendWrap = "postgres.CertificateRepo.GetIssuanceTrend: %w"
)

// CertificateRepo satisfies usecase.CertificateRepo via Postgres.
type CertificateRepo struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewCertificateRepo constructs a CertificateRepo backed by the given pool.
func NewCertificateRepo(pool *pgxpool.Pool) *CertificateRepo {
	return &CertificateRepo{pool: pool, queries: db.New(pool)}
}

// Upsert inserts or updates a certificate; idempotent on token_id.
func (r *CertificateRepo) Upsert(ctx context.Context, c domain.Certificate) error {
	_, err := r.queries.UpsertCertificate(ctx, db.UpsertCertificateParams{
		TokenID:       c.TokenID,
		OwnerAddress:  c.Owner.String(),
		Title:         c.Title,
		RecipientName: c.RecipientName,
		IssuerName:    c.IssuerName,
		Description:   c.Description,
		MetadataUri:   c.MetadataURI,
		IssuedAt:      pgtype.Timestamptz{Time: c.IssuedAt.UTC(), Valid: true},
		ChainID:       c.ChainID,
		TxHash:        c.TxHash,
		LogIndex:      c.LogIndex,
		BlockNumber:   c.BlockNumber,
		BlockHash:     c.BlockHash,
	})
	if err != nil {
		return fmt.Errorf("postgres.CertificateRepo.Upsert: %w", err)
	}
	return nil
}

// GetByTokenID returns a certificate or domain.ErrNotFound.
func (r *CertificateRepo) GetByTokenID(ctx context.Context, tokenID string) (domain.Certificate, error) {
	row, err := r.queries.GetCertificateByTokenID(ctx, tokenID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Certificate{}, fmt.Errorf("%w: token_id %s", domain.ErrNotFound, tokenID)
		}
		return domain.Certificate{}, fmt.Errorf("postgres.CertificateRepo.GetByTokenID: %w", err)
	}
	return toDomain(row)
}

// List returns a keyset page — O(log n + limit) via index scan.
func (r *CertificateRepo) List(ctx context.Context, p usecase.ListParams) (usecase.CertificatePage, error) {
	owner, err := normalizeOwner(p.Owner)
	if err != nil {
		// bad filter: return empty page (caller's mistake, not a server error)
		return usecase.CertificatePage{}, nil
	}

	limit := resolveLimit(p.Limit)
	cursor, err := parseCursor(p.Cursor)
	if err != nil {
		return usecase.CertificatePage{}, fmt.Errorf("postgres.CertificateRepo.List: cursor: %w", err)
	}

	rows, err := r.dispatch(ctx, owner, p.Query, cursor, int32(limit+1))
	if err != nil {
		return usecase.CertificatePage{}, fmt.Errorf("postgres.CertificateRepo.List: %w", err)
	}

	return buildPage(rows, limit)
}

// Count returns the total number of indexed certificates.
func (r *CertificateRepo) Count(ctx context.Context) (int64, error) {
	n, err := r.queries.CountCertificates(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres.CertificateRepo.Count: %w", err)
	}
	return n, nil
}

// GetState retrieves the singleton indexer checkpoint.
// On no-rows (cold start) it returns a zero IndexerState, nil.
func (r *CertificateRepo) GetState(ctx context.Context) (domain.IndexerState, error) {
	row, err := r.queries.GetIndexerState(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.IndexerState{}, nil
		}
		return domain.IndexerState{}, fmt.Errorf("postgres.CertificateRepo.GetState: %w", err)
	}
	return domain.IndexerState{
		ChainID:            row.ChainID,
		LastProcessedBlock: uint64(row.LastProcessedBlock),
		LastProcessedHash:  row.LastProcessedHash,
	}, nil
}

// SaveState persists the indexer checkpoint.
func (r *CertificateRepo) SaveState(ctx context.Context, s domain.IndexerState) error {
	_, err := r.queries.UpsertIndexerState(ctx, db.UpsertIndexerStateParams{
		ChainID:            s.ChainID,
		LastProcessedBlock: int64(s.LastProcessedBlock),
		LastProcessedHash:  s.LastProcessedHash,
	})
	if err != nil {
		return fmt.Errorf("postgres.CertificateRepo.SaveState: %w", err)
	}
	return nil
}

// DeleteFromBlock removes all certificates at or above blockNumber for chainID —
// O(k) rows deleted, where k is the number of certificates in the reorg window
// (bounded by the 12-block confirmation depth, small at any realistic scale).
func (r *CertificateRepo) DeleteFromBlock(ctx context.Context, chainID int64, blockNumber uint64) error {
	err := r.queries.DeleteCertificatesFromBlock(ctx, db.DeleteCertificatesFromBlockParams{
		ChainID:     chainID,
		BlockNumber: int64(blockNumber), //nolint:gosec // blockNumber realistically << max int64
	})
	if err != nil {
		return fmt.Errorf("postgres.CertificateRepo.DeleteFromBlock: %w", err)
	}
	return nil
}

// GetIssuanceTrend returns raw (non-zero-filled) trend rows since the given time, dispatched
// by bucket granularity — mirrors the dispatch() switch pattern used by List/Search above.
// O(certs in range) via the issued_at index, then a Postgres GROUP BY aggregate.
func (r *CertificateRepo) GetIssuanceTrend(ctx context.Context, bucket usecase.TrendBucket, since time.Time) ([]usecase.TrendPoint, error) {
	sinceArg := pgtype.Timestamptz{Time: since.UTC(), Valid: true}

	switch bucket {
	case usecase.TrendBucketDay:
		rows, err := r.queries.TrendByDay(ctx, sinceArg)
		if err != nil {
			return nil, fmt.Errorf(errGetIssuanceTrendWrap, err)
		}
		return toTrendPoints(rows), nil
	case usecase.TrendBucketWeek:
		rows, err := r.queries.TrendByWeek(ctx, sinceArg)
		if err != nil {
			return nil, fmt.Errorf(errGetIssuanceTrendWrap, err)
		}
		return toTrendPoints(rows), nil
	case usecase.TrendBucketMonth:
		rows, err := r.queries.TrendByMonth(ctx, sinceArg)
		if err != nil {
			return nil, fmt.Errorf(errGetIssuanceTrendWrap, err)
		}
		return toTrendPoints(rows), nil
	default:
		return nil, fmt.Errorf("postgres.CertificateRepo.GetIssuanceTrend: unknown bucket %d", bucket)
	}
}

// trendRow is satisfied by every sqlc-generated TrendByXRow (structurally identical,
// distinct generated types — Go generics bridge them without repeating the mapping 3x).
type trendRow interface {
	db.TrendByDayRow | db.TrendByWeekRow | db.TrendByMonthRow
}

func toTrendPoints[R trendRow](rows []R) []usecase.TrendPoint {
	points := make([]usecase.TrendPoint, 0, len(rows))
	for _, row := range rows {
		switch v := any(row).(type) {
		case db.TrendByDayRow:
			points = append(points, usecase.TrendPoint{BucketStart: v.BucketStart.Time, Count: v.Cnt})
		case db.TrendByWeekRow:
			points = append(points, usecase.TrendPoint{BucketStart: v.BucketStart.Time, Count: v.Cnt})
		case db.TrendByMonthRow:
			points = append(points, usecase.TrendPoint{BucketStart: v.BucketStart.Time, Count: v.Cnt})
		}
	}
	return points
}

// --- helpers ---

// normalizeOwner lowercases and validates the owner filter.
// Returns ("", nil) when owner is empty (no filter).
func normalizeOwner(owner string) (string, error) {
	if owner == "" {
		return "", nil
	}
	addr, err := domain.NewAddress(owner)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}

// resolveLimit applies the default and cap.
func resolveLimit(n int) int {
	if n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

// parseCursor converts a keyset cursor string to pgtype.Numeric.
// Empty string → Invalid (SQL NULL branch = first page).
func parseCursor(s string) (pgtype.Numeric, error) {
	if s == "" {
		return pgtype.Numeric{Valid: false}, nil
	}
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("invalid cursor %q: %w", s, err)
	}
	return n, nil
}

// dispatch routes to the index-optimal query based on which filters are set.
// O(log n + limit) keyset via (owner_address, token_id DESC) / PK index — never OFFSET.
func (r *CertificateRepo) dispatch(
	ctx context.Context,
	owner, query string,
	cursor pgtype.Numeric,
	limit int32,
) ([]db.Certificate, error) {
	switch {
	case owner == "" && query == "":
		return r.queries.ListCertificates(ctx, db.ListCertificatesParams{
			Column1: cursor,
			Limit:   limit,
		})
	case owner != "" && query == "":
		return r.queries.ListCertificatesByOwner(ctx, db.ListCertificatesByOwnerParams{
			OwnerAddress: owner,
			Column2:      cursor,
			Limit:        limit,
		})
	case owner == "" && query != "":
		return r.queries.SearchCertificates(ctx, db.SearchCertificatesParams{
			Column1: pgtype.Text{String: query, Valid: true},
			Column2: cursor,
			Limit:   limit,
		})
	default: // owner != "" && query != ""
		return r.queries.SearchCertificatesByOwner(ctx, db.SearchCertificatesByOwnerParams{
			OwnerAddress: owner,
			Column2:      pgtype.Text{String: query, Valid: true},
			Column3:      cursor,
			Limit:        limit,
		})
	}
}

// buildPage slices rows using the limit+1 trick to detect HasMore.
func buildPage(rows []db.Certificate, limit int) (usecase.CertificatePage, error) {
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	items := make([]domain.Certificate, 0, len(rows))
	for _, row := range rows {
		c, err := toDomain(row)
		if err != nil {
			return usecase.CertificatePage{}, err
		}
		items = append(items, c)
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].TokenID
	}

	return usecase.CertificatePage{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// toDomain maps a db.Certificate row to domain.Certificate.
func toDomain(row db.Certificate) (domain.Certificate, error) {
	owner, err := domain.NewAddress(row.OwnerAddress)
	if err != nil {
		return domain.Certificate{}, fmt.Errorf("toDomain: invalid owner_address %q: %w", row.OwnerAddress, err)
	}

	var issuedAt time.Time
	if row.IssuedAt.Valid {
		issuedAt = row.IssuedAt.Time
	}

	return domain.Certificate{
		TokenID:       row.TokenID,
		Owner:         owner,
		Title:         row.Title,
		RecipientName: row.RecipientName,
		IssuerName:    row.IssuerName,
		Description:   row.Description,
		MetadataURI:   row.MetadataUri,
		IssuedAt:      issuedAt,
		ChainID:       row.ChainID,
		TxHash:        row.TxHash,
		LogIndex:      row.LogIndex,
		BlockNumber:   row.BlockNumber,
		BlockHash:     row.BlockHash,
	}, nil
}
