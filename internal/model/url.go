// Package model defines the core domain types for the URL management platform.
package model

import "time"

// URL represents a shortened URL entity in the system.
type URL struct {
	ID          int64     `json:"id"          db:"id"`
	ShortCode   string    `json:"short_code"  db:"short_code"`
	OriginalURL string    `json:"original_url" db:"original_url"`
	Clicks      int64     `json:"clicks"      db:"clicks"`
	CreatedAt   time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"  db:"updated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsActive    bool      `json:"is_active"   db:"is_active"`
	CreatedBy   string    `json:"created_by,omitempty" db:"created_by"`
}

// ShortenRequest is the inbound payload for creating a short URL.
type ShortenRequest struct {
	URL       string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
	ExpiresIn  int    `json:"expires_in,omitempty"` // seconds
}

// ShortenResponse is returned after a URL is successfully shortened.
type ShortenResponse struct {
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// StatsResponse returns analytics for a short URL.
type StatsResponse struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	Clicks      int64     `json:"clicks"`
	CreatedAt   time.Time `json:"created_at"`
	IsActive    bool      `json:"is_active"`
}

// HealthResponse represents the health check payload.
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
}
