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

func TestHandleCallsUsecase(t *testing.T) {
	controller := gomock.NewController(t)
	usecase := NewMockusecase(controller)
	id := uuid.New()
	result := trip.New(uuid.New(), uuid.New(), 1, 2, 3, 4, 500, time.Now().UTC())
	usecase.EXPECT().Execute(gomock.Any(), id).Return(result, nil)

	returned, err := New(usecase).Handle(context.Background(), id)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if returned.ID() != result.ID() {
		t.Fatal("handler must return usecase result")
	}
}

func TestHandleReturnsUsecaseError(t *testing.T) {
	controller := gomock.NewController(t)
	usecase := NewMockusecase(controller)
	want := errors.New("trip not found")
	id := uuid.New()
	usecase.EXPECT().Execute(gomock.Any(), id).Return(trip.Trip{}, want)

	_, err := New(usecase).Handle(context.Background(), id)

	if !errors.Is(err, want) {
		t.Fatalf("want usecase error, got %v", err)
	}
}
