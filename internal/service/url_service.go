// Package service provides the core business logic for URL management.
package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/raizel/gateway/internal/cache"
	"github.com/raizel/gateway/internal/config"
	"github.com/raizel/gateway/internal/encoding"
	"github.com/raizel/gateway/internal/model"
	"github.com/raizel/gateway/internal/repository"
	"github.com/raizel/gateway/internal/telemetry"
	"github.com/raizel/gateway/pkg/validator"
)

// URLService orchestrates URL shortening and resolution with cache-aside pattern.
type URLService struct {
	repo    repository.URLRepository
	cache   cache.URLCache
	bloom   *cache.BloomFilter
	encoder *encoding.Encoder
	cfg     *config.Config
	metrics *telemetry.Metrics
	logger  *zap.Logger
}

// NewURLService creates a new URLService instance with all dependencies.
func NewURLService(
	repo repository.URLRepository,
	urlCache cache.URLCache,
	bloom *cache.BloomFilter,
	encoder *encoding.Encoder,
	cfg *config.Config,
	metrics *telemetry.Metrics,
	logger *zap.Logger,
) *URLService {
	return &URLService{
		repo:    repo,
		cache:   urlCache,
		bloom:   bloom,
		encoder: encoder,
		cfg:     cfg,
		metrics: metrics,
		logger:  logger,
	}
}

// Shorten creates a new short URL for the given request.
func (s *URLService) Shorten(ctx context.Context, req *model.ShortenRequest) (*model.ShortenResponse, error) {
	// Validate the input URL.
	normalizedURL := validator.NormalizeURL(req.URL)
	if !validator.IsValidURL(normalizedURL) {
		return nil, model.ErrInvalidURL
	}

	var shortCode string
	var err error

	if req.CustomCode != "" {
		// Use custom code if provided.
		if !s.encoder.IsValidCode(req.CustomCode) {
			return nil, model.ErrInvalidShortCode
		}
		shortCode = req.CustomCode
	} else {
		// Generate a random code with collision retry.
		for attempt := 0; attempt < s.cfg.ShortURL.MaxRetries; attempt++ {
			shortCode, err = s.encoder.GenerateRandom()
			if err != nil {
				return nil, fmt.Errorf("generating short code: %w", err)
			}
			// Quick bloom check — if bloom says "not exists", it's definitely available.
			if !s.bloom.MayExist(shortCode) {
				break
			}
			// Bloom says "maybe exists" — try another code.
			shortCode = ""
		}
		if shortCode == "" {
			// All retries collided — generate one more as a last resort.
			shortCode, err = s.encoder.GenerateRandom()
			if err != nil {
				return nil, fmt.Errorf("generating short code (final): %w", err)
			}
		}
	}

	// Build the URL entity.
	urlEntity := &model.URL{
		ShortCode:   shortCode,
		OriginalURL: normalizedURL,
		IsActive:    true,
	}

	if req.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		urlEntity.ExpiresAt = &expiry
	}

	// Persist to database.
	if err := s.repo.Store(ctx, urlEntity); err != nil {
		return nil, err
	}

	// Register in bloom filter.
	s.bloom.Add(shortCode)

	// Cache the new entry.
	if err := s.cache.Set(ctx, urlEntity, s.cfg.Redis.CacheTTL); err != nil {
		s.logger.Warn("failed to cache newly shortened URL",
			zap.String("code", shortCode),
			zap.Error(err),
		)
	}

	(*s.metrics.URLsShortenedTotal).Inc()

	return &model.ShortenResponse{
		ShortCode:   shortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.cfg.ShortURL.BaseURL, shortCode),
		OriginalURL: normalizedURL,
	}, nil
}

// Resolve looks up the original URL for a short code using the cache-aside pattern:
// 1. Check bloom filter (reject definite non-existence).
// 2. Check null-marker cache (reject recently-confirmed non-existence).
// 3. Check URL cache (return on hit).
// 4. Fall through to database (populate cache on success, set null on miss).
func (s *URLService) Resolve(ctx context.Context, code string) (string, error) {
	// Step 1: Bloom filter check.
	if !s.bloom.MayExist(code) {
		s.metrics.BloomFilterRejections.Inc()
		return "", model.ErrURLNotFound
	}

	// Step 2: Null-marker check (cache penetration defense).
	isNull, err := s.cache.IsNullCached(ctx, code)
	if err != nil {
		s.logger.Warn("null cache check failed", zap.String("code", code), zap.Error(err))
	}
	if isNull {
		return "", model.ErrURLNotFound
	}

	// Step 3: Cache lookup.
	cached, err := s.cache.Get(ctx, code)
	if err != nil {
		s.logger.Warn("cache GET failed", zap.String("code", code), zap.Error(err))
	}
	if cached != nil {
		s.metrics.URLCacheHits.Inc()
		// Fire-and-forget click increment.
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := s.repo.IncrementClicks(bgCtx, code); err != nil {
				s.logger.Warn("failed to increment clicks", zap.String("code", code), zap.Error(err))
			}
		}()
		(*s.metrics.URLsResolvedTotal).Inc()
		return cached.OriginalURL, nil
	}

	s.metrics.URLCacheMisses.Inc()

	// Step 4: Database lookup.
	urlEntity, err := s.repo.FindByShortCode(ctx, code)
	if err != nil {
		if err == model.ErrURLNotFound {
			// Set null marker to prevent repeated DB lookups.
			_ = s.cache.SetNull(ctx, code, 2*time.Minute)
		}
		return "", err
	}

	// Populate cache for future requests.
	if cacheErr := s.cache.Set(ctx, urlEntity, s.cfg.Redis.CacheTTL); cacheErr != nil {
		s.logger.Warn("failed to cache resolved URL", zap.String("code", code), zap.Error(cacheErr))
	}

	// Increment clicks asynchronously.
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := s.repo.IncrementClicks(bgCtx, code); err != nil {
			s.logger.Warn("failed to increment clicks", zap.String("code", code), zap.Error(err))
		}
	}()

	(*s.metrics.URLsResolvedTotal).Inc()
	return urlEntity.OriginalURL, nil
}

// GetStats returns analytics for a short URL.
func (s *URLService) GetStats(ctx context.Context, code string) (*model.StatsResponse, error) {
	urlEntity, err := s.repo.FindByShortCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return &model.StatsResponse{
		ShortCode:   urlEntity.ShortCode,
		OriginalURL: urlEntity.OriginalURL,
		Clicks:      urlEntity.Clicks,
		CreatedAt:   urlEntity.CreatedAt,
		IsActive:    urlEntity.IsActive,
	}, nil
}
