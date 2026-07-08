package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/raizel/gateway/internal/service"
)

// ResolveHandler handles GET /:code requests — the hot redirect path.
type ResolveHandler struct {
	svc *service.URLService
}

// NewResolveHandler creates a new resolve handler.
func NewResolveHandler(svc *service.URLService) *ResolveHandler {
	return &ResolveHandler{svc: svc}
}

// ServeHTTP looks up the short code and issues a 301 permanent redirect.
func (h *ResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("short code is required"))
		return
	}

	originalURL, err := h.svc.Resolve(r.Context(), code)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}
