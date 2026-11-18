package position

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	now := time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		latitude   float64
		longitude  float64
		recordedAt time.Time
		wantErr    bool
	}{
		{name: "valid", latitude: 59.93, longitude: 30.31, recordedAt: now},
		{name: "one minute clock skew is allowed", latitude: 90, longitude: 180, recordedAt: now.Add(time.Minute)},
		{name: "latitude out of range", latitude: 90.1, longitude: 0, recordedAt: now, wantErr: true},
		{name: "longitude out of range", latitude: 0, longitude: -180.1, recordedAt: now, wantErr: true},
		{name: "nan", latitude: math.NaN(), longitude: 0, recordedAt: now, wantErr: true},
		{name: "zero time", latitude: 0, longitude: 0, wantErr: true},
		{name: "too far in future", latitude: 0, longitude: 0, recordedAt: now.Add(time.Minute + time.Nanosecond), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.latitude, tt.longitude, tt.recordedAt, now)
			if tt.wantErr != errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate() error = %v, want invalid = %v", err, tt.wantErr)
			}
		})
	}
}
