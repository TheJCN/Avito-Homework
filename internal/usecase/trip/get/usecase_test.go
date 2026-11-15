package get

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

func TestExecuteReturnsTrip(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	id := uuid.New()
	value := trip.Restore(id, uuid.New(), uuid.New(), 1, 2, 3, 4, 500, trip.StatusActive, time.Now().UTC(), nil, time.Now().UTC(), time.Now().UTC())
	repository.EXPECT().FindByID(gomock.Any(), id).Return(value, nil)

	result, err := New(repository).Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if result.ID() != id {
		t.Fatal("usecase must return repository trip")
	}
}

func TestExecuteReturnsRepositoryError(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	want := errors.New("database error")
	id := uuid.New()
	repository.EXPECT().FindByID(gomock.Any(), id).Return(trip.Trip{}, want)

	_, err := New(repository).Execute(context.Background(), id)

	if !errors.Is(err, want) {
		t.Fatalf("want repository error, got %v", err)
	}
}
