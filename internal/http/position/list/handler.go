package list

import (
	"context"

	"github.com/google/uuid"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
)

type Handler struct{ usecase usecase }

func New(usecase usecase) *Handler { return &Handler{usecase: usecase} }

func (h *Handler) Handle(ctx context.Context, tripID uuid.UUID) ([]positiondomain.Position, error) {
	return h.usecase.Execute(ctx, tripID)
}
