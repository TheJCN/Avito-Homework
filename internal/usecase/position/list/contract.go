package list

import (
	"context"

	"github.com/google/uuid"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type positionRepository interface {
	ListByTripID(context.Context, uuid.UUID) ([]positiondomain.Position, error)
}

type tripRepository interface {
	FindByID(context.Context, uuid.UUID) (trip.Trip, error)
}
