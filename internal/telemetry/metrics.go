// Package telemetry provides Prometheus metrics collectors for the gateway.
package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metric collectors for the application.
type Metrics struct {
	// HTTP metrics.
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// URL service metrics.
	URLsShortenedTotal *prometheus.Counter
	URLsResolvedTotal  *prometheus.Counter
	URLCacheHits       prometheus.Counter
	URLCacheMisses     prometheus.Counter

	// Rate limiter metrics.
	RateLimitAllowed prometheus.Counter
	RateLimitDenied  prometheus.Counter

	// Circuit breaker metrics.
	CircuitBreakerState *prometheus.GaugeVec

	// Bloom filter metrics.
	BloomFilterRejections prometheus.Counter
}

// NewMetrics registers and returns all Prometheus metrics.
func NewMetrics() *Metrics {
	httpRequestsTotal := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gateway",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests by method, path, and status.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "gateway",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency distribution in seconds.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path"},
	)

	httpResponseSize := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "gateway",
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response body size distribution in bytes.",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 6),
		},
		[]string{"method", "path"},
	)

	urlsShortenedTotal := promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gateway",
			Subsystem: "urls",
			Name:      "shortened_total",
			Help:      "Total number of URLs shortened.",
		},
	)

	urlsResolvedTotal := promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gateway",
			Subsystem: "urls",
			Name:      "resolved_total",
			Help:      "Total number of URLs resolved (redirected).",
		},
	)

	urlCacheHits := promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gateway",
			Subsystem: "cache",
			Name:      "hits_total",
			Help:      "Total number of cache hits.",
		},
	)

	urlCacheMisses := promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gateway",
			Subsystem: "cache",
			Name:      "misses_total",
			Help:      "Total number of cache misses.",
		},
	)

	rateLimitAllowed := promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gateway",
			Subsystem: "ratelimit",
			Name:      "allowed_total",
			Help:      "Total requests allowed by rate limiter.",
		},
	)

	rateLimitDenied := promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gateway",
			Subsystem: "ratelimit",
			Name:      "denied_total",
			Help:      "Total requests denied by rate limiter.",
		},
	)

	circuitBreakerState := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gateway",
			Subsystem: "circuit_breaker",
			Name:      "state",
			Help:      "Current circuit breaker state (0=closed, 1=half-open, 2=open).",
		},
		[]string{"name"},
	)

	bloomFilterRejections := promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gateway",
			Subsystem: "bloom",
			Name:      "rejections_total",
			Help:      "Total lookups rejected by the bloom filter (definite non-existence).",
		},
	)

	return &Metrics{
		HTTPRequestsTotal:     httpRequestsTotal,
		HTTPRequestDuration:   httpRequestDuration,
		HTTPResponseSize:      httpResponseSize,
		URLsShortenedTotal:    &urlsShortenedTotal,
		URLsResolvedTotal:     &urlsResolvedTotal,
		URLCacheHits:          urlCacheHits,
		URLCacheMisses:        urlCacheMisses,
		RateLimitAllowed:      rateLimitAllowed,
		RateLimitDenied:       rateLimitDenied,
		CircuitBreakerState:   circuitBreakerState,
		BloomFilterRejections: bloomFilterRejections,
	}
}
