package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/model"
)

type repository interface {
	FindStale(context.Context, time.Duration) ([]model.StaleTrip, error)
}

type pushGateway interface {
	RequestPosition(context.Context, uuid.UUID, uuid.UUID) error
}

type metrics interface {
	StaleCheck(context.Context, string)
	PositionQueueSize(context.Context, int)
	PositionWorkersActive(context.Context, int64)
	PositionRequest(context.Context, string, time.Duration)
	PositionRequestDropped(context.Context)
}

type Ticker interface {
	C() <-chan time.Time
	Stop()
}
