package trip

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

type Trip struct {
	id             uuid.UUID
	userID         uuid.UUID
	driverID       uuid.UUID
	startLatitude  float64
	startLongitude float64
	endLatitude    float64
	endLongitude   float64
	price          int64
	status         string
	startedAt      time.Time
	finishedAt     *time.Time
	createdAt      time.Time
	updatedAt      time.Time
}

type StatusHistory struct {
	TripID     uuid.UUID
	FromStatus *string
	ToStatus   string
	Reason     *string
}

func New(
	userID uuid.UUID,
	driverID uuid.UUID,
	startLatitude float64,
	startLongitude float64,
	endLatitude float64,
	endLongitude float64,
	price int64,
	startedAt time.Time,
) Trip {
	return Trip{
		id:             uuid.New(),
		userID:         userID,
		driverID:       driverID,
		startLatitude:  startLatitude,
		startLongitude: startLongitude,
		endLatitude:    endLatitude,
		endLongitude:   endLongitude,
		price:          price,
		status:         StatusActive,
		startedAt:      startedAt,
	}
}

func Restore(
	id uuid.UUID,
	userID uuid.UUID,
	driverID uuid.UUID,
	startLatitude float64,
	startLongitude float64,
	endLatitude float64,
	endLongitude float64,
	price int64,
	status string,
	startedAt time.Time,
	finishedAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) Trip {
	finishedAt = copyTime(finishedAt)

	return Trip{
		id:             id,
		userID:         userID,
		driverID:       driverID,
		startLatitude:  startLatitude,
		startLongitude: startLongitude,
		endLatitude:    endLatitude,
		endLongitude:   endLongitude,
		price:          price,
		status:         status,
		startedAt:      startedAt,
		finishedAt:     finishedAt,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

func (t *Trip) Finish(finishedAt time.Time) error {
	if t.status != StatusActive {
		return ErrCompleted
	}

	t.status = StatusCompleted
	t.finishedAt = copyTime(&finishedAt)
	return nil
}

func (t Trip) ID() uuid.UUID { return t.id }

func (t Trip) UserID() uuid.UUID { return t.userID }

func (t Trip) DriverID() uuid.UUID { return t.driverID }

func (t Trip) StartLatitude() float64 { return t.startLatitude }

func (t Trip) StartLongitude() float64 { return t.startLongitude }

func (t Trip) EndLatitude() float64 { return t.endLatitude }

func (t Trip) EndLongitude() float64 { return t.endLongitude }

func (t Trip) Price() int64 { return t.price }

func (t Trip) Status() string { return t.status }

func (t Trip) StartedAt() time.Time { return t.startedAt }

func (t Trip) FinishedAt() *time.Time { return copyTime(t.finishedAt) }

func (t Trip) CreatedAt() time.Time { return t.createdAt }

func (t Trip) UpdatedAt() time.Time { return t.updatedAt }

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	copy := *value
	return &copy
}
