// Package router registers all HTTP routes for the gateway.
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"

	"github.com/raizel/gateway/internal/cache"
	"github.com/raizel/gateway/internal/config"
	"github.com/raizel/gateway/internal/handler"
	"github.com/raizel/gateway/internal/middleware"
	"github.com/raizel/gateway/internal/repository"
	"github.com/raizel/gateway/internal/service"
	"github.com/raizel/gateway/internal/telemetry"
)

// Deps holds all dependencies required to construct the router.
type Deps struct {
	URLService     *service.URLService
	Repo           repository.URLRepository
	Cache          cache.URLCache
	RedisClient    *redis.Client
	CircuitBreaker *gobreaker.CircuitBreaker[*http.Response]
	Config         *config.Config
	Metrics        *telemetry.Metrics
	Logger         *zap.Logger
}

// New creates and returns a fully configured chi router.
func New(deps *Deps) *chi.Mux {
	r := chi.NewRouter()

	// --- Global middleware stack (order matters) ---
	r.Use(chimiddleware.RealIP)               // Extract real IP from proxy headers
	r.Use(middleware.RequestID)                // Inject X-Request-ID
	r.Use(middleware.CORS())                   // CORS headers
	r.Use(chimiddleware.Recoverer)             // Panic recovery
	r.Use(middleware.Logger(deps.Logger, deps.Metrics)) // Structured logging + metrics

	// Rate limiter (applied globally).
	r.Use(middleware.RateLimiter(deps.RedisClient, &deps.Config.RateLimit, deps.Metrics))

	// --- Health probes (no circuit breaker / no auth) ---
	healthHandler := handler.NewHealthHandler(deps.Repo, deps.Cache)
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// --- Prometheus metrics endpoint ---
	if deps.Config.Telemetry.MetricsEnabled {
		r.Handle(deps.Config.Telemetry.MetricsPath, promhttp.Handler())
	}

	// --- API v1 routes (behind circuit breaker) ---
	r.Route("/api/v1", func(apiRouter chi.Router) {
		apiRouter.Use(middleware.CircuitBreaker(deps.CircuitBreaker, deps.Metrics))

		shortenHandler := handler.NewShortenHandler(deps.URLService)
		statsHandler := handler.NewStatsHandler(deps.URLService)

		apiRouter.Post("/shorten", shortenHandler.ServeHTTP)
		apiRouter.Get("/stats/{code}", statsHandler.ServeHTTP)
	})

	// --- Redirect route (hot path, behind circuit breaker) ---
	resolveHandler := handler.NewResolveHandler(deps.URLService)
	r.Group(func(redirectRouter chi.Router) {
		redirectRouter.Use(middleware.CircuitBreaker(deps.CircuitBreaker, deps.Metrics))
		redirectRouter.Get("/{code}", resolveHandler.ServeHTTP)
	})

	return r
}
