package position

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type Position struct {
	ID         int64
	TripID     uuid.UUID
	Latitude   float64
	Longitude  float64
	RecordedAt time.Time
	CreatedAt  time.Time
}

func Validate(latitude, longitude float64, recordedAt, now time.Time) error {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 {
		return fmt.Errorf("%w: latitude out of range", ErrInvalid)
	}
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		return fmt.Errorf("%w: longitude out of range", ErrInvalid)
	}
	if recordedAt.IsZero() || recordedAt.After(now.Add(time.Minute)) {
		return fmt.Errorf("%w: invalid recorded_at", ErrInvalid)
	}
	return nil
}
