// Package repository defines the data-access interface and its PostgreSQL implementation.
package repository

import (
	"context"

	"github.com/raizel/gateway/internal/model"
)

// URLRepository defines the contract for URL persistence operations.
type URLRepository interface {
	// Store persists a new shortened URL. Returns ErrDuplicateCode on conflict.
	Store(ctx context.Context, url *model.URL) error

	// FindByShortCode retrieves a URL by its short code.
	FindByShortCode(ctx context.Context, code string) (*model.URL, error)

	// IncrementClicks atomically increments the click counter.
	IncrementClicks(ctx context.Context, code string) error

	// Ping verifies database connectivity.
	Ping(ctx context.Context) error
}
