package create

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type Usecase struct {
	positions   positionRepository
	trips       tripRepository
	transaction transactionManager
	now         func() time.Time
	canceler    pendingCanceler
	metrics     metrics
}

func New(positions positionRepository, trips tripRepository, transaction transactionManager) *Usecase {
	return &Usecase{positions: positions, trips: trips, transaction: transaction, now: time.Now}
}

func NewWithObservers(positions positionRepository, trips tripRepository, transaction transactionManager, canceler pendingCanceler, metric metrics) *Usecase {
	return &Usecase{positions: positions, trips: trips, transaction: transaction, now: time.Now, canceler: canceler, metrics: metric}
}

func (u *Usecase) Execute(ctx context.Context, tripID uuid.UUID, latitude, longitude float64, recordedAt time.Time) (positiondomain.Position, error) {
	if err := positiondomain.Validate(latitude, longitude, recordedAt, u.now().UTC()); err != nil {
		return positiondomain.Position{}, err
	}

	var result positiondomain.Position
	err := u.transaction.Do(ctx, func(ctx context.Context) error {
		value, err := u.trips.FindByIDForUpdate(ctx, tripID)
		if err != nil {
			return fmt.Errorf("find trip: %w", err)
		}
		if value.Status() == trip.StatusCompleted {
			return trip.ErrCompleted
		}

		result, err = u.positions.Create(ctx, positiondomain.Position{
			TripID: tripID, Latitude: latitude, Longitude: longitude, RecordedAt: recordedAt,
		})
		if err != nil {
			return err
		}
		if err := u.positions.UpdateLastPositionAt(ctx, tripID, recordedAt); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return positiondomain.Position{}, err
	}
	if u.canceler != nil {
		u.canceler.Cancel(tripID)
	}
	if u.metrics != nil {
		u.metrics.PositionSaved(ctx)
	}
	return result, nil
}
