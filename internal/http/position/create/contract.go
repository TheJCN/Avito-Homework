package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
)

type usecase interface {
	Execute(context.Context, uuid.UUID, float64, float64, time.Time) (positiondomain.Position, error)
}
