package list

import (
	"context"

	"github.com/google/uuid"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
)

type usecase interface {
	Execute(context.Context, uuid.UUID) ([]positiondomain.Position, error)
}
