// Package asynqjobs holds the asynq task definitions and handlers for the indexer's
// background jobs (currently just the trend-cache refresh).
package asynqjobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// trendRefresher is the subset of *usecase.TrendService this handler needs — a narrow
// interface makes the handler testable without a real Postgres/Redis-backed TrendService.
type trendRefresher interface {
	RefreshCache(ctx context.Context, bucket usecase.TrendBucket, since time.Time, preset string) ([]usecase.TrendPoint, error)
}

// NewRefreshTrendCacheTask builds the (payload-less) refresh task — recompute-all needs no
// parameters, so every enqueue of this type is identical, which is exactly what makes
// asynq.Unique-based deduplication (see the asynqjobs.Enqueuer) work.
func NewRefreshTrendCacheTask() *asynq.Task {
	return asynq.NewTask(usecase.TrendRefreshTaskType, nil, asynq.MaxRetry(2))
}

// RefreshTrendCacheHandler recomputes every supported bucket/range-preset combination and
// writes each into the cache via TrendService.RefreshCache. Registered against
// usecase.TrendRefreshTaskType in the asynq ServeMux.
type RefreshTrendCacheHandler struct {
	trend trendRefresher
	log   *slog.Logger
}

// NewRefreshTrendCacheHandler constructs a RefreshTrendCacheHandler.
func NewRefreshTrendCacheHandler(trend trendRefresher, log *slog.Logger) *RefreshTrendCacheHandler {
	if log == nil {
		log = slog.Default()
	}
	return &RefreshTrendCacheHandler{trend: trend, log: log}
}

// ProcessTask satisfies asynq.Handler. O(number of bucket/preset combinations) — a small,
// fixed table (7 combinations as of Phase 5: day 30d/90d/365d, week 12w/52w, month 12m/24m),
// not proportional to certificate count.
func (h *RefreshTrendCacheHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	now := time.Now()
	for bucket, presets := range usecase.AllowedPresets() {
		for preset := range presets {
			since, err := usecase.RangePresetToSince(bucket, preset, now)
			if err != nil {
				return fmt.Errorf("range preset bucket=%d preset=%s: %w", bucket, preset, err)
			}
			if _, err := h.trend.RefreshCache(ctx, bucket, since, preset); err != nil {
				h.log.Error("refresh trend cache", "bucket", bucket, "preset", preset, "err", err)
				return fmt.Errorf("refresh bucket=%d preset=%s: %w", bucket, preset, err)
			}
		}
	}
	return nil
}
