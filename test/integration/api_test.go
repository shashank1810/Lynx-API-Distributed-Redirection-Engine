// Package integration provides end-to-end API tests for the gateway.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/raizel/gateway/internal/cache"
	"github.com/raizel/gateway/internal/config"
	"github.com/raizel/gateway/internal/encoding"
	"github.com/raizel/gateway/internal/middleware"
	"github.com/raizel/gateway/internal/model"
	"github.com/raizel/gateway/internal/router"
	"github.com/raizel/gateway/internal/service"
	"github.com/raizel/gateway/internal/telemetry"
)

// mockURLRepo implements repository.URLRepository for testing without PostgreSQL.
type mockURLRepo struct {
	urls map[string]*model.URL
}

func newMockURLRepo() *mockURLRepo {
	return &mockURLRepo{urls: make(map[string]*model.URL)}
}

func (m *mockURLRepo) Store(_ context.Context, u *model.URL) error {
	if _, exists := m.urls[u.ShortCode]; exists {
		return model.ErrDuplicateCode
	}
	u.ID = int64(len(m.urls) + 1)
	m.urls[u.ShortCode] = u
	return nil
}

func (m *mockURLRepo) FindByShortCode(_ context.Context, code string) (*model.URL, error) {
	u, ok := m.urls[code]
	if !ok {
		return nil, model.ErrURLNotFound
	}
	return u, nil
}

func (m *mockURLRepo) IncrementClicks(_ context.Context, code string) error {
	u, ok := m.urls[code]
	if !ok {
		return model.ErrURLNotFound
	}
	u.Clicks++
	return nil
}

func (m *mockURLRepo) Ping(_ context.Context) error {
	return nil
}

// setupTestRouter creates a test router with mock dependencies.
// NOTE: This test requires a running Redis instance on localhost:6379.
// In CI, use a Redis container or skip with: go test -short
func setupTestRouter(t *testing.T) http.Handler {
	t.Helper()

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
			DB:   1, // use DB 1 for tests
		},
		RateLimit: config.RateLimitConfig{
			Enabled:    false, // disable rate limiting in tests
			Rate:       1000,
			BucketSize: 2000,
			KeyPrefix:  "test:rl:",
		},
		ShortURL: config.ShortURLConfig{
			BaseURL:    "http://localhost:8080",
			CodeLength: 7,
			MaxRetries: 3,
		},
		Telemetry: config.TelemetryConfig{
			MetricsEnabled: false,
		},
	}

	logger := zap.NewNop()
	metrics := telemetry.NewMetrics()
	enc := encoding.NewEncoder(cfg.ShortURL.CodeLength)
	bloom := cache.NewBloomFilter(1000, 0.01)

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr,
		DB:   cfg.Redis.DB,
	})

	urlCache := cache.NewRedisCache(redisClient)
	repo := newMockURLRepo()
	cb := middleware.NewCircuitBreaker("test-db")

	svc := service.NewURLService(repo, urlCache, bloom, enc, cfg, metrics, logger)

	return router.New(&router.Deps{
		URLService:     svc,
		Repo:           repo,
		Cache:          urlCache,
		RedisClient:    redisClient,
		CircuitBreaker: cb,
		Config:         cfg,
		Metrics:        metrics,
		Logger:         logger,
	})
}

func TestHealthzEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", body["status"])
	}
}

func TestShortenAndResolve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := setupTestRouter(t)

	// Step 1: Shorten a URL.
	payload := `{"url": "https://example.com/very/long/path"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("shorten: expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var shortenResp model.ShortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&shortenResp); err != nil {
		t.Fatalf("failed to decode shorten response: %v", err)
	}

	if shortenResp.ShortCode == "" {
		t.Fatal("short_code should not be empty")
	}

	// Step 2: Resolve the short code (should 301 redirect).
	req = httptest.NewRequest(http.MethodGet, "/"+shortenResp.ShortCode, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("resolve: expected 301, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	location := rec.Header().Get("Location")
	if location != "https://example.com/very/long/path" {
		t.Errorf("expected redirect to original URL, got %s", location)
	}
}

func TestShortenInvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := setupTestRouter(t)

	payload := `{"url": "not-a-valid-url"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestResolveNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
