package trip

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
)

type testScanner struct {
	id        uuid.UUID
	userID    uuid.UUID
	driverID  uuid.UUID
	finished  *time.Time
	scanError error
}

func (s testScanner) Scan(dest ...any) error {
	if s.scanError != nil {
		return s.scanError
	}
	*dest[0].(*uuid.UUID) = s.id
	*dest[1].(*uuid.UUID) = s.userID
	*dest[2].(*uuid.UUID) = s.driverID
	*dest[3].(*float64) = 10
	*dest[4].(*float64) = 20
	*dest[5].(*float64) = 30
	*dest[6].(*float64) = 40
	*dest[7].(*int64) = 500
	*dest[8].(*string) = trip.StatusActive
	*dest[9].(*time.Time) = time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	*dest[10].(**time.Time) = s.finished
	*dest[11].(*time.Time) = time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	*dest[12].(*time.Time) = time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	return nil
}

func TestScanTripCreatesDomainTrip(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	driverID := uuid.New()

	value, err := scanTrip(testScanner{id: id, userID: userID, driverID: driverID})
	if err != nil {
		t.Fatalf("scanTrip returned error: %v", err)
	}
	if value.ID() != id || value.UserID() != userID || value.DriverID() != driverID {
		t.Fatal("scanTrip must keep ids")
	}
	if value.Status() != trip.StatusActive || value.Price() != 500 {
		t.Fatal("scanTrip must keep trip fields")
	}
}

func TestScanTripReturnsScannerError(t *testing.T) {
	want := errors.New("cannot read row")

	_, err := scanTrip(testScanner{scanError: want})

	if !errors.Is(err, want) {
		t.Fatalf("want scanner error, got %v", err)
	}
}

func TestWriteErrorMapsActiveDriverConstraint(t *testing.T) {
	err := writeError("create trip", &pgconn.PgError{Code: "23505", ConstraintName: "trips_active_driver_idx"})

	if !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("want ErrDriverBusy, got %v", err)
	}
}
