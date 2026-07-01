package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/cache"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

func newTestCache(t *testing.T) *cache.RedisTrendCache {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return cache.NewRedisTrendCache(client)
}

func TestRedisTrendCache_MissThenSetThenHit(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	_, ok, err := c.Get(ctx, "trend:v1:1:1:30d")
	if err != nil {
		t.Fatalf("Get (miss): %v", err)
	}
	if ok {
		t.Fatal("expected a miss on an empty cache")
	}

	want := []usecase.TrendPoint{{BucketStart: time.Now().UTC().Truncate(time.Second), Count: 3}}
	if err := c.Set(ctx, "trend:v1:1:1:30d", want); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok, err := c.Get(ctx, "trend:v1:1:1:30d")
	if err != nil {
		t.Fatalf("Get (hit): %v", err)
	}
	if !ok {
		t.Fatal("expected a hit after Set")
	}
	if len(got) != 1 || got[0].Count != 3 {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
