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

type testRepository struct {
	tripToReturn trip.Trip
	err          error
	requestedID  uuid.UUID
}

func (r *testRepository) FindByID(_ context.Context, id uuid.UUID) (trip.Trip, error) {
	r.requestedID = id
	return r.tripToReturn, r.err
}

func TestExecuteReturnsTrip(t *testing.T) {
	id := uuid.New()
	value := trip.Restore(id, uuid.New(), uuid.New(), 1, 2, 3, 4, 500, trip.StatusActive, time.Now().UTC(), nil, time.Now().UTC(), time.Now().UTC())
	repository := &testRepository{tripToReturn: value}

	result, err := New(repository).Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if repository.requestedID != id {
		t.Fatal("repository must receive requested id")
	}
	if result.ID() != id {
		t.Fatal("usecase must return repository trip")
	}
}

func TestExecuteReturnsRepositoryError(t *testing.T) {
	want := errors.New("database error")

	_, err := New(&testRepository{err: want}).Execute(context.Background(), uuid.New())

	if !errors.Is(err, want) {
		t.Fatalf("want repository error, got %v", err)
	}
}

func TestExecuteWithGomock(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	id := uuid.New()
	value := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())
	repository.EXPECT().FindByID(gomock.Any(), id).Return(value, nil)

	result, err := New(repository).Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if result.ID() != value.ID() {
		t.Fatal("usecase must return trip from mock repository")
	}
}
