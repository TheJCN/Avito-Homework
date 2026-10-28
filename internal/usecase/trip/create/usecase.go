package create

import (
	"context"
	"fmt"
	"time"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/model"
)

type Usecase struct {
	repository  tripRepository
	transaction transactionManager
	metrics     metrics
}

func New(repository tripRepository, transaction transactionManager, values ...metrics) *Usecase {
	var metric metrics
	if len(values) > 0 {
		metric = values[0]
	}
	return &Usecase{repository: repository, transaction: transaction, metrics: metric}
}

func (u *Usecase) Execute(ctx context.Context, input model.CreateTrip) (trip.Trip, error) {
	value := trip.New(
		input.UserID,
		input.DriverID,
		input.StartLatitude,
		input.StartLongitude,
		input.EndLatitude,
		input.EndLongitude,
		input.Price,
		time.Now().UTC(),
	)

	err := u.transaction.Do(ctx, func(ctx context.Context) error {
		if err := u.repository.Create(ctx, value); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}

		if err := u.repository.CreateStatusHistory(ctx, trip.StatusHistory{
			TripID:   value.ID(),
			ToStatus: trip.StatusActive,
		}); err != nil {
			return fmt.Errorf("create trip history: %w", err)
		}

		result, err := u.repository.FindByID(ctx, value.ID())
		if err != nil {
			return fmt.Errorf("find created trip: %w", err)
		}
		value = result
		return nil
	})
	if err != nil {
		return trip.Trip{}, err
	}
	if u.metrics != nil {
		u.metrics.TripCreated(ctx)
	}

	return value, nil
}
