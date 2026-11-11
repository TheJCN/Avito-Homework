package create

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/api"
	"github.com/TheJCN/Avito-Course/internal/model"
)

var ErrInvalidRequest = errors.New("invalid request")

type Handler struct {
	usecase usecase
}

func New(usecase usecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(ctx context.Context, request *http.Request, idempotencyKey *uuid.UUID) (model.CreateTripResult, error) {
	data, hash, err := readData(request)
	if err != nil {
		return model.CreateTripResult{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	return h.usecase.Execute(ctx, model.CreateTrip{
		UserID:         data.UserId,
		DriverID:       data.DriverId,
		StartLatitude:  data.StartPoint.Latitude,
		StartLongitude: data.StartPoint.Longitude,
		EndLatitude:    data.EndPoint.Latitude,
		EndLongitude:   data.EndPoint.Longitude,
		Price:          data.Price,
		IdempotencyKey: idempotencyKey,
		RequestHash:    hash[:],
	})
}

func readData(request *http.Request) (api.TripData, [sha256.Size]byte, error) {
	var data api.TripData
	var hash [sha256.Size]byte
	body, err := io.ReadAll(io.LimitReader(request.Body, 1048577))
	if err != nil || len(body) > 1048576 {
		return data, hash, errors.New("invalid request body")
	}
	hash = sha256.Sum256(body)

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return data, hash, err
	}
	if !hasFields(fields, "user_id", "driver_id", "start_point", "end_point", "price") {
		return data, hash, errors.New("missing fields")
	}
	if !validCoordinates(fields["start_point"]) || !validCoordinates(fields["end_point"]) {
		return data, hash, errors.New("invalid coordinates")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return data, hash, err
	}
	return data, hash, nil
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
