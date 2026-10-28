package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Metrics struct {
	tripsCreated   metric.Int64Counter
	tripsCompleted metric.Int64Counter
	logger         *slog.Logger
}

type traceHandler struct{ next slog.Handler }

func (h traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h traceHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanContextFromContext(ctx)
	if span.HasTraceID() {
		record.AddAttrs(slog.String("trace_id", span.TraceID().String()), slog.String("span_id", span.SpanID().String()))
	}
	return h.next.Handle(ctx, record)
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{next: h.next.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{next: h.next.WithGroup(name)}
}

func WithTraceContext(logger *slog.Logger) *slog.Logger {
	return slog.New(traceHandler{next: logger.Handler()})
}

func New(ctx context.Context, logger *slog.Logger) (*Metrics, func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	traceExporter, metricExporter, err := exporters(ctx, endpoint, protocol)
	if err != nil {
		return nil, nil, err
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "trip-service"
	}
	resource := resource.NewWithAttributes("",
		attribute.String("service.name", serviceName),
		attribute.String("service.version", "dev"),
		attribute.String("service.namespace", "tripgo"),
		attribute.String("deployment.environment", "local"),
	)
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(resource), sdktrace.WithSampler(sampler()))
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))), sdkmetric.WithResource(resource))
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	meter := otel.Meter("trip-service")
	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		return nil, nil, err
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		return nil, nil, err
	}

	metrics := &Metrics{tripsCreated: created, tripsCompleted: completed, logger: logger}
	shutdown := func(ctx context.Context) error {
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
		return tracerProvider.Shutdown(ctx)
	}
	return metrics, shutdown, nil
}

func exporters(ctx context.Context, endpoint, protocol string) (sdktrace.SpanExporter, sdkmetric.Exporter, error) {
	if protocol == "" || protocol == "grpc" {
		if endpoint == "" {
			endpoint = "localhost:4317"
		}
		traces, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		metrics, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(endpoint), otlpmetricgrpc.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		return traces, metrics, nil
	}
	if protocol == "http/protobuf" {
		if endpoint == "" {
			endpoint = "http://localhost:4318"
		}
		host, err := httpHost(endpoint)
		if err != nil {
			return nil, nil, err
		}
		traces, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(host), otlptracehttp.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		metrics, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpoint(host), otlpmetrichttp.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		return traces, metrics, nil
	}
	return nil, nil, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL: %s", protocol)
}

func httpHost(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid OTEL_EXPORTER_OTLP_ENDPOINT: %s", endpoint)
	}
	return parsed.Host, nil
}

func sampler() sdktrace.Sampler {
	switch os.Getenv("OTEL_TRACES_SAMPLER") {
	case "always_off":
		return sdktrace.NeverSample()
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.AlwaysSample()
	}
}

func (m *Metrics) TripCreated(ctx context.Context) {
	m.tripsCreated.Add(ctx, 1)
	m.logger.InfoContext(ctx, "trip created")
}

func (m *Metrics) TripCompleted(ctx context.Context) {
	m.tripsCompleted.Add(ctx, 1)
	m.logger.InfoContext(ctx, "trip completed")
}
