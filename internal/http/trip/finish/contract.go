package finish

//go:generate go run go.uber.org/mock/mockgen -source contract.go -destination mocks_test.go -package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type usecase interface {
	Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error)
}
