package create

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/model"
)

func TestHandleReadsRequestAndCallsUsecase(t *testing.T) {
	controller := gomock.NewController(t)
	usecase := NewMockusecase(controller)
	userID := uuid.New()
	driverID := uuid.New()
	result := trip.New(userID, driverID, 10, 20, 30, 40, 500, time.Now().UTC())
	usecase.EXPECT().Execute(gomock.Any(), gomock.Cond(func(input model.CreateTrip) bool {
		return input.UserID == userID && input.DriverID == driverID && input.Price == 500
	})).Return(model.CreateTripResult{Trip: result}, nil)
	request := httptest.NewRequest("POST", "/api/v1/trips", validRequestBody(userID, driverID))

	returned, err := New(usecase).Handle(context.Background(), request, nil)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if returned.Trip.ID() != result.ID() {
		t.Fatal("handler must return usecase result")
	}
}

func TestHandleRejectsBadRequest(t *testing.T) {
	controller := gomock.NewController(t)
	usecase := NewMockusecase(controller)
	request := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(`{"price":500}`))

	_, err := New(usecase).Handle(context.Background(), request, nil)

	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
}

func TestReadDataPassesCoordinatesToUsecase(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	body := `{"user_id":"` + userID.String() + `","driver_id":"` + driverID.String() + `","start_point":{"latitude":100,"longitude":20},"end_point":{"latitude":30,"longitude":40},"price":500}`
	request := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(body))

	data, _, err := readData(request)
	if err != nil {
		t.Fatalf("readData returned error: %v", err)
	}
	if data.StartPoint.Latitude != 100 {
		t.Fatal("handler must leave business validation to usecase")
	}
}

func TestReadDataRejectsUnknownField(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	body := `{"user_id":"` + userID.String() + `","driver_id":"` + driverID.String() + `","start_point":{"latitude":10,"longitude":20},"end_point":{"latitude":30,"longitude":40},"price":500,"unknown":true}`
	request := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(body))

	_, _, err := readData(request)

	if err == nil {
		t.Fatal("unknown field must return error")
	}
}

func validRequestBody(userID, driverID uuid.UUID) *strings.Reader {
	body := `{"user_id":"` + userID.String() + `","driver_id":"` + driverID.String() + `","start_point":{"latitude":10,"longitude":20},"end_point":{"latitude":30,"longitude":40},"price":500}`
	return strings.NewReader(body)
}
