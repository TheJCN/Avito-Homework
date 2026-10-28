package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type usecase interface {
	Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error)
}
