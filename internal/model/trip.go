package model

import (
	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type CreateTrip struct {
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
	IdempotencyKey *uuid.UUID
	RequestHash    []byte
}

type CreateTripResult struct {
	Trip     trip.Trip
	Replayed bool
}
