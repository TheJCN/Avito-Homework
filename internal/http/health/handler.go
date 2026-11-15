package health

import (
	"context"
	"time"

	"github.com/TheJCN/Avito-Course/api"
)

type pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	database pinger
	timeout  time.Duration
}

func New(database pinger, timeout time.Duration) *Handler {
	return &Handler{database: database, timeout: timeout}
}

func (h *Handler) Health() api.HealthResponse {
	return api.HealthResponse{Status: api.Ok}
}

func (h *Handler) Ready(ctx context.Context) (api.HealthResponse, bool) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	if err := h.database.Ping(ctx); err != nil {
		return api.HealthResponse{Status: api.Unavailable}, false
	}

	return api.HealthResponse{Status: api.Ok}, true
}
