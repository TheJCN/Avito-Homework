package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/TheJCN/Avito-Course/api"
	"github.com/TheJCN/Avito-Course/internal/mapper"
	"github.com/TheJCN/Avito-Course/internal/model"
	"github.com/google/uuid"
)

type TripService interface {
	Create(ctx context.Context, trip model.Trip, idempotencyKey *uuid.UUID, requestHash []byte) (model.Trip, bool, error)
	FindByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
	Finish(ctx context.Context, id uuid.UUID) (model.Trip, error)
}

type Handler struct {
	trips   TripService
	db      Pinger
	timeout time.Duration
}

var _ api.ServerInterface = (*Handler)(nil)

func New(
	trips TripService,
	db Pinger,
	timeout time.Duration,
) http.Handler {
	handler := &Handler{
		trips:   trips,
		db:      db,
		timeout: timeout,
	}

	return api.HandlerWithOptions(handler, api.ChiServerOptions{
		ErrorHandlerFunc: badRequest,
	})
}

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	params api.CreateTripParams,
) {
	data, requestHash, err := readTrip(r)
	if err != nil {
		badRequest(w, r, err)
		return
	}

	trip, replayed, err := h.trips.Create(
		r.Context(),
		mapper.TripFromRequest(data),
		params.IdempotencyKey,
		requestHash[:],
	)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if replayed {
		writeJSON(w, http.StatusOK, mapper.TripResponse(trip))
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	writeJSON(w, http.StatusCreated, mapper.TripResponse(trip))
}

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	id api.TripId,
) {
	trip, err := h.trips.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, mapper.TripResponse(trip))
}

func (h *Handler) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	id api.TripId,
) {
	trip, err := h.trips.Finish(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, mapper.TripResponse(trip))
}

func readTrip(r *http.Request) (api.TripData, [sha256.Size]byte, error) {
	var data api.TripData
	var hash [sha256.Size]byte

	body, err := io.ReadAll(io.LimitReader(r.Body, 1048577))
	if err != nil || len(body) > 1048576 {
		return data, hash, errors.New("invalid request body")
	}
	hash = sha256.Sum256(body)

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return data, hash, err
	}

	if !hasFields(fields,
		"user_id",
		"driver_id",
		"start_point",
		"end_point",
		"price",
	) {
		return data, hash, errors.New("missing fields")
	}

	if !validCoordinates(fields["start_point"]) ||
		!validCoordinates(fields["end_point"]) {
		return data, hash, errors.New("invalid coordinates")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&data); err != nil {
		return data, hash, err
	}

	if data.UserId == uuid.Nil ||
		data.DriverId == uuid.Nil ||
		data.Price < 0 {
		return data, hash, errors.New("invalid trip data")
	}

	if data.StartPoint.Latitude < -90 ||
		data.StartPoint.Latitude > 90 ||
		data.EndPoint.Latitude < -90 ||
		data.EndPoint.Latitude > 90 ||
		data.StartPoint.Longitude < -180 ||
		data.StartPoint.Longitude > 180 ||
		data.EndPoint.Longitude < -180 ||
		data.EndPoint.Longitude > 180 {
		return data, hash, errors.New("invalid coordinates")
	}

	return data, hash, nil
}

func hasFields(
	fields map[string]json.RawMessage,
	names ...string,
) bool {
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
