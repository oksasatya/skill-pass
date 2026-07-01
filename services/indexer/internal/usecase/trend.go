package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TrendBucket is the aggregation granularity for GetIssuanceTrend.
type TrendBucket int

const (
	TrendBucketDay TrendBucket = iota + 1
	TrendBucketWeek
	TrendBucketMonth
)

// TrendPoint is one bucketed count in a certificate-issuance trend.
type TrendPoint struct {
	BucketStart time.Time
	Count       int64
}

// ErrInvalidTrendRequest is returned for an unknown bucket or a preset not supported for it.
var ErrInvalidTrendRequest = errors.New("invalid trend request")

// allowedPresets maps each bucket to its supported range presets and the number of days
// each preset covers. Bounded on purpose (see the Phase 5 design spec): an unbounded
// client-supplied range can't be validated cheaply or cached in Phase 6.
var allowedPresets = map[TrendBucket]map[string]int{
	TrendBucketDay:   {"30d": 30, "90d": 90, "365d": 365},
	TrendBucketWeek:  {"12w": 12 * 7, "52w": 52 * 7},
	TrendBucketMonth: {"12m": 12 * 31, "24m": 24 * 31}, // approximate; bucket boundaries are exact via date_trunc
}

// AllowedPresets returns the full bucket -> preset -> days table (used by the Phase 6 refresh
// job to enumerate every combination to precompute).
func AllowedPresets() map[TrendBucket]map[string]int {
	return allowedPresets
}

// RangePresetToSince validates preset against bucket and returns the UTC "since" timestamp
// (now minus the preset's day count). now is injected for testability.
func RangePresetToSince(bucket TrendBucket, preset string, now time.Time) (time.Time, error) {
	presets, ok := allowedPresets[bucket]
	if !ok {
		return time.Time{}, fmt.Errorf("%w: unknown bucket %d", ErrInvalidTrendRequest, bucket)
	}
	days, ok := presets[preset]
	if !ok {
		return time.Time{}, fmt.Errorf("%w: preset %q not supported for this bucket", ErrInvalidTrendRequest, preset)
	}
	return now.UTC().AddDate(0, 0, -days), nil
}

// AlignedBuckets returns every expected bucket-start timestamp from since to now (inclusive),
// aligned to UTC day/week(Monday)/month boundaries. O(number of buckets in range) — bounded
// by the preset table above (at most 365 for the day bucket).
func AlignedBuckets(bucket TrendBucket, since, now time.Time) []time.Time {
	cur := truncateToBucket(bucket, since)
	end := truncateToBucket(bucket, now)

	var out []time.Time
	for !cur.After(end) {
		out = append(out, cur)
		cur = advanceBucket(bucket, cur)
	}
	return out
}

func truncateToBucket(bucket TrendBucket, t time.Time) time.Time {
	t = t.UTC()
	switch bucket {
	case TrendBucketWeek:
		offset := (int(t.Weekday()) + 6) % 7 // Monday = 0
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -offset)
	case TrendBucketMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	default: // TrendBucketDay
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
}

func advanceBucket(bucket TrendBucket, t time.Time) time.Time {
	switch bucket {
	case TrendBucketWeek:
		return t.AddDate(0, 0, 7)
	case TrendBucketMonth:
		return t.AddDate(0, 1, 0)
	default:
		return t.AddDate(0, 0, 1)
	}
}

// ZeroFillTrend merges DB rows (which only contain buckets with >=1 certificate) into the
// full aligned bucket list, filling zero-count for any missing bucket.
// O(len(expected) + len(rows)) time and O(len(rows)) space — a hash map keyed by bucket
// avoids an O(len(expected) * len(rows)) nested scan.
func ZeroFillTrend(expected []time.Time, rows []TrendPoint) []TrendPoint {
	byBucket := make(map[int64]int64, len(rows))
	for _, r := range rows {
		byBucket[r.BucketStart.Unix()] = r.Count
	}
	out := make([]TrendPoint, 0, len(expected))
	for _, t := range expected {
		out = append(out, TrendPoint{BucketStart: t, Count: byBucket[t.Unix()]})
	}
	return out
}

// TrendService orchestrates CertificateRepo (Postgres) and an optional TrendCache (Phase 6)
// to serve GetIssuanceTrend. Nil-safe on cache — a cold/absent cache just recomputes.
type TrendService struct {
	repo    CertificateRepo
	chainID int64
	cache   TrendCache // optional; nil-safe
}

// NewTrendService constructs a TrendService with no cache (Phase 5 default).
func NewTrendService(repo CertificateRepo, chainID int64) *TrendService {
	return &TrendService{repo: repo, chainID: chainID}
}

// SetCache wires an optional TrendCache. Call before serving traffic; safe to never call.
func (s *TrendService) SetCache(cache TrendCache) {
	s.cache = cache
}

// GetTrend returns the zero-filled trend for bucket/preset, since already validated by the
// caller (RangePresetToSince) — reads the cache first when one is set, computing on a miss.
func (s *TrendService) GetTrend(ctx context.Context, bucket TrendBucket, since time.Time, preset string) ([]TrendPoint, error) {
	if s.cache != nil {
		if points, ok, err := s.cache.Get(ctx, s.cacheKey(bucket, preset)); err == nil && ok {
			return points, nil
		}
	}
	return s.compute(ctx, bucket, since, preset)
}

// RefreshCache force-recomputes and writes to the cache, bypassing the cache-read path.
// Used by the Phase 6 asynq job, never by the read path.
func (s *TrendService) RefreshCache(ctx context.Context, bucket TrendBucket, since time.Time, preset string) ([]TrendPoint, error) {
	return s.compute(ctx, bucket, since, preset)
}

func (s *TrendService) compute(ctx context.Context, bucket TrendBucket, since time.Time, preset string) ([]TrendPoint, error) {
	rows, err := s.repo.GetIssuanceTrend(ctx, bucket, since)
	if err != nil {
		return nil, fmt.Errorf("get issuance trend: %w", err)
	}
	points := ZeroFillTrend(AlignedBuckets(bucket, since, time.Now()), rows)

	if s.cache != nil {
		// A cache-write failure must not fail the read — TrendService always returns correct
		// data regardless of cache health; the adapter logs its own errors if it wants to.
		_ = s.cache.Set(ctx, s.cacheKey(bucket, preset), points)
	}
	return points, nil
}

func (s *TrendService) cacheKey(bucket TrendBucket, preset string) string {
	return fmt.Sprintf("trend:v1:%d:%d:%s", s.chainID, bucket, preset)
}
