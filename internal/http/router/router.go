package router

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/TheJCN/Avito-Course/api"
	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/http/domainerror"
	"github.com/TheJCN/Avito-Course/internal/http/health"
	createhttp "github.com/TheJCN/Avito-Course/internal/http/trip/create"
	finishhttp "github.com/TheJCN/Avito-Course/internal/http/trip/finish"
	gethttp "github.com/TheJCN/Avito-Course/internal/http/trip/get"
)

type Handler struct {
	logger      *slog.Logger
	create      *createhttp.Handler
	get         *gethttp.Handler
	finish      *finishhttp.Handler
	health      *health.Handler
	middlewares []func(http.Handler) http.Handler
}

func New(
	logger *slog.Logger,
	create *createhttp.Handler,
	get *gethttp.Handler,
	finish *finishhttp.Handler,
	health *health.Handler,
	middlewares ...func(http.Handler) http.Handler,
) http.Handler {
	handler := &Handler{logger: logger, create: create, get: get, finish: finish, health: health, middlewares: middlewares}
	mux := chi.NewRouter()
	for _, middleware := range handler.middlewares {
		mux.Use(middleware)
	}
	return api.HandlerWithOptions(handler, api.ChiServerOptions{BaseRouter: mux, ErrorHandlerFunc: handler.badRequest})
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	value, err := h.create.Handle(r.Context(), r)
	if err != nil {
		if errors.Is(err, createhttp.ErrInvalidRequest) {
			h.badRequest(w, r, err)
			return
		}
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+value.ID().String())
	h.writeJSON(w, http.StatusCreated, tripResponse(value))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	value, err := h.get.Handle(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, tripResponse(value))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	value, err := h.finish.Handle(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, tripResponse(value))
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, h.health.Health())
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	response, ready := h.health.Ready(r.Context())
	if !ready {
		h.writeJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	h.writeJSON(w, http.StatusOK, response)
}

func tripResponse(value trip.Trip) api.Trip {
	return api.Trip{
		Id:       value.ID(),
		UserId:   value.UserID(),
		DriverId: value.DriverID(),
		StartPoint: api.Coordinates{
			Latitude:  value.StartLatitude(),
			Longitude: value.StartLongitude(),
		},
		EndPoint: api.Coordinates{
			Latitude:  value.EndLatitude(),
			Longitude: value.EndLongitude(),
		},
		Price:      value.Price(),
		Status:     api.TripStatus(value.Status()),
		StartedAt:  value.StartedAt(),
		FinishedAt: value.FinishedAt(),
	}
}

func (h *Handler) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.DebugContext(r.Context(), "invalid request", "error", err)
	path := r.URL.Path
	h.writeProblem(w, http.StatusBadRequest, api.Problem{
		Type:     "https://tripgo.example/problems/invalid-request",
		Title:    "Invalid request",
		Status:   http.StatusBadRequest,
		Code:     "invalid_request",
		Detail:   stringPointer("Request validation failed"),
		Instance: &path,
	})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, problem := domainerror.Problem(err, r.URL.Path)
	if status == http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "request failed", "error", err)
	}
	h.writeProblem(w, status, problem)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.logger.Error("write response", "error", err)
	}
}

func (h *Handler) writeProblem(w http.ResponseWriter, status int, value api.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.logger.Error("write problem", "error", err)
	}
}

func stringPointer(value string) *string {
	return &value
}
