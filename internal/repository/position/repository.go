package position

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
	"github.com/TheJCN/Avito-Course/internal/infra/postgres/transaction"
)

type Repository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func New(pool *pgxpool.Pool, timeout time.Duration) *Repository {
	return &Repository{pool: pool, timeout: timeout}
}

func (r *Repository) Create(ctx context.Context, value positiondomain.Position) (positiondomain.Position, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	row := r.executor(ctx).QueryRow(ctx, `
		INSERT INTO trip_positions (trip_id, latitude, longitude, recorded_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, trip_id, latitude, longitude, recorded_at, created_at`,
		value.TripID, value.Latitude, value.Longitude, value.RecordedAt,
	)
	result, err := scanPosition(row)
	if err != nil {
		return positiondomain.Position{}, fmt.Errorf("create position: %w", err)
	}
	return result, nil
}

func (r *Repository) UpdateLastPositionAt(ctx context.Context, tripID uuid.UUID, recordedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err := r.executor(ctx).Exec(ctx, `
		UPDATE trips
		SET last_position_at = GREATEST(COALESCE(last_position_at, $2), $2), updated_at = now()
		WHERE id = $1`, tripID, recordedAt)
	if err != nil {
		return fmt.Errorf("update last position time: %w", err)
	}
	return nil
}

func (r *Repository) ListByTripID(ctx context.Context, tripID uuid.UUID) ([]positiondomain.Position, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	rows, err := r.executor(ctx).Query(ctx, `
		SELECT id, trip_id, latitude, longitude, recorded_at, created_at
		FROM trip_positions
		WHERE trip_id = $1
		ORDER BY recorded_at, id`, tripID)
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}
	defer rows.Close()

	result := make([]positiondomain.Position, 0)
	for rows.Next() {
		value, err := scanPosition(rows)
		if err != nil {
			return nil, fmt.Errorf("read position: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read positions: %w", err)
	}
	return result, nil
}

type executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Repository) executor(ctx context.Context) executor {
	if tx := transaction.FromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}

type scanner interface{ Scan(...any) error }

func scanPosition(row scanner) (positiondomain.Position, error) {
	var value positiondomain.Position
	err := row.Scan(&value.ID, &value.TripID, &value.Latitude, &value.Longitude, &value.RecordedAt, &value.CreatedAt)
	return value, err
}
