package finish

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
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

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	var result trip.Trip

	err := u.transaction.Do(ctx, func(ctx context.Context) error {
		value, err := u.repository.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("find trip: %w", err)
		}

		fromStatus := value.Status()
		if err := value.Finish(time.Now().UTC()); err != nil {
			return err
		}

		if err := u.repository.Update(ctx, value); err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}

		if err := u.repository.CreateStatusHistory(ctx, trip.StatusHistory{
			TripID:     value.ID(),
			FromStatus: &fromStatus,
			ToStatus:   trip.StatusCompleted,
		}); err != nil {
			return fmt.Errorf("create trip history: %w", err)
		}

		result, err = u.repository.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("find completed trip: %w", err)
		}
		return nil
	})
	if err != nil {
		return trip.Trip{}, err
	}
	if u.metrics != nil {
		u.metrics.TripCompleted(ctx)
	}

	return result, nil
}
