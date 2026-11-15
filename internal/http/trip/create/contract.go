package create

//go:generate go run go.uber.org/mock/mockgen -source contract.go -destination mocks_test.go -package create

import (
	"context"

	"github.com/TheJCN/Avito-Course/internal/model"
)

type usecase interface {
	Execute(ctx context.Context, input model.CreateTrip) (model.CreateTripResult, error)
}
