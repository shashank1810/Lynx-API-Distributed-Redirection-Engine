package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/raizel/gateway/internal/service"
)

// StatsHandler handles GET /api/v1/stats/:code requests.
type StatsHandler struct {
	svc *service.URLService
}

// NewStatsHandler creates a new stats handler.
func NewStatsHandler(svc *service.URLService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

// ServeHTTP returns analytics data for a given short code.
func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("short code is required"))
		return
	}

	stats, err := h.svc.GetStats(r.Context(), code)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
