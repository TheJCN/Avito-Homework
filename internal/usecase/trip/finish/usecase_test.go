package finish

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

func TestExecuteFinishesTrip(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	id := uuid.New()
	activeTrip := trip.Restore(id, uuid.New(), uuid.New(), 1, 2, 3, 4, 500, trip.StatusActive, time.Now().UTC(), nil, time.Now().UTC(), time.Now().UTC())
	completedTrip := activeTrip
	if err := completedTrip.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("prepare trip: %v", err)
	}
	var updatedTrip trip.Trip
	var createdHistory trip.StatusHistory
	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
	gomock.InOrder(
		repository.EXPECT().FindByID(gomock.Any(), id).Return(activeTrip, nil),
		repository.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, value trip.Trip) error {
			updatedTrip = value
			return nil
		}),
		repository.EXPECT().CreateStatusHistory(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, history trip.StatusHistory) error {
			createdHistory = history
			return nil
		}),
		repository.EXPECT().FindByID(gomock.Any(), id).Return(completedTrip, nil),
	)

	result, err := New(repository, transaction).Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if updatedTrip.Status() != trip.StatusCompleted {
		t.Fatal("repository must receive completed trip")
	}
	if createdHistory.FromStatus == nil || *createdHistory.FromStatus != trip.StatusActive {
		t.Fatal("history must keep old status")
	}
	if createdHistory.ToStatus != trip.StatusCompleted {
		t.Fatal("history must keep new status")
	}
	if result.Status() != trip.StatusCompleted {
		t.Fatal("usecase must return completed trip")
	}
}

func TestExecuteDoesNotFinishAlreadyCompletedTrip(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	value := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())
	if err := value.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("prepare trip: %v", err)
	}
	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
	repository.EXPECT().FindByID(gomock.Any(), value.ID()).Return(value, nil)

	_, err := New(repository, transaction).Execute(context.Background(), value.ID())

	if !errors.Is(err, trip.ErrCompleted) {
		t.Fatalf("want ErrCompleted, got %v", err)
	}
}

func TestExecuteReturnsFindError(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	want := errors.New("database error")
	id := uuid.New()
	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
	repository.EXPECT().FindByID(gomock.Any(), id).Return(trip.Trip{}, want)

	_, err := New(repository, transaction).Execute(context.Background(), id)

	if !errors.Is(err, want) {
		t.Fatalf("want repository error, got %v", err)
	}
}

func TestExecuteRecordsMetric(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	metrics := NewMockmetrics(controller)
	id := uuid.New()
	active := trip.Restore(id, uuid.New(), uuid.New(), 1, 2, 3, 4, 500, trip.StatusActive, time.Now().UTC(), nil, time.Now().UTC(), time.Now().UTC())
	completed := active
	if err := completed.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("prepare trip: %v", err)
	}

	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
	gomock.InOrder(
		repository.EXPECT().FindByID(gomock.Any(), id).Return(active, nil),
		repository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil),
		repository.EXPECT().CreateStatusHistory(gomock.Any(), gomock.Any()).Return(nil),
		repository.EXPECT().FindByID(gomock.Any(), id).Return(completed, nil),
	)
	metrics.EXPECT().TripCompleted(gomock.Any())

	returned, err := New(repository, transaction, metrics).Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if returned.Status() != trip.StatusCompleted {
		t.Fatal("usecase must return completed trip")
	}
}

func runTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
