package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheJCN/Avito-Course/internal/config"
	pushgateway "github.com/TheJCN/Avito-Course/internal/gateway/push"
	"github.com/TheJCN/Avito-Course/internal/http/health"
	"github.com/TheJCN/Avito-Course/internal/http/middleware"
	positioncreatehttp "github.com/TheJCN/Avito-Course/internal/http/position/create"
	positionlisthttp "github.com/TheJCN/Avito-Course/internal/http/position/list"
	"github.com/TheJCN/Avito-Course/internal/http/router"
	createhttp "github.com/TheJCN/Avito-Course/internal/http/trip/create"
	finishhttp "github.com/TheJCN/Avito-Course/internal/http/trip/finish"
	gethttp "github.com/TheJCN/Avito-Course/internal/http/trip/get"
	"github.com/TheJCN/Avito-Course/internal/infra/postgres/transaction"
	"github.com/TheJCN/Avito-Course/internal/infra/telemetry"
	positionrepository "github.com/TheJCN/Avito-Course/internal/repository/position"
	triprepository "github.com/TheJCN/Avito-Course/internal/repository/trip"
	positioncreateusecase "github.com/TheJCN/Avito-Course/internal/usecase/position/create"
	positionlistusecase "github.com/TheJCN/Avito-Course/internal/usecase/position/list"
	createusecase "github.com/TheJCN/Avito-Course/internal/usecase/trip/create"
	finishusecase "github.com/TheJCN/Avito-Course/internal/usecase/trip/finish"
	getusecase "github.com/TheJCN/Avito-Course/internal/usecase/trip/get"
	positionworkers "github.com/TheJCN/Avito-Course/internal/worker/positions"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Read()
	if err != nil {
		return err
	}

	logger, err := setupLogger(cfg.LogLevel)
	if err != nil {
		return err
	}
	metrics, shutdownTelemetry, err := telemetry.New(context.Background(), logger)
	if err != nil {
		return fmt.Errorf("start telemetry: %w", err)
	}
	var shutdownDeadline time.Time
	defer func() {
		if shutdownDeadline.IsZero() {
			shutdownDeadline = time.Now().Add(cfg.ShutdownTimeout)
		}
		shutdownCtx, cancel := context.WithDeadline(context.Background(), shutdownDeadline)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			logger.Error("stop telemetry", "error", err)
		}
	}()
	server, pool, workers, pushClient, err := newServer(context.Background(), cfg, logger, metrics)
	if err != nil {
		return err
	}
	defer pool.Close()
	defer pushClient.CloseIdleConnections()
	if err := workers.Start(context.Background()); err != nil {
		return fmt.Errorf("start position workers: %w", err)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	stopShutdown := context.AfterFunc(signalCtx, func() {
		defer close(done)
		shutdownDeadline = time.Now().Add(cfg.ShutdownTimeout)
		shutdownCtx, cancel := context.WithDeadline(context.Background(), shutdownDeadline)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown failed", "error", err)
			if err := server.Close(); err != nil {
				logger.Error("close server failed", "error", err)
			}
		}
	})

	logger.Info("listening", "address", cfg.HTTPAddress)
	logger.Info("service started")
	err = server.ListenAndServe()
	if !stopShutdown() {
		<-done
	}
	if shutdownDeadline.IsZero() {
		shutdownDeadline = time.Now().Add(cfg.ShutdownTimeout)
	}
	shutdownCtx, cancel := context.WithDeadline(context.Background(), shutdownDeadline)
	defer cancel()
	if err := workers.Drain(shutdownCtx); err != nil {
		logger.Error("drain position workers", "error", err)
	}
	logger.Info("service stopped")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}

	return nil
}

func setupLogger(logLevel string) (*slog.Logger, error) {
	level := new(slog.Level)
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		return nil, fmt.Errorf("invalid LOG_LEVEL: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).With("service", "trip-service")
	return telemetry.WithTraceContext(logger), nil
}

func newServer(ctx context.Context, cfg config.Config, logger *slog.Logger, metrics *telemetry.Metrics) (*http.Server, *pgxpool.Pool, *positionworkers.Pool, *pushgateway.Client, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	poolConfig.MaxConns = cfg.DatabaseMaxConns
	poolConfig.MinConns = cfg.DatabaseMinConns
	poolConfig.MaxConnLifetime = cfg.DatabaseMaxLifetime
	poolConfig.ConnConfig.ConnectTimeout = cfg.DatabaseConnectTimeout
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()

	connectCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("open database: %w", err)
	}
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, nil, nil, nil, fmt.Errorf("ping database: %w", err)
	}
	if err := otelpgx.RecordStats(pool); err != nil {
		pool.Close()
		return nil, nil, nil, nil, fmt.Errorf("start database metrics: %w", err)
	}

	repository := triprepository.New(pool, cfg.DatabaseQueryTimeout)
	positionsRepository := positionrepository.New(pool, cfg.DatabaseQueryTimeout)
	transactionManager := transaction.New(pool, cfg.DatabaseQueryTimeout)
	pushClient, err := pushgateway.New(cfg.PushHTTPURL, cfg.PushRequestTimeout)
	if err != nil {
		pool.Close()
		return nil, nil, nil, nil, fmt.Errorf("create push client: %w", err)
	}
	workers, err := positionworkers.New(positionworkers.Config{
		StaleAfter:     cfg.PositionStaleAfter,
		RequestTimeout: cfg.PushRequestTimeout,
		Workers:        int(cfg.PositionWorkers),
		QueueSize:      int(cfg.PositionQueueSize),
	}, repository, pushClient, positionworkers.NewTicker(cfg.PositionCheckInterval), logger, metrics)
	if err != nil {
		pool.Close()
		pushClient.CloseIdleConnections()
		return nil, nil, nil, nil, fmt.Errorf("create position workers: %w", err)
	}
	create := createusecase.New(repository, transactionManager, metrics)
	get := getusecase.New(repository)
	finish := finishusecase.New(repository, transactionManager, metrics)
	positionCreate := positioncreateusecase.NewWithObservers(positionsRepository, repository, transactionManager, workers, metrics)
	positionList := positionlistusecase.New(positionsRepository, repository)
	metricMiddleware, err := middleware.Metrics()
	if err != nil {
		pool.Close()
		pushClient.CloseIdleConnections()
		return nil, nil, nil, nil, fmt.Errorf("create HTTP metrics: %w", err)
	}
	handler := router.NewWithPositions(
		logger,
		createhttp.New(create),
		gethttp.New(get),
		finishhttp.New(finish),
		positioncreatehttp.New(positionCreate),
		positionlisthttp.New(positionList),
		health.New(pool, cfg.DatabaseQueryTimeout),
		middleware.Tracing(),
		middleware.Logging(logger),
		metricMiddleware,
		middleware.Recover(logger),
	)

	return &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           handler,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}, pool, workers, pushClient, nil
}
