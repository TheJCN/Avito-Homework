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

func TestExecuteCreatesTripAndHistory(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	returnedTrip := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())
	var createdTrip trip.Trip
	var createdHistory trip.StatusHistory
	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
	repository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, value trip.Trip) error {
		createdTrip = value
		return nil
	})
	repository.EXPECT().CreateStatusHistory(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, history trip.StatusHistory) error {
		createdHistory = history
		return nil
	})
	repository.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(returnedTrip, nil)
	usecase := New(repository, transaction)

	result, err := usecase.Execute(context.Background(), model.CreateTrip{
		UserID: uuid.New(), DriverID: uuid.New(), StartLatitude: 10, StartLongitude: 20,
		EndLatitude: 30, EndLongitude: 40, Price: 500,
	})
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if createdTrip.ID() == uuid.Nil {
		t.Fatal("repository must receive new trip")
	}
	if createdHistory.TripID != createdTrip.ID() {
		t.Fatal("history must belong to created trip")
	}
	if createdHistory.ToStatus != trip.StatusActive {
		t.Fatal("first history status must be active")
	}
	if result.Trip.ID() != returnedTrip.ID() {
		t.Fatal("usecase must return trip from repository")
	}
}

func TestExecuteReturnsRepositoryError(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	want := errors.New("cannot save trip")
	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
	repository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(want)
	usecase := New(repository, transaction)

	_, err := usecase.Execute(context.Background(), model.CreateTrip{UserID: uuid.New(), DriverID: uuid.New()})

	if !errors.Is(err, want) {
		t.Fatalf("want repository error, got %v", err)
	}
}

func TestExecuteReturnsTransactionError(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	want := errors.New("cannot begin transaction")
	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).Return(want)
	usecase := New(repository, transaction)

	_, err := usecase.Execute(context.Background(), model.CreateTrip{UserID: uuid.New(), DriverID: uuid.New()})

	if !errors.Is(err, want) {
		t.Fatalf("want transaction error, got %v", err)
	}
}

func TestExecuteRecordsMetric(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	metrics := NewMockmetrics(controller)
	result := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())

	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
	repository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	repository.EXPECT().CreateStatusHistory(gomock.Any(), gomock.Any()).Return(nil)
	repository.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(result, nil)
	metrics.EXPECT().TripCreated(gomock.Any())

	returned, err := New(repository, transaction, metrics).Execute(context.Background(), model.CreateTrip{UserID: uuid.New(), DriverID: uuid.New()})
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if returned.Trip.ID() != result.ID() {
		t.Fatal("usecase must return trip from mock repository")
	}
}

func TestExecuteDoesNotRecordMetricForReplay(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	transaction := NewMocktransactionManager(controller)
	metrics := NewMockmetrics(controller)
	idempotencyKey := uuid.New()
	existingID := uuid.New()
	existingTrip := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())

	transaction.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
	repository.EXPECT().ReserveIdempotencyKey(gomock.Any(), idempotencyKey, gomock.Any(), gomock.Any()).Return(&existingID, nil)
	repository.EXPECT().FindByID(gomock.Any(), existingID).Return(existingTrip, nil)

	result, err := New(repository, transaction, metrics).Execute(context.Background(), model.CreateTrip{
		UserID: uuid.New(), DriverID: uuid.New(), StartLatitude: 1, StartLongitude: 2,
		EndLatitude: 3, EndLongitude: 4, Price: 500, IdempotencyKey: &idempotencyKey,
	})
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if !result.Replayed {
		t.Fatal("result must be marked as replayed")
	}
}

func runTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
