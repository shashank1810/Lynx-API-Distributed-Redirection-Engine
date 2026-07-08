package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/raizel/gateway/internal/model"
)

const (
	// cachePrefix namespaces all URL cache keys.
	cachePrefix = "url:"

	// nullPrefix namespaces null-marker keys for cache penetration defense.
	nullPrefix = "null:"

	// nullMarker is the sentinel value stored for nonexistent keys.
	nullMarker = "∅"

	// nullTTL is the default TTL for null markers (short to allow self-healing).
	nullTTL = 2 * time.Minute
)

// RedisCache implements URLCache backed by Redis.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache instance.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Get retrieves a cached URL by its short code.
// Returns (nil, nil) on cache miss (key does not exist).
func (c *RedisCache) Get(ctx context.Context, code string) (*model.URL, error) {
	key := cachePrefix + code

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // cache miss
		}
		return nil, fmt.Errorf("redis GET %s: %w", key, err)
	}

	var url model.URL
	if err := json.Unmarshal(data, &url); err != nil {
		// Corrupted entry — delete it and treat as miss.
		_ = c.client.Del(ctx, key)
		return nil, nil
	}

	return &url, nil
}

// Set caches a URL with the specified TTL.
func (c *RedisCache) Set(ctx context.Context, url *model.URL, ttl time.Duration) error {
	key := cachePrefix + url.ShortCode

	data, err := json.Marshal(url)
	if err != nil {
		return fmt.Errorf("marshalling url for cache: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis SET %s: %w", key, err)
	}

	// Remove any null marker for this code since it now exists.
	_ = c.client.Del(ctx, nullPrefix+url.ShortCode)

	return nil
}

// Delete removes a cached URL entry.
func (c *RedisCache) Delete(ctx context.Context, code string) error {
	key := cachePrefix + code
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis DEL %s: %w", key, err)
	}
	return nil
}

// SetNull caches a null marker to prevent repeated database lookups for nonexistent keys.
func (c *RedisCache) SetNull(ctx context.Context, code string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = nullTTL
	}
	key := nullPrefix + code
	if err := c.client.Set(ctx, key, nullMarker, ttl).Err(); err != nil {
		return fmt.Errorf("redis SET null marker %s: %w", key, err)
	}
	return nil
}

// IsNullCached checks whether a code has been flagged as nonexistent.
func (c *RedisCache) IsNullCached(ctx context.Context, code string) (bool, error) {
	key := nullPrefix + code
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("redis GET null marker %s: %w", key, err)
	}
	return val == nullMarker, nil
}

// Ping verifies Redis connectivity.
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
