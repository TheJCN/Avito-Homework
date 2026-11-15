package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testDatabase struct{ err error }

func (d testDatabase) Ping(_ context.Context) error { return d.err }

func TestHealthAlwaysReturnsOK(t *testing.T) {
	response := New(testDatabase{}, time.Second).Health()

	if string(response.Status) != "ok" {
		t.Fatalf("want ok, got %q", response.Status)
	}
}

func TestReadyReturnsOKWhenDatabaseWorks(t *testing.T) {
	response, ready := New(testDatabase{}, time.Second).Ready(context.Background())

	if !ready {
		t.Fatal("service must be ready")
	}
	if string(response.Status) != "ok" {
		t.Fatalf("want ok, got %q", response.Status)
	}
}

func TestReadyReturnsUnavailableWhenDatabaseFails(t *testing.T) {
	response, ready := New(testDatabase{err: errors.New("database is down")}, time.Second).Ready(context.Background())

	if ready {
		t.Fatal("service must not be ready")
	}
	if string(response.Status) != "unavailable" {
		t.Fatalf("want unavailable, got %q", response.Status)
	}
}
