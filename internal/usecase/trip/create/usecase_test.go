package create

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/model"
)

type testRepository struct {
	createdTrip    trip.Trip
	createdHistory trip.StatusHistory
	createError    error
	historyError   error
	findError      error
	tripToReturn   trip.Trip
}

func (r *testRepository) Create(_ context.Context, value trip.Trip) error {
	r.createdTrip = value
	return r.createError
}

func (r *testRepository) FindByID(_ context.Context, _ uuid.UUID) (trip.Trip, error) {
	if r.findError != nil {
		return trip.Trip{}, r.findError
	}
	return r.tripToReturn, nil
}

func (r *testRepository) CreateStatusHistory(_ context.Context, history trip.StatusHistory) error {
	r.createdHistory = history
	return r.historyError
}

type testTransaction struct {
	didRun bool
	err    error
}

func (t *testTransaction) Do(ctx context.Context, fn func(context.Context) error) error {
	t.didRun = true
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

func TestExecuteCreatesTripAndHistory(t *testing.T) {
	returnedTrip := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())
	repository := &testRepository{tripToReturn: returnedTrip}
	transaction := &testTransaction{}
	usecase := New(repository, transaction)

	result, err := usecase.Execute(context.Background(), model.CreateTrip{
		UserID: uuid.New(), DriverID: uuid.New(), StartLatitude: 10, StartLongitude: 20,
		EndLatitude: 30, EndLongitude: 40, Price: 500,
	})
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !transaction.didRun {
		t.Fatal("usecase must use transaction")
	}
	if repository.createdTrip.ID() == uuid.Nil {
		t.Fatal("repository must receive new trip")
	}
	if repository.createdHistory.TripID != repository.createdTrip.ID() {
		t.Fatal("history must belong to created trip")
	}
	if repository.createdHistory.ToStatus != trip.StatusActive {
		t.Fatal("first history status must be active")
	}
	if result.ID() != returnedTrip.ID() {
		t.Fatal("usecase must return trip from repository")
	}
}

func TestExecuteReturnsRepositoryError(t *testing.T) {
	want := errors.New("cannot save trip")
	usecase := New(&testRepository{createError: want}, &testTransaction{})

	_, err := usecase.Execute(context.Background(), model.CreateTrip{UserID: uuid.New(), DriverID: uuid.New()})

	if !errors.Is(err, want) {
		t.Fatalf("want repository error, got %v", err)
	}
}

func TestExecuteReturnsTransactionError(t *testing.T) {
	want := errors.New("cannot begin transaction")
	usecase := New(&testRepository{}, &testTransaction{err: want})

	_, err := usecase.Execute(context.Background(), model.CreateTrip{UserID: uuid.New(), DriverID: uuid.New()})

	if !errors.Is(err, want) {
		t.Fatalf("want transaction error, got %v", err)
	}
}

func TestExecuteWithGomock(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	metrics := NewMockmetrics(controller)
	result := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())

	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	repository.EXPECT().CreateStatusHistory(gomock.Any(), gomock.Any()).Return(nil)
	repository.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(result, nil)
	metrics.EXPECT().TripCreated(gomock.Any())

	returned, err := New(repository, transaction, metrics).Execute(context.Background(), model.CreateTrip{UserID: uuid.New(), DriverID: uuid.New()})
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if returned.ID() != result.ID() {
		t.Fatal("usecase must return trip from mock repository")
	}
}
