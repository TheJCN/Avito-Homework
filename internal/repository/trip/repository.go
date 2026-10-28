package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/infra/postgres/transaction"
)

type Repository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(pool *pgxpool.Pool, timeout time.Duration) *Repository {
	return &Repository{pool: pool, timeout: timeout}
}

func (r *Repository) Create(ctx context.Context, value trip.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query, args, err := sq.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").
		Values(value.ID(), value.UserID(), value.DriverID(), value.StartLatitude(), value.StartLongitude(), value.EndLatitude(), value.EndLongitude(), value.Price(), value.Status(), value.StartedAt(), value.FinishedAt()).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create trip query: %w", err)
	}

	if _, err := r.executor(ctx).Exec(ctx, query, args...); err != nil {
		return writeError("create trip", err)
	}

	return nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query, args, err := selectTrips().Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return trip.Trip{}, fmt.Errorf("build find trip query: %w", err)
	}

	value, err := scanTrip(r.executor(ctx).QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, trip.ErrNotFound
	}
	if err != nil {
		return trip.Trip{}, fmt.Errorf("find trip: %w", err)
	}

	return value, nil
}

func (r *Repository) FindAll(ctx context.Context) ([]trip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query, args, err := selectTrips().OrderBy("started_at", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find trips query: %w", err)
	}

	rows, err := r.executor(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find trips: %w", err)
	}
	defer rows.Close()

	trips := make([]trip.Trip, 0)
	for rows.Next() {
		value, err := scanTrip(rows)
		if err != nil {
			return nil, fmt.Errorf("read trip: %w", err)
		}
		trips = append(trips, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read trips: %w", err)
	}

	return trips, nil
}

func (r *Repository) Update(ctx context.Context, value trip.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query, args, err := sq.Update("trips").
		Set("status", value.Status()).
		Set("finished_at", value.FinishedAt()).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"id": value.ID(), "status": trip.StatusActive}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip query: %w", err)
	}

	result, err := r.executor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return writeError("update trip", err)
	}
	if result.RowsAffected() == 0 {
		return r.updateError(ctx, value.ID())
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query, args, err := sq.Delete("trips").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete trip query: %w", err)
	}

	result, err := r.executor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete trip: %w", err)
	}
	if result.RowsAffected() == 0 {
		return trip.ErrNotFound
	}

	return nil
}

func (r *Repository) CreateStatusHistory(ctx context.Context, history trip.StatusHistory) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query, args, err := sq.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(history.TripID, history.FromStatus, history.ToStatus, history.Reason).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create history query: %w", err)
	}

	if _, err := r.executor(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("create trip history: %w", err)
	}

	return nil
}

func (r *Repository) executor(ctx context.Context) executor {
	if tx := transaction.FromContext(ctx); tx != nil {
		return tx
	}

	return r.pool
}

func (r *Repository) updateError(ctx context.Context, id uuid.UUID) error {
	value, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if value.Status() == trip.StatusCompleted {
		return trip.ErrCompleted
	}

	return errors.New("update trip: no rows updated")
}

func selectTrips() sq.SelectBuilder {
	return sq.Select("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at", "finished_at", "created_at", "updated_at").
		From("trips").
		PlaceholderFormat(sq.Dollar)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTrip(row scanner) (trip.Trip, error) {
	var id uuid.UUID
	var userID uuid.UUID
	var driverID uuid.UUID
	var startLatitude float64
	var startLongitude float64
	var endLatitude float64
	var endLongitude float64
	var price int64
	var status string
	var startedAt time.Time
	var finishedAt *time.Time
	var createdAt time.Time
	var updatedAt time.Time

	err := row.Scan(
		&id,
		&userID,
		&driverID,
		&startLatitude,
		&startLongitude,
		&endLatitude,
		&endLongitude,
		&price,
		&status,
		&startedAt,
		&finishedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return trip.Trip{}, err
	}

	return trip.Restore(
		id,
		userID,
		driverID,
		startLatitude,
		startLongitude,
		endLatitude,
		endLongitude,
		price,
		status,
		startedAt,
		finishedAt,
		createdAt,
		updatedAt,
	), nil
}

func writeError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_active_driver_idx" {
		return fmt.Errorf("%s: %w", operation, trip.ErrDriverBusy)
	}

	return fmt.Errorf("%s: %w", operation, err)
}
