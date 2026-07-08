// Package handler provides HTTP request handlers for the gateway API.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/raizel/gateway/internal/model"
	"github.com/raizel/gateway/internal/service"
)

// ShortenHandler handles POST /api/v1/shorten requests.
type ShortenHandler struct {
	svc *service.URLService
}

// NewShortenHandler creates a new shorten handler.
func NewShortenHandler(svc *service.URLService) *ShortenHandler {
	return &ShortenHandler{svc: svc}
}

// ServeHTTP processes URL shortening requests.
func (h *ShortenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse("method not allowed"))
		return
	}

	var req model.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}
	defer r.Body.Close()

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("url is required"))
		return
	}

	resp, err := h.svc.Shorten(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}
