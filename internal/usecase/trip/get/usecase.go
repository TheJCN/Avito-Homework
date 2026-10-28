package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type Usecase struct {
	repository tripRepository
}

func New(repository tripRepository) *Usecase {
	return &Usecase{repository: repository}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	value, err := u.repository.FindByID(ctx, id)
	if err != nil {
		return trip.Trip{}, fmt.Errorf("find trip: %w", err)
	}

	return value, nil
}
