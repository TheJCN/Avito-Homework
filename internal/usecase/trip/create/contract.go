package create

//go:generate go run go.uber.org/mock/mockgen -source contract.go -destination mocks_test.go -package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type tripRepository interface {
	Create(ctx context.Context, value trip.Trip) error
	FindByID(ctx context.Context, id uuid.UUID) (trip.Trip, error)
	CreateStatusHistory(ctx context.Context, history trip.StatusHistory) error
	ReserveIdempotencyKey(ctx context.Context, key uuid.UUID, requestHash []byte, expiresAt time.Time) (*uuid.UUID, error)
	SetIdempotencyTrip(ctx context.Context, key, tripID uuid.UUID) error
}

type transactionManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type metrics interface {
	TripCreated(ctx context.Context)
}
