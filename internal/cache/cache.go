// Package cache defines the caching interface and its Redis implementation.
package cache

import (
	"context"
	"time"

	"github.com/raizel/gateway/internal/model"
)

// URLCache defines the contract for URL caching operations (cache-aside pattern).
type URLCache interface {
	// Get retrieves a cached URL by short code. Returns nil if not found.
	Get(ctx context.Context, code string) (*model.URL, error)

	// Set caches a URL with the given TTL.
	Set(ctx context.Context, url *model.URL, ttl time.Duration) error

	// Delete removes a cached URL entry.
	Delete(ctx context.Context, code string) error

	// SetNull caches a null marker to defend against cache penetration.
	// Prevents repeated DB lookups for nonexistent keys.
	SetNull(ctx context.Context, code string, ttl time.Duration) error

	// IsNullCached checks whether a code has been marked as nonexistent.
	IsNullCached(ctx context.Context, code string) (bool, error)

	// Ping verifies cache connectivity.
	Ping(ctx context.Context) error
}
