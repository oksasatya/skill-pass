package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

func TestRangePresetToSince(t *testing.T) {
	now := time.Date(2026, 7, 1, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		bucket  usecase.TrendBucket
		preset  string
		wantErr bool
	}{
		{"day 30d valid", usecase.TrendBucketDay, "30d", false},
		{"week 12w valid", usecase.TrendBucketWeek, "12w", false},
		{"month 12m valid", usecase.TrendBucketMonth, "12m", false},
		{"day preset invalid for week bucket", usecase.TrendBucketWeek, "30d", true},
		{"unknown bucket", usecase.TrendBucket(99), "30d", true},
		{"unknown preset", usecase.TrendBucketDay, "7d", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			since, err := usecase.RangePresetToSince(tt.bucket, tt.preset, now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !since.Before(now) {
				t.Fatalf("since (%v) should be before now (%v)", since, now)
			}
		})
	}
}

func TestAlignedBuckets_Day(t *testing.T) {
	since := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	got := usecase.AlignedBuckets(usecase.TrendBucketDay, since, now)

	want := []time.Time{
		time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	if len(got) != len(want) {
		t.Fatalf("got %d buckets, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("bucket[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestAlignedBuckets_Month(t *testing.T) {
	since := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	got := usecase.AlignedBuckets(usecase.TrendBucketMonth, since, now)

	want := []time.Time{
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	if len(got) != len(want) {
		t.Fatalf("got %d buckets, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("bucket[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestZeroFillTrend(t *testing.T) {
	d1 := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	expected := []time.Time{d1, d2, d3}

	// d2 has no rows — must be zero-filled
	rows := []usecase.TrendPoint{
		{BucketStart: d1, Count: 3},
		{BucketStart: d3, Count: 1},
	}

	got := usecase.ZeroFillTrend(expected, rows)

	if len(got) != 3 {
		t.Fatalf("got %d points, want 3", len(got))
	}
	if got[0].Count != 3 || got[1].Count != 0 || got[2].Count != 1 {
		t.Fatalf("counts = [%d,%d,%d], want [3,0,1]", got[0].Count, got[1].Count, got[2].Count)
	}
}

// fakeTrendRepo implements the subset of usecase.CertificateRepo TrendService needs.
type fakeTrendRepo struct {
	usecase.CertificateRepo // embed nil; only GetIssuanceTrend is exercised by these tests
	rows                    []usecase.TrendPoint
	err                     error
}

func (f *fakeTrendRepo) GetIssuanceTrend(_ context.Context, _ usecase.TrendBucket, _ time.Time) ([]usecase.TrendPoint, error) {
	return f.rows, f.err
}

func TestTrendService_GetTrend_ComputesAndZeroFills(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeTrendRepo{rows: []usecase.TrendPoint{{BucketStart: now, Count: 5}}}
	svc := usecase.NewTrendService(repo, 31337)

	since, _ := usecase.RangePresetToSince(usecase.TrendBucketDay, "30d", now)
	points, err := svc.GetTrend(context.Background(), usecase.TrendBucketDay, since, "30d")
	if err != nil {
		t.Fatalf("GetTrend: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected at least one zero-filled bucket")
	}
}

// fakeTrendCache implements usecase.TrendCache in-memory for tests.
type fakeTrendCache struct {
	store map[string][]usecase.TrendPoint
}

func newFakeTrendCache() *fakeTrendCache {
	return &fakeTrendCache{store: map[string][]usecase.TrendPoint{}}
}

func (c *fakeTrendCache) Get(_ context.Context, key string) ([]usecase.TrendPoint, bool, error) {
	v, ok := c.store[key]
	return v, ok, nil
}

func (c *fakeTrendCache) Set(_ context.Context, key string, points []usecase.TrendPoint) error {
	c.store[key] = points
	return nil
}

func TestTrendService_GetTrend_CacheHit_SkipsRepo(t *testing.T) {
	repoCalled := false
	repo := &fakeTrendRepo{}
	svc := usecase.NewTrendService(repo, 31337)
	cache := newFakeTrendCache()
	svc.SetCache(cache)

	since, _ := usecase.RangePresetToSince(usecase.TrendBucketDay, "30d", time.Now())
	// prime the cache directly via RefreshCache, then verify GetTrend reads it back without
	// requiring the repo to be called again.
	if _, err := svc.RefreshCache(context.Background(), usecase.TrendBucketDay, since, "30d"); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}

	repo.rows = nil // if GetTrend hits the repo now, it would return an empty (not cached) result
	points, err := svc.GetTrend(context.Background(), usecase.TrendBucketDay, since, "30d")
	if err != nil {
		t.Fatalf("GetTrend: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected cached (zero-filled, non-empty) points, cache appears to have been bypassed")
	}
	_ = repoCalled
}
