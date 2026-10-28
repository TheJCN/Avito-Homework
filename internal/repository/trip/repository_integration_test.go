//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheJCN/Avito-Course/internal/domain/trip"
	transaction "github.com/TheJCN/Avito-Course/internal/infra/postgres/transaction"
	repository "github.com/TheJCN/Avito-Course/internal/repository/trip"
)

func TestRepositoryWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	repo := repository.New(pool, time.Second)
	value := trip.New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, time.Now().UTC())
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", value.ID())
	})

	if err := repo.Create(context.Background(), value); err != nil {
		t.Fatalf("create trip: %v", err)
	}
	if err := repo.CreateStatusHistory(context.Background(), trip.StatusHistory{TripID: value.ID(), ToStatus: trip.StatusActive}); err != nil {
		t.Fatalf("create history: %v", err)
	}

	found, err := repo.FindByID(context.Background(), value.ID())
	if err != nil {
		t.Fatalf("find created trip: %v", err)
	}
	if found.ID() != value.ID() || found.Status() != trip.StatusActive {
		t.Fatal("found trip must be the created active trip")
	}

	if err := found.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("finish domain trip: %v", err)
	}
	if err := repo.Update(context.Background(), found); err != nil {
		t.Fatalf("update trip: %v", err)
	}

	finished, err := repo.FindByID(context.Background(), value.ID())
	if err != nil {
		t.Fatalf("find finished trip: %v", err)
	}
	if finished.Status() != trip.StatusCompleted || finished.FinishedAt() == nil {
		t.Fatal("database must keep completed trip")
	}
}

func TestRepositoryWithPostgresRollsBackTripAndHistory(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	repo := repository.New(pool, time.Second)
	manager := transaction.New(pool, time.Second)
	value := trip.New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, time.Now().UTC())

	err = manager.Do(context.Background(), func(ctx context.Context) error {
		if err := repo.Create(ctx, value); err != nil {
			return err
		}
		return repo.CreateStatusHistory(ctx, trip.StatusHistory{TripID: value.ID(), ToStatus: "wrong_status"})
	})
	if err == nil {
		t.Fatal("wrong history status must return error")
	}

	_, err = repo.FindByID(context.Background(), value.ID())
	if !errors.Is(err, trip.ErrNotFound) {
		t.Fatalf("trip must be rolled back, got %v", err)
	}
}
