package middleware

import (
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/raizel/gateway/internal/telemetry"
)

// Logger returns middleware that logs every HTTP request with structured fields
// and records Prometheus metrics for request duration and status.
func Logger(logger *zap.Logger, metrics *telemetry.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture status code and bytes written.
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			statusStr := strconv.Itoa(wrapped.statusCode)
			path := r.URL.Path

			// Prometheus metrics.
			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration.Seconds())
			metrics.HTTPResponseSize.WithLabelValues(r.Method, path).Observe(float64(wrapped.bytesWritten))

			// Structured log.
			logger.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", path),
				zap.Int("status", wrapped.statusCode),
				zap.Duration("duration", duration),
				zap.Int("bytes", wrapped.bytesWritten),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("request_id", GetRequestID(r.Context())),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}
