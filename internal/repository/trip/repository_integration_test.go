//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"strings"
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

func TestRepositoryFindStale(t *testing.T) {
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
	now := time.Now().UTC()
	oldest := trip.New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, now.Add(-3*time.Hour))
	stalePosition := trip.New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, now.Add(-4*time.Hour))
	fresh := trip.New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, now.Add(-4*time.Hour))
	completed := trip.New(uuid.New(), uuid.New(), 10, 20, 30, 40, 500, now.Add(-5*time.Hour))
	values := []trip.Trip{oldest, stalePosition, fresh, completed}
	for _, value := range values {
		if err := repo.Create(context.Background(), value); err != nil {
			t.Fatalf("create trip: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, value := range values {
			_, _ = pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", value.ID())
		}
	})

	if _, err := pool.Exec(context.Background(), "UPDATE trips SET last_position_at = $2 WHERE id = $1", stalePosition.ID(), now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("set stale position time: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE trips SET last_position_at = $2 WHERE id = $1", fresh.ID(), now); err != nil {
		t.Fatalf("set fresh position time: %v", err)
	}
	if err := completed.Finish(now.Add(-time.Hour)); err != nil {
		t.Fatalf("finish trip: %v", err)
	}
	if err := repo.Update(context.Background(), completed); err != nil {
		t.Fatalf("update completed trip: %v", err)
	}

	result, err := repo.FindStale(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("FindStale() error = %v", err)
	}
	var matched []uuid.UUID
	for _, value := range result {
		if value.ID == oldest.ID() || value.ID == stalePosition.ID() || value.ID == fresh.ID() || value.ID == completed.ID() {
			matched = append(matched, value.ID)
		}
	}
	if len(matched) != 2 || matched[0] != oldest.ID() || matched[1] != stalePosition.ID() {
		t.Fatalf("FindStale() matched = %v, want [%s %s]", matched, oldest.ID(), stalePosition.ID())
	}
}

func TestRepositoryFindStaleUsesIndex(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	for _, query := range []string{
		"CREATE TEMP TABLE trips (LIKE public.trips INCLUDING ALL) ON COMMIT DROP",
		`INSERT INTO trips (id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at)
		SELECT gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 0, 0, 0, 0, 0,
			CASE WHEN n <= 30000 THEN 'active' ELSE 'completed' END,
			CASE WHEN n <= 200 THEN now() - interval '1 hour' ELSE now() END,
			CASE WHEN n > 30000 THEN now() END
		FROM generate_series(1, 50000) AS n`,
		"ANALYZE trips",
	} {
		if _, err := tx.Exec(context.Background(), query); err != nil {
			t.Fatalf("prepare planner fixture: %v", err)
		}
	}
	rows, err := tx.Query(context.Background(), `
		EXPLAIN (ANALYZE, COSTS OFF)
		SELECT id, driver_id
		FROM trips
		WHERE status = 'active'
		  AND COALESCE(last_position_at, started_at) < now() - make_interval(secs => $1)
		ORDER BY COALESCE(last_position_at, started_at), id`, float64(60))
	if err != nil {
		t.Fatalf("explain stale query: %v", err)
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("read plan: %v", err)
		}
		lines = append(lines, line)
	}
	plan := strings.Join(lines, "\n")
	if !strings.Contains(plan, "COALESCE(last_position_at, started_at) <") || !strings.Contains(plan, "Index Cond:") {
		t.Fatalf("stale query does not use expected index:\n%s", plan)
	}
	t.Logf("stale query plan:\n%s", plan)
}
