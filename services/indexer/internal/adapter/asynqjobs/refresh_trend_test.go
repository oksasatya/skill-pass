package asynqjobs_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/asynqjobs"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// fakeTrendRefresher records every (bucket, since, preset) combination it's asked to refresh.
type fakeTrendRefresher struct {
	calls []string
}

func (f *fakeTrendRefresher) RefreshCache(_ context.Context, bucket usecase.TrendBucket, _ time.Time, preset string) ([]usecase.TrendPoint, error) {
	f.calls = append(f.calls, preset)
	_ = bucket
	return nil, nil
}

func TestRefreshTrendCacheHandler_RefreshesEveryPreset(t *testing.T) {
	refresher := &fakeTrendRefresher{}
	h := asynqjobs.NewRefreshTrendCacheHandler(refresher, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := h.ProcessTask(context.Background(), asynqjobs.NewRefreshTrendCacheTask()); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}

	wantCount := 0
	for _, presets := range usecase.AllowedPresets() {
		wantCount += len(presets)
	}
	if len(refresher.calls) != wantCount {
		t.Fatalf("refreshed %d combinations, want %d", len(refresher.calls), wantCount)
	}
}

func TestNewRefreshTrendCacheTask_HasCorrectType(t *testing.T) {
	task := asynqjobs.NewRefreshTrendCacheTask()
	if task.Type() != usecase.TrendRefreshTaskType {
		t.Fatalf("task type = %q, want %q", task.Type(), usecase.TrendRefreshTaskType)
	}
}

var _ asynq.Handler = (*asynqjobs.RefreshTrendCacheHandler)(nil)
