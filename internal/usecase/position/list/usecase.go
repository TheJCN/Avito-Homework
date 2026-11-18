package list

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
)

type Usecase struct {
	positions positionRepository
	trips     tripRepository
}

func New(positions positionRepository, trips tripRepository) *Usecase {
	return &Usecase{positions: positions, trips: trips}
}

func (u *Usecase) Execute(ctx context.Context, tripID uuid.UUID) ([]positiondomain.Position, error) {
	if _, err := u.trips.FindByID(ctx, tripID); err != nil {
		return nil, fmt.Errorf("find trip: %w", err)
	}
	result, err := u.positions.ListByTripID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	return result, nil
}
