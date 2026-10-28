package get

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type testUsecase struct {
	requestedID uuid.UUID
	result      trip.Trip
	err         error
}

func (u *testUsecase) Execute(_ context.Context, id uuid.UUID) (trip.Trip, error) {
	u.requestedID = id
	return u.result, u.err
}

func TestHandleCallsUsecase(t *testing.T) {
	id := uuid.New()
	result := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())
	usecase := &testUsecase{result: result}

	returned, err := New(usecase).Handle(context.Background(), id)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if usecase.requestedID != id {
		t.Fatal("handler must pass id to usecase")
	}
	if returned.ID() != result.ID() {
		t.Fatal("handler must return usecase result")
	}
}

func TestHandleReturnsUsecaseError(t *testing.T) {
	want := errors.New("trip not found")

	_, err := New(&testUsecase{err: want}).Handle(context.Background(), uuid.New())

	if !errors.Is(err, want) {
		t.Fatalf("want usecase error, got %v", err)
	}
}
