package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type healthResponse struct {
	Status string `json:"status"`
}

type DatabasePinger interface {
	Ping(ctx context.Context) error
}

type ReadyHandler struct {
	db DatabasePinger
}

func Live(w http.ResponseWriter, _ *http.Request) {
	writeHealthResponse(w, http.StatusOK, "ok")
}

func NewReadyHandler(db DatabasePinger) ReadyHandler {
	return ReadyHandler{db: db}
}

func (h ReadyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		writeHealthResponse(w, http.StatusServiceUnavailable, "unavailable")
		return
	}

	writeHealthResponse(w, http.StatusOK, "ok")
}

func writeHealthResponse(w http.ResponseWriter, statusCode int, status string) {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: status})
}
