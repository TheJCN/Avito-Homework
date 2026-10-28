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

type testRepository struct {
	tripToFind     trip.Trip
	tripToReturn   trip.Trip
	findCalls      int
	updatedTrip    trip.Trip
	createdHistory trip.StatusHistory
	findError      error
	updateError    error
	historyError   error
}

func (r *testRepository) FindByID(_ context.Context, _ uuid.UUID) (trip.Trip, error) {
	r.findCalls++
	if r.findError != nil {
		return trip.Trip{}, r.findError
	}
	if r.findCalls == 1 {
		return r.tripToFind, nil
	}
	return r.tripToReturn, nil
}

func (r *testRepository) Update(_ context.Context, value trip.Trip) error {
	r.updatedTrip = value
	return r.updateError
}

func (r *testRepository) CreateStatusHistory(_ context.Context, history trip.StatusHistory) error {
	r.createdHistory = history
	return r.historyError
}

type testTransaction struct{ didRun bool }

func (t *testTransaction) Do(ctx context.Context, fn func(context.Context) error) error {
	t.didRun = true
	return fn(ctx)
}

func TestExecuteFinishesTrip(t *testing.T) {
	id := uuid.New()
	activeTrip := trip.Restore(id, uuid.New(), uuid.New(), 1, 2, 3, 4, 500, trip.StatusActive, time.Now().UTC(), nil, time.Now().UTC(), time.Now().UTC())
	completedTrip := activeTrip
	if err := completedTrip.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("prepare trip: %v", err)
	}
	repository := &testRepository{tripToFind: activeTrip, tripToReturn: completedTrip}
	transaction := &testTransaction{}

	result, err := New(repository, transaction).Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !transaction.didRun {
		t.Fatal("usecase must use transaction")
	}
	if repository.updatedTrip.Status() != trip.StatusCompleted {
		t.Fatal("repository must receive completed trip")
	}
	if repository.createdHistory.FromStatus == nil || *repository.createdHistory.FromStatus != trip.StatusActive {
		t.Fatal("history must keep old status")
	}
	if repository.createdHistory.ToStatus != trip.StatusCompleted {
		t.Fatal("history must keep new status")
	}
	if result.Status() != trip.StatusCompleted {
		t.Fatal("usecase must return completed trip")
	}
}

func TestExecuteDoesNotFinishAlreadyCompletedTrip(t *testing.T) {
	value := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())
	if err := value.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("prepare trip: %v", err)
	}
	repository := &testRepository{tripToFind: value}

	_, err := New(repository, &testTransaction{}).Execute(context.Background(), value.ID())

	if !errors.Is(err, trip.ErrCompleted) {
		t.Fatalf("want ErrCompleted, got %v", err)
	}
	if repository.updatedTrip.ID() != uuid.Nil {
		t.Fatal("completed trip must not be updated again")
	}
}

func TestExecuteReturnsFindError(t *testing.T) {
	want := errors.New("database error")

	_, err := New(&testRepository{findError: want}, &testTransaction{}).Execute(context.Background(), uuid.New())

	if !errors.Is(err, want) {
		t.Fatalf("want repository error, got %v", err)
	}
}

func TestExecuteWithGomock(t *testing.T) {
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

	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
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
