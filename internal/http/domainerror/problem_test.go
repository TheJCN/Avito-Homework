package domainerror

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

func TestProblemMapsKnownErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"trip not found", trip.ErrNotFound, http.StatusNotFound, "trip_not_found"},
		{"driver busy", trip.ErrDriverBusy, http.StatusConflict, "driver_busy"},
		{"trip completed", trip.ErrCompleted, http.StatusConflict, "trip_completed"},
		{"unknown error", fmt.Errorf("database is unavailable"), http.StatusInternalServerError, "internal_error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, problem := Problem(fmt.Errorf("operation failed: %w", test.err), "/api/v1/trips/1")

			if status != test.status {
				t.Fatalf("want status %d, got %d", test.status, status)
			}
			if problem.Code != test.code {
				t.Fatalf("want code %q, got %q", test.code, problem.Code)
			}
			if problem.Instance == nil || *problem.Instance != "/api/v1/trips/1" {
				t.Fatal("problem must contain request path")
			}
		})
	}
}
