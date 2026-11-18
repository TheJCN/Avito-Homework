package create

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
)

type usecaseStub struct {
	latitude   float64
	longitude  float64
	recordedAt time.Time
}

func (s *usecaseStub) Execute(_ context.Context, _ uuid.UUID, latitude, longitude float64, recordedAt time.Time) (positiondomain.Position, error) {
	s.latitude, s.longitude, s.recordedAt = latitude, longitude, recordedAt
	return positiondomain.Position{ID: 1}, nil
}

func TestHandlerHandle(t *testing.T) {
	stub := &usecaseStub{}
	handler := New(stub)
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"coordinates":{"latitude":59.93,"longitude":30.31},"recorded_at":"2026-11-16T12:00:00Z"}`))

	result, err := handler.Handle(context.Background(), request, uuid.New())
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.ID != 1 {
		t.Fatalf("Handle() result ID = %d, want 1", result.ID)
	}
	if stub.latitude != 59.93 || stub.longitude != 30.31 || !stub.recordedAt.Equal(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected request data: %+v", stub)
	}
}

func TestHandlerRejectsInvalidBody(t *testing.T) {
	tests := []string{
		`{"coordinates":{"latitude":59.93},"recorded_at":"2026-11-16T12:00:00Z"}`,
		`{"coordinates":{"latitude":59.93,"longitude":30.31},"recorded_at":"bad"}`,
		`{"coordinates":{"latitude":59.93,"longitude":30.31},"recorded_at":"2026-11-16T12:00:00Z","extra":true}`,
	}
	for _, body := range tests {
		request := httptest.NewRequest("POST", "/", strings.NewReader(body))
		_, err := New(&usecaseStub{}).Handle(context.Background(), request, uuid.New())
		if !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("Handle() error = %v, want ErrInvalidRequest", err)
		}
	}
}
