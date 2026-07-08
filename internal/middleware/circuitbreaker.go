package middleware

import (
	"net/http"

	"github.com/sony/gobreaker/v2"

	"github.com/raizel/gateway/internal/telemetry"
)

// CircuitBreaker wraps an HTTP handler with a circuit breaker to prevent
// cascade failures when downstream services (PostgreSQL) are degraded.
func CircuitBreaker(cb *gobreaker.CircuitBreaker[*http.Response], metrics *telemetry.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Update metrics with current state.
			state := cb.State()
			metrics.CircuitBreakerState.WithLabelValues("database").Set(float64(state))

			if state == gobreaker.StateOpen {
				http.Error(w, `{"error":"service temporarily unavailable"}`, http.StatusServiceUnavailable)
				return
			}

			// Execute the request through the circuit breaker.
			_, cbErr := cb.Execute(func() (*http.Response, error) {
				// Create a response recorder to capture the result.
				rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
				next.ServeHTTP(rec, r)

				// Treat 5xx errors as circuit breaker failures.
				if rec.statusCode >= 500 {
					return nil, &serverError{code: rec.statusCode}
				}
				return nil, nil
			})

			if cbErr != nil {
				// If the circuit breaker itself rejected (not a passthrough error):
				if _, ok := cbErr.(*serverError); !ok {
					http.Error(w, `{"error":"service temporarily unavailable"}`, http.StatusServiceUnavailable)
				}
			}
		})
	}
}

// NewCircuitBreaker creates a configured gobreaker instance.
func NewCircuitBreaker(name string) *gobreaker.CircuitBreaker[*http.Response] {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,                  // half-open: allow 3 probe requests
		Interval:    0,                  // don't reset counts in closed state
		Timeout:     10,                 // 10 seconds in open state before half-open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.6
		},
	}
	return gobreaker.NewCircuitBreaker[*http.Response](settings)
}

// statusRecorder captures the HTTP status code written by the handler.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// serverError signals a server-side failure to the circuit breaker.
type serverError struct {
	code int
}

func (e *serverError) Error() string {
	return http.StatusText(e.code)
}
