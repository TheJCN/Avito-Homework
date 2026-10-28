package create

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/api"
	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/model"
)

var ErrInvalidRequest = errors.New("invalid request")

type Handler struct {
	usecase usecase
}

func New(usecase usecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(ctx context.Context, request *http.Request) (trip.Trip, error) {
	data, err := readData(request)
	if err != nil {
		return trip.Trip{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	return h.usecase.Execute(ctx, model.CreateTrip{
		UserID:         data.UserId,
		DriverID:       data.DriverId,
		StartLatitude:  data.StartPoint.Latitude,
		StartLongitude: data.StartPoint.Longitude,
		EndLatitude:    data.EndPoint.Latitude,
		EndLongitude:   data.EndPoint.Longitude,
		Price:          data.Price,
	})
}

func readData(request *http.Request) (api.TripData, error) {
	var data api.TripData
	body, err := io.ReadAll(io.LimitReader(request.Body, 1048577))
	if err != nil || len(body) > 1048576 {
		return data, errors.New("invalid request body")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return data, err
	}
	if !hasFields(fields, "user_id", "driver_id", "start_point", "end_point", "price") {
		return data, errors.New("missing fields")
	}
	if !validCoordinates(fields["start_point"]) || !validCoordinates(fields["end_point"]) {
		return data, errors.New("invalid coordinates")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return data, err
	}
	if data.UserId == uuid.Nil || data.DriverId == uuid.Nil || data.Price < 0 {
		return data, errors.New("invalid trip data")
	}
	if data.StartPoint.Latitude < -90 || data.StartPoint.Latitude > 90 || data.EndPoint.Latitude < -90 || data.EndPoint.Latitude > 90 || data.StartPoint.Longitude < -180 || data.StartPoint.Longitude > 180 || data.EndPoint.Longitude < -180 || data.EndPoint.Longitude > 180 {
		return data, errors.New("invalid coordinates")
	}

	return data, nil
}

func hasFields(fields map[string]json.RawMessage, names ...string) bool {
	if len(fields) != len(names) {
		return false
	}

	for _, name := range names {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return false
		}
	}

	return true
}

func validCoordinates(raw json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return false
	}

	return hasFields(fields, "latitude", "longitude")
}
