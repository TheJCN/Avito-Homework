package trip

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewCreatesActiveTrip(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	startedAt := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)

	value := New(userID, driverID, 10, 20, 30, 40, 500, startedAt)

	if value.ID() == uuid.Nil {
		t.Fatal("new trip must have an id")
	}
	if value.UserID() != userID {
		t.Fatal("trip must keep user id")
	}
	if value.DriverID() != driverID {
		t.Fatal("trip must keep driver id")
	}
	if value.Status() != StatusActive {
		t.Fatal("new trip must be active")
	}
	if value.FinishedAt() != nil {
		t.Fatal("new trip must not have finish time")
	}
}

func TestFinishCompletesActiveTrip(t *testing.T) {
	value := New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, time.Now().UTC())
	finishedAt := time.Date(2026, 10, 28, 11, 0, 0, 0, time.UTC)

	err := value.Finish(finishedAt)
	if err != nil {
		t.Fatalf("finish returned error: %v", err)
	}
	if value.Status() != StatusCompleted {
		t.Fatal("finished trip must be completed")
	}
	if value.FinishedAt() == nil || !value.FinishedAt().Equal(finishedAt) {
		t.Fatal("finished trip must keep finish time")
	}
}

func TestFinishCompletedTripReturnsError(t *testing.T) {
	value := New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, time.Now().UTC())
	if err := value.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("first finish returned error: %v", err)
	}

	err := value.Finish(time.Now().UTC())

	if !errors.Is(err, ErrCompleted) {
		t.Fatalf("want ErrCompleted, got %v", err)
	}
}

func TestFinishedAtReturnsCopy(t *testing.T) {
	finishedAt := time.Date(2026, 10, 28, 11, 0, 0, 0, time.UTC)
	value := Restore(uuid.New(), uuid.New(), uuid.New(), 10, 20, 30, 40, 500, StatusCompleted, time.Now().UTC(), &finishedAt, time.Now().UTC(), time.Now().UTC())

	returned := value.FinishedAt()
	changed := returned.Add(time.Hour)

	if value.FinishedAt().Equal(changed) {
		t.Fatal("changing returned time must not change trip")
	}
}
