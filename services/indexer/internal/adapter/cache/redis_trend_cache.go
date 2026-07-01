// Package cache implements usecase.TrendCache over Redis.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// cacheTTL bounds how long a cached entry lives if the refresh job ever stops running —
// the read path never depends on TTL for correctness; a miss just recomputes from Postgres.
const cacheTTL = 30 * time.Minute

var _ usecase.TrendCache = (*RedisTrendCache)(nil)

// RedisTrendCache implements usecase.TrendCache over a Redis client.
type RedisTrendCache struct {
	client *redis.Client
}

// NewRedisTrendCache constructs a RedisTrendCache over an already-connected client.
func NewRedisTrendCache(client *redis.Client) *RedisTrendCache {
	return &RedisTrendCache{client: client}
}

// Get returns the cached points for key, or (nil, false, nil) on a miss.
func (c *RedisTrendCache) Get(ctx context.Context, key string) ([]usecase.TrendPoint, bool, error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis get %s: %w", key, err)
	}

	var points []usecase.TrendPoint
	if err := json.Unmarshal(raw, &points); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached trend %s: %w", key, err)
	}
	return points, true, nil
}

// Set writes points to key with a TTL backstop.
func (c *RedisTrendCache) Set(ctx context.Context, key string, points []usecase.TrendPoint) error {
	raw, err := json.Marshal(points)
	if err != nil {
		return fmt.Errorf("marshal trend %s: %w", key, err)
	}
	if err := c.client.Set(ctx, key, raw, cacheTTL).Err(); err != nil {
		return fmt.Errorf("redis set %s: %w", key, err)
	}
	return nil
}
