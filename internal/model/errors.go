package model

import "errors"

// Domain-level sentinel errors used across all layers.
var (
	// ErrURLNotFound indicates the requested short code does not exist.
	ErrURLNotFound = errors.New("url not found")

	// ErrURLExpired indicates the short URL has passed its expiration.
	ErrURLExpired = errors.New("url has expired")

	// ErrURLInactive indicates the short URL has been deactivated.
	ErrURLInactive = errors.New("url is inactive")

	// ErrDuplicateCode indicates the short code already exists in storage.
	ErrDuplicateCode = errors.New("short code already exists")

	// ErrInvalidURL indicates the provided URL failed validation.
	ErrInvalidURL = errors.New("invalid url provided")

	// ErrRateLimited indicates the client has exceeded the allowed request rate.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrServiceUnavailable indicates a downstream dependency is unreachable.
	ErrServiceUnavailable = errors.New("service temporarily unavailable")

	// ErrInvalidShortCode indicates the short code format is invalid.
	ErrInvalidShortCode = errors.New("invalid short code format")
)
