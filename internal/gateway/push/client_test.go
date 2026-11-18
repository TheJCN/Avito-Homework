package push

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRequestPosition(t *testing.T) {
	driverID := uuid.New()
	tripID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/push" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Content-Type: %s", r.Header.Get("Content-Type"))
		}
		var body struct {
			RequestID   string            `json:"request_id"`
			RecipientID uuid.UUID         `json:"recipient_id"`
			Kind        string            `json:"kind"`
			Data        map[string]string `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.RequestID == "" || body.RecipientID != driverID || body.Kind != "REQUEST_POSITION" || body.Data["trip_id"] != tripID.String() {
			t.Errorf("unexpected body: %+v", body)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(server.URL, time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := client.RequestPosition(context.Background(), driverID, tripID); err != nil {
		t.Fatalf("RequestPosition() error = %v", err)
	}
}

func TestRequestPositionClassifiesStatusCodes(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{status: http.StatusBadRequest, want: ErrClient},
		{status: http.StatusTooManyRequests, want: ErrClient},
		{status: http.StatusInternalServerError, want: ErrServer},
		{status: http.StatusServiceUnavailable, want: ErrServer},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()
			client, err := New(server.URL, time.Second)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			err = client.RequestPosition(context.Background(), uuid.New(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Fatalf("RequestPosition() error = %v, want %v", err, tt.want)
			}
			var statusError *HTTPStatusError
			if !errors.As(err, &statusError) || statusError.StatusCode != tt.status {
				t.Fatalf("status error = %v, want HTTP %d", err, tt.status)
			}
		})
	}
}

func TestRequestPositionTimeoutIsTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()
	client, err := New(server.URL, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.RequestPosition(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrTransport) {
		t.Fatalf("RequestPosition() error = %v, want transport error", err)
	}
}

func TestNewValidatesConfiguration(t *testing.T) {
	if _, err := New("", time.Second); err == nil {
		t.Fatal("empty URL must fail")
	}
	if _, err := New("http://push", 0); err == nil {
		t.Fatal("non-positive timeout must fail")
	}
}
