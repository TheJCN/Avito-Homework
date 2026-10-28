package domainerror

import (
	"errors"
	"net/http"

	"github.com/TheJCN/Avito-Course/api"
	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

func Problem(err error, path string) (int, api.Problem) {
	status := http.StatusInternalServerError
	code := "internal_error"
	name := "internal-error"
	title := "Internal Server Error"
	detail := "Internal server error"

	switch {
	case errors.Is(err, trip.ErrNotFound):
		status = http.StatusNotFound
		code = "trip_not_found"
		name = "trip-not-found"
		title = "Trip not found"
		detail = "Trip was not found"
	case errors.Is(err, trip.ErrDriverBusy):
		status = http.StatusConflict
		code = "driver_busy"
		name = "driver-busy"
		title = "Driver busy"
		detail = "Driver already has an active trip"
	case errors.Is(err, trip.ErrCompleted):
		status = http.StatusConflict
		code = "trip_completed"
		name = "trip-completed"
		title = "Trip completed"
		detail = "Trip is already completed"
	}

	return status, api.Problem{
		Type:     "https://tripgo.example/problems/" + name,
		Title:    title,
		Status:   int32(status),
		Code:     code,
		Detail:   &detail,
		Instance: &path,
	}
}
