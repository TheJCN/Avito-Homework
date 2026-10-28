package create

import (
	"context"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/model"
)

type usecase interface {
	Execute(ctx context.Context, input model.CreateTrip) (trip.Trip, error)
}
