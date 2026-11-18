package positions

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/TheJCN/Avito-Course/internal/model"
)

var ErrStopped = errors.New("position worker pool is stopped")

type EnqueueResult string

const (
	EnqueueQueued    EnqueueResult = "queued"
	EnqueueDuplicate EnqueueResult = "duplicate"
	EnqueueDropped   EnqueueResult = "dropped"
)

type Config struct {
	StaleAfter     time.Duration
	RequestTimeout time.Duration
	Workers        int
	QueueSize      int
}

type task struct {
	trip       model.StaleTrip
	generation uint64
	link       trace.Link
}

type Pool struct {
	config     Config
	repository repository
	push       pushGateway
	ticker     Ticker
	logger     *slog.Logger
	metrics    metrics
	tracer     trace.Tracer

	mu          sync.Mutex
	pending     map[uuid.UUID]uint64
	next        uint64
	queue       chan task
	started     bool
	closing     bool
	scannerStop context.CancelFunc
	workersStop context.CancelFunc
	active      atomic.Int64

	scannerWG   sync.WaitGroup
	workersWG   sync.WaitGroup
	closeOnce   sync.Once
	scannerOnce sync.Once
}

func New(config Config, repository repository, push pushGateway, ticker Ticker, logger *slog.Logger, metricValues ...metrics) (*Pool, error) {
	if config.StaleAfter <= 0 || config.RequestTimeout <= 0 || config.Workers < 1 || config.QueueSize < 1 {
		return nil, errors.New("invalid position worker configuration")
	}
	if repository == nil || push == nil || ticker == nil || logger == nil {
		return nil, errors.New("position worker dependency is nil")
	}
	var metric metrics
	if len(metricValues) > 0 {
		metric = metricValues[0]
	}
	return &Pool{
		config: config, repository: repository, push: push, ticker: ticker, logger: logger,
		metrics: metric, tracer: otel.Tracer("trip-service/position-workers"),
		pending: make(map[uuid.UUID]uint64), queue: make(chan task, config.QueueSize),
	}, nil
}

func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.started || p.closing {
		p.mu.Unlock()
		return ErrStopped
	}
	p.started = true
	scannerCtx, cancel := context.WithCancel(ctx)
	p.scannerStop = cancel
	workersCtx, workersCancel := context.WithCancel(context.WithoutCancel(ctx))
	p.workersStop = workersCancel
	p.mu.Unlock()

	for range p.config.Workers {
		p.workersWG.Add(1)
		go p.worker(workersCtx)
	}
	p.scannerWG.Add(1)
	go p.scan(scannerCtx)
	p.logger.Info("position scanner and workers started", "workers", p.config.Workers, "queue_size", p.config.QueueSize)
	return nil
}

func (p *Pool) Enqueue(value model.StaleTrip) EnqueueResult {
	return p.enqueue(context.Background(), value)
}

func (p *Pool) enqueue(ctx context.Context, value model.StaleTrip) EnqueueResult {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closing {
		return EnqueueDropped
	}
	if _, exists := p.pending[value.ID]; exists {
		return EnqueueDuplicate
	}
	p.next++
	current := task{trip: value, generation: p.next}
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		current.link = trace.Link{SpanContext: spanContext}
	}
	p.pending[value.ID] = current.generation
	select {
	case p.queue <- current:
		p.recordQueueSize(ctx)
		return EnqueueQueued
	default:
		delete(p.pending, value.ID)
		p.logger.Warn("position request queue is full", "trip_id", value.ID)
		if p.metrics != nil {
			p.metrics.PositionRequestDropped(ctx)
		}
		return EnqueueDropped
	}
}

func (p *Pool) Cancel(tripID uuid.UUID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.pending[tripID]; !exists {
		return false
	}
	delete(p.pending, tripID)
	return true
}

func (p *Pool) StopScanner() {
	p.scannerOnce.Do(func() {
		p.mu.Lock()
		cancel := p.scannerStop
		p.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		p.ticker.Stop()
		p.scannerWG.Wait()
		p.logger.Info("position scanner stopped")
	})
}

func (p *Pool) Drain(ctx context.Context) error {
	p.StopScanner()
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closing = true
		close(p.queue)
		p.mu.Unlock()
	})
	done := make(chan struct{})
	go func() {
		p.workersWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		p.stopWorkers()
		p.logger.Info("position workers stopped")
		return nil
	case <-ctx.Done():
		p.stopWorkers()
		<-done
		p.logger.Warn("position workers stopped after drain timeout", "error", ctx.Err())
		return ctx.Err()
	}
}

func (p *Pool) scan(ctx context.Context) {
	defer p.scannerWG.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.ticker.C():
			scanCtx, span := p.tracer.Start(ctx, "positions.scan")
			values, err := p.repository.FindStale(scanCtx, p.config.StaleAfter)
			if err != nil {
				p.logger.ErrorContext(scanCtx, "find stale trips", "error", err)
				span.RecordError(err)
				span.End()
				continue
			}
			for _, value := range values {
				result := p.enqueue(scanCtx, value)
				if p.metrics != nil {
					p.metrics.StaleCheck(scanCtx, string(result))
				}
			}
			span.End()
		}
	}
}

func (p *Pool) worker(ctx context.Context) {
	defer p.workersWG.Done()
	for {
		var current task
		var ok bool
		select {
		case <-ctx.Done():
			return
		case current, ok = <-p.queue:
			if !ok {
				return
			}
		}
		p.recordQueueSize(ctx)
		if !p.isCurrent(current) {
			continue
		}
		spanOptions := make([]trace.SpanStartOption, 0, 1)
		if current.link.SpanContext.IsValid() {
			spanOptions = append(spanOptions, trace.WithLinks(current.link))
		}
		workerCtx, span := p.tracer.Start(ctx, "positions.worker", spanOptions...)
		active := p.active.Add(1)
		if p.metrics != nil {
			p.metrics.PositionWorkersActive(workerCtx, active)
		}
		startedAt := time.Now()
		requestCtx, cancel := context.WithTimeout(workerCtx, p.config.RequestTimeout)
		err := p.push.RequestPosition(requestCtx, current.trip.DriverID, current.trip.ID)
		cancel()
		result := "success"
		if err != nil {
			result = "error"
			span.RecordError(err)
			p.logger.WarnContext(workerCtx, "push service request failed", "trip_id", current.trip.ID, "error", err)
		}
		if p.metrics != nil {
			p.metrics.PositionRequest(workerCtx, result, time.Since(startedAt))
			p.metrics.PositionWorkersActive(workerCtx, p.active.Add(-1))
		} else {
			p.active.Add(-1)
		}
		p.complete(current)
		span.End()
	}
}

func (p *Pool) recordQueueSize(ctx context.Context) {
	if p.metrics != nil {
		p.metrics.PositionQueueSize(ctx, len(p.queue))
	}
}

func (p *Pool) stopWorkers() {
	p.mu.Lock()
	cancel := p.workersStop
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (p *Pool) isCurrent(current task) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pending[current.trip.ID] == current.generation
}

func (p *Pool) complete(current task) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pending[current.trip.ID] == current.generation {
		delete(p.pending, current.trip.ID)
	}
}
