package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type Handler struct {
	usecase usecase
}

func New(usecase usecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	return h.usecase.Execute(ctx, id)
}
