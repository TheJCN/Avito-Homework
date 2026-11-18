package positions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/internal/model"
)

type fakeTicker struct{ ticks chan time.Time }

func newFakeTicker() *fakeTicker          { return &fakeTicker{ticks: make(chan time.Time)} }
func (t *fakeTicker) C() <-chan time.Time { return t.ticks }
func (t *fakeTicker) Stop()               {}
func (t *fakeTicker) Tick()               { t.ticks <- time.Now() }

type repositoryStub struct {
	values []model.StaleTrip
	err    error
	calls  chan struct{}
}

func (r *repositoryStub) FindStale(context.Context, time.Duration) ([]model.StaleTrip, error) {
	if r.calls != nil {
		r.calls <- struct{}{}
	}
	return r.values, r.err
}

type gatewayFunc func(context.Context, uuid.UUID, uuid.UUID) error

func (f gatewayFunc) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	return f(ctx, driverID, tripID)
}

func newTestPool(t *testing.T, config Config, repository repository, gateway pushGateway, ticker Ticker) *Pool {
	t.Helper()
	pool, err := New(config, repository, gateway, ticker, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return pool
}

func TestEnqueueDeduplicatesAndDropsWhenFull(t *testing.T) {
	pool := newTestPool(t, Config{StaleAfter: time.Second, RequestTimeout: time.Second, Workers: 1, QueueSize: 1}, &repositoryStub{}, gatewayFunc(func(context.Context, uuid.UUID, uuid.UUID) error { return nil }), newFakeTicker())
	first := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	second := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}

	if result := pool.Enqueue(first); result != EnqueueQueued {
		t.Fatalf("first enqueue = %s", result)
	}
	if result := pool.Enqueue(first); result != EnqueueDuplicate {
		t.Fatalf("duplicate enqueue = %s", result)
	}
	if result := pool.Enqueue(second); result != EnqueueDropped {
		t.Fatalf("full queue enqueue = %s", result)
	}
}

func TestCancelAndRequeueSkipsOldGeneration(t *testing.T) {
	ticker := newFakeTicker()
	calls := make(chan uuid.UUID, 2)
	pool := newTestPool(t, Config{StaleAfter: time.Second, RequestTimeout: time.Second, Workers: 1, QueueSize: 2}, &repositoryStub{}, gatewayFunc(func(_ context.Context, _, tripID uuid.UUID) error {
		calls <- tripID
		return nil
	}), ticker)
	value := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}

	if pool.Enqueue(value) != EnqueueQueued || !pool.Cancel(value.ID) || pool.Enqueue(value) != EnqueueQueued {
		t.Fatal("cancel and requeue setup failed")
	}
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	select {
	case got := <-calls:
		if got != value.ID {
			t.Fatalf("trip ID = %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("new generation was not processed")
	}
	select {
	case <-calls:
		t.Fatal("old generation must be skipped")
	case <-time.After(20 * time.Millisecond):
	}
	drainPool(t, pool)
}

func TestScannerUsesManagedTicks(t *testing.T) {
	ticker := newFakeTicker()
	value := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	repository := &repositoryStub{values: []model.StaleTrip{value}, calls: make(chan struct{}, 1)}
	requests := make(chan uuid.UUID, 1)
	pool := newTestPool(t, Config{StaleAfter: time.Minute, RequestTimeout: time.Second, Workers: 1, QueueSize: 1}, repository, gatewayFunc(func(_ context.Context, _, tripID uuid.UUID) error {
		requests <- tripID
		return nil
	}), ticker)
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	ticker.Tick()
	select {
	case <-repository.calls:
	case <-time.After(time.Second):
		t.Fatal("scanner did not query repository")
	}
	select {
	case got := <-requests:
		if got != value.ID {
			t.Fatalf("trip ID = %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not send push request")
	}
	drainPool(t, pool)
}

func TestWorkerCountIsFixed(t *testing.T) {
	const workers = 3
	started := make(chan struct{}, workers+1)
	release := make(chan struct{})
	var active atomic.Int32
	var maximum atomic.Int32
	gateway := gatewayFunc(func(context.Context, uuid.UUID, uuid.UUID) error {
		current := active.Add(1)
		for current > maximum.Load() && !maximum.CompareAndSwap(maximum.Load(), current) {
		}
		started <- struct{}{}
		<-release
		active.Add(-1)
		return errors.New("push failed")
	})
	pool := newTestPool(t, Config{StaleAfter: time.Second, RequestTimeout: time.Second, Workers: workers, QueueSize: workers + 1}, &repositoryStub{}, gateway, newFakeTicker())
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	for range workers + 1 {
		if result := pool.Enqueue(model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}); result != EnqueueQueued {
			t.Fatalf("enqueue = %s", result)
		}
	}
	for range workers {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("not all workers started")
		}
	}
	select {
	case <-started:
		t.Fatal("more than configured workers ran concurrently")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	drainPool(t, pool)
	if maximum.Load() != workers {
		t.Fatalf("maximum active workers = %d, want %d", maximum.Load(), workers)
	}
}

func TestPushErrorDoesNotStopWorker(t *testing.T) {
	requests := make(chan uuid.UUID, 2)
	var count atomic.Int32
	pool := newTestPool(t, Config{StaleAfter: time.Second, RequestTimeout: time.Second, Workers: 1, QueueSize: 2}, &repositoryStub{}, gatewayFunc(func(_ context.Context, _, tripID uuid.UUID) error {
		requests <- tripID
		if count.Add(1) == 1 {
			return errors.New("push failed")
		}
		return nil
	}), newFakeTicker())
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	first := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	second := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	pool.Enqueue(first)
	pool.Enqueue(second)
	for _, want := range []uuid.UUID{first.ID, second.ID} {
		select {
		case got := <-requests:
			if got != want {
				t.Fatalf("request trip = %s, want %s", got, want)
			}
		case <-time.After(time.Second):
			t.Fatal("worker stopped after push error")
		}
	}
	drainPool(t, pool)
}

func TestDrainTimeoutCancelsActiveRequest(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	pool := newTestPool(t, Config{StaleAfter: time.Second, RequestTimeout: time.Hour, Workers: 1, QueueSize: 1}, &repositoryStub{}, gatewayFunc(func(ctx context.Context, _, _ uuid.UUID) error {
		close(started)
		<-ctx.Done()
		close(canceled)
		return ctx.Err()
	}), newFakeTicker())
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	pool.Enqueue(model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := pool.Drain(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Drain() error = %v, want context canceled", err)
	}
	select {
	case <-canceled:
	default:
		t.Fatal("active push request was not canceled")
	}
}

func drainPool(t *testing.T, pool *Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pool.Drain(ctx); err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
}
