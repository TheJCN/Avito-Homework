package create

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/model"
)

type testUsecase struct {
	input  model.CreateTrip
	result trip.Trip
	err    error
}

func (u *testUsecase) Execute(_ context.Context, input model.CreateTrip) (trip.Trip, error) {
	u.input = input
	return u.result, u.err
}

func TestHandleReadsRequestAndCallsUsecase(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	result := trip.New(userID, driverID, 10, 20, 30, 40, 500, time.Now().UTC())
	usecase := &testUsecase{result: result}
	request := httptest.NewRequest("POST", "/api/v1/trips", validRequestBody(userID, driverID))

	returned, err := New(usecase).Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if usecase.input.UserID != userID || usecase.input.DriverID != driverID {
		t.Fatal("handler must pass ids to usecase")
	}
	if usecase.input.Price != 500 {
		t.Fatal("handler must pass price to usecase")
	}
	if returned.ID() != result.ID() {
		t.Fatal("handler must return usecase result")
	}
}

func TestHandleRejectsBadRequest(t *testing.T) {
	request := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(`{"price":500}`))

	_, err := New(&testUsecase{}).Handle(context.Background(), request)

	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
}

func TestReadDataRejectsWrongCoordinates(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	body := `{"user_id":"` + userID.String() + `","driver_id":"` + driverID.String() + `","start_point":{"latitude":100,"longitude":20},"end_point":{"latitude":30,"longitude":40},"price":500}`
	request := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(body))

	_, err := readData(request)

	if err == nil {
		t.Fatal("wrong coordinates must return error")
	}
}

func TestReadDataRejectsUnknownField(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	body := `{"user_id":"` + userID.String() + `","driver_id":"` + driverID.String() + `","start_point":{"latitude":10,"longitude":20},"end_point":{"latitude":30,"longitude":40},"price":500,"unknown":true}`
	request := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(body))

	_, err := readData(request)

	if err == nil {
		t.Fatal("unknown field must return error")
	}
}

func validRequestBody(userID, driverID uuid.UUID) *strings.Reader {
	body := `{"user_id":"` + userID.String() + `","driver_id":"` + driverID.String() + `","start_point":{"latitude":10,"longitude":20},"end_point":{"latitude":30,"longitude":40},"price":500}`
	return strings.NewReader(body)
}
