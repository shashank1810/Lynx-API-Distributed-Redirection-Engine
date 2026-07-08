package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/raizel/gateway/internal/model"
)

// writeJSON writes a JSON response with the given status code and body.
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// errorResponse creates a standard error payload.
func errorResponse(msg string) map[string]string {
	return map[string]string{"error": msg}
}

// handleServiceError maps domain errors to HTTP status codes.
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrURLNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse("url not found"))
	case errors.Is(err, model.ErrURLExpired):
		writeJSON(w, http.StatusGone, errorResponse("url has expired"))
	case errors.Is(err, model.ErrURLInactive):
		writeJSON(w, http.StatusGone, errorResponse("url is inactive"))
	case errors.Is(err, model.ErrDuplicateCode):
		writeJSON(w, http.StatusConflict, errorResponse("short code already exists"))
	case errors.Is(err, model.ErrInvalidURL):
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid url"))
	case errors.Is(err, model.ErrInvalidShortCode):
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid short code format"))
	case errors.Is(err, model.ErrRateLimited):
		writeJSON(w, http.StatusTooManyRequests, errorResponse("rate limit exceeded"))
	case errors.Is(err, model.ErrServiceUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, errorResponse("service temporarily unavailable"))
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
	}
}
