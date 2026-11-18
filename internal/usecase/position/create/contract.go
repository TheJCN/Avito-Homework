package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type positionRepository interface {
	Create(context.Context, positiondomain.Position) (positiondomain.Position, error)
	UpdateLastPositionAt(context.Context, uuid.UUID, time.Time) error
}

type tripRepository interface {
	FindByIDForUpdate(context.Context, uuid.UUID) (trip.Trip, error)
}

type transactionManager interface {
	Do(context.Context, func(context.Context) error) error
}

type pendingCanceler interface {
	Cancel(uuid.UUID) bool
}

type metrics interface {
	PositionSaved(context.Context)
}
