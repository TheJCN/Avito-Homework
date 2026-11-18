//go:build integration

package create_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool" //nolint:depguard // integration test verifies the transaction against PostgreSQL

	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	"github.com/TheJCN/Avito-Course/internal/infra/postgres/transaction"
	positionrepository "github.com/TheJCN/Avito-Course/internal/repository/position"
	triprepository "github.com/TheJCN/Avito-Course/internal/repository/trip"
	positioncreate "github.com/TheJCN/Avito-Course/internal/usecase/position/create"
)

type failingPositionRepository struct {
	delegate *positionrepository.Repository
}

func (r failingPositionRepository) Create(ctx context.Context, value positiondomain.Position) (positiondomain.Position, error) {
	return r.delegate.Create(ctx, value)
}

func (failingPositionRepository) UpdateLastPositionAt(context.Context, uuid.UUID, time.Time) error {
	return errors.New("forced update failure")
}

func TestSavePositionIsAtomic(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	tripRepo := triprepository.New(pool, time.Second)
	positionRepo := positionrepository.New(pool, time.Second)
	manager := transaction.New(pool, time.Second)
	value := createTestTrip(t, pool, tripRepo)

	usecase := positioncreate.New(failingPositionRepository{delegate: positionRepo}, tripRepo, manager)
	_, err := usecase.Execute(context.Background(), value.ID(), 59.93, 30.31, time.Now().UTC())
	if err == nil {
		t.Fatal("forced update failure must be returned")
	}

	var count int
	var lastPositionAt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT (SELECT count(*) FROM trip_positions WHERE trip_id = $1), last_position_at
		FROM trips WHERE id = $1`, value.ID()).Scan(&count, &lastPositionAt); err != nil {
		t.Fatalf("read transaction result: %v", err)
	}
	if count != 0 || lastPositionAt != nil {
		t.Fatalf("transaction was not rolled back: positions=%d last_position_at=%v", count, lastPositionAt)
	}
}

func TestLatePositionDoesNotMoveLastPositionBack(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	tripRepo := triprepository.New(pool, time.Second)
	positionRepo := positionrepository.New(pool, time.Second)
	manager := transaction.New(pool, time.Second)
	value := createTestTrip(t, pool, tripRepo)
	usecase := positioncreate.New(positionRepo, tripRepo, manager)
	fresh := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	late := fresh.Add(-time.Hour)

	if _, err := usecase.Execute(context.Background(), value.ID(), 59.94, 30.32, fresh); err != nil {
		t.Fatalf("save fresh position: %v", err)
	}
	if _, err := usecase.Execute(context.Background(), value.ID(), 59.93, 30.31, late); err != nil {
		t.Fatalf("save late position: %v", err)
	}

	var lastPositionAt time.Time
	if err := pool.QueryRow(context.Background(), "SELECT last_position_at FROM trips WHERE id = $1", value.ID()).Scan(&lastPositionAt); err != nil {
		t.Fatalf("read last_position_at: %v", err)
	}
	if !lastPositionAt.Equal(fresh) {
		t.Fatalf("last_position_at = %s, want %s", lastPositionAt, fresh)
	}
	positions, err := positionRepo.ListByTripID(context.Background(), value.ID())
	if err != nil {
		t.Fatalf("list positions: %v", err)
	}
	if len(positions) != 2 || !positions[0].RecordedAt.Equal(late) || !positions[1].RecordedAt.Equal(fresh) {
		t.Fatalf("positions are not ordered by recorded_at: %+v", positions)
	}
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	return pool
}

func createTestTrip(t *testing.T, pool *pgxpool.Pool, repository *triprepository.Repository) trip.Trip {
	t.Helper()
	value := trip.New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, time.Now().UTC())
	if err := repository.Create(context.Background(), value); err != nil {
		t.Fatalf("create trip: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", value.ID()) })
	return value
}
