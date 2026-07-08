package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/raizel/gateway/internal/cache"
	"github.com/raizel/gateway/internal/repository"
)

// HealthHandler handles GET /healthz and GET /readyz requests.
type HealthHandler struct {
	repo  repository.URLRepository
	cache cache.URLCache
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(repo repository.URLRepository, cache cache.URLCache) *HealthHandler {
	return &HealthHandler{repo: repo, cache: cache}
}

// Liveness handles GET /healthz — indicates the process is running.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
	})
}

// Readiness handles GET /readyz — verifies all dependencies are reachable.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	services := make(map[string]string)
	healthy := true

	// Check PostgreSQL.
	if err := h.repo.Ping(ctx); err != nil {
		services["postgres"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		services["postgres"] = "healthy"
	}

	// Check Redis.
	if err := h.cache.Ping(ctx); err != nil {
		services["redis"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		services["redis"] = "healthy"
	}

	status := "ok"
	statusCode := http.StatusOK
	if !healthy {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().UTC(),
		"services":  services,
	})
}
