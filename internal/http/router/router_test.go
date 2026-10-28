package router

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/http/health"
	createhttp "github.com/TheJCN/Avito-Course/internal/http/trip/create"
	finishhttp "github.com/TheJCN/Avito-Course/internal/http/trip/finish"
	gethttp "github.com/TheJCN/Avito-Course/internal/http/trip/get"
	"github.com/TheJCN/Avito-Course/internal/model"
)

type createUsecase struct{ result trip.Trip }

func (u createUsecase) Execute(_ context.Context, _ model.CreateTrip) (trip.Trip, error) {
	return u.result, nil
}

type getUsecase struct{ err error }

func (u getUsecase) Execute(_ context.Context, _ uuid.UUID) (trip.Trip, error) {
	return trip.Trip{}, u.err
}

type finishUsecase struct{}

func (finishUsecase) Execute(_ context.Context, _ uuid.UUID) (trip.Trip, error) {
	return trip.Trip{}, nil
}

type database struct{ err error }

func (d database) Ping(_ context.Context) error { return d.err }

func TestServerCreatesTrip(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	value := trip.New(userID, driverID, 10, 20, 30, 40, 500, time.Now().UTC())
	server := New(
		slog.Default(),
		createhttp.New(createUsecase{result: value}),
		gethttp.New(getUsecase{}),
		finishhttp.New(finishUsecase{}),
		health.New(database{}, time.Second),
	)
	body := `{"user_id":"` + userID.String() + `","driver_id":"` + driverID.String() + `","start_point":{"latitude":10,"longitude":20},"end_point":{"latitude":30,"longitude":40},"price":500}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("want status 201, got %d", recorder.Code)
	}
	if recorder.Header().Get("Location") != "/api/v1/trips/"+value.ID().String() {
		t.Fatal("created trip must have location header")
	}
	if !strings.Contains(recorder.Body.String(), value.ID().String()) {
		t.Fatal("response must contain created trip")
	}
}

func TestServerMapsTripNotFoundTo404(t *testing.T) {
	server := New(
		slog.Default(),
		createhttp.New(createUsecase{}),
		gethttp.New(getUsecase{err: fmt.Errorf("get trip: %w", trip.ErrNotFound)}),
		finishhttp.New(finishUsecase{}),
		health.New(database{}, time.Second),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+uuid.New().String(), nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("want status 404, got %d", recorder.Code)
	}
}

func TestServerReturnsBadRequestForWrongTripID(t *testing.T) {
	server := New(
		slog.Default(),
		createhttp.New(createUsecase{}),
		gethttp.New(getUsecase{}),
		finishhttp.New(finishUsecase{}),
		health.New(database{}, time.Second),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/not-a-uuid", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d", recorder.Code)
	}
}

func TestServerReturnsUnavailableWhenDatabaseFails(t *testing.T) {
	server := New(
		slog.Default(),
		createhttp.New(createUsecase{}),
		gethttp.New(getUsecase{}),
		finishhttp.New(finishUsecase{}),
		health.New(database{err: errors.New("database is down")}, time.Second),
	)
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("want status 503, got %d", recorder.Code)
	}
}
