package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic recovered", "error", recovered, "stack", string(debug.Stack()))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func Tracing() func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware(
		"http.server",
		otelhttp.WithMeterProvider(noop.NewMeterProvider()),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + route(r)
		}),
	)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			writer := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(writer, r)
			span := trace.SpanContextFromContext(r.Context())
			attributes := []any{
				"method", r.Method,
				"route", route(r),
				"status", writer.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
			}
			if span.HasTraceID() {
				attributes = append(attributes, "trace_id", span.TraceID().String(), "span_id", span.SpanID().String())
			}
			if writer.status >= http.StatusInternalServerError {
				logger.ErrorContext(r.Context(), "request handled", attributes...)
				return
			}
			logger.InfoContext(r.Context(), "request handled", attributes...)
		})
	}
}

func Metrics() (func(http.Handler) http.Handler, error) {
	meter := otel.Meter("trip-service")
	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	active, err := meter.Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			writer := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			attributes := metric.WithAttributes(attribute.String("http.request.method", r.Method), attribute.String("http.route", route(r)))
			active.Add(r.Context(), 1, attributes)
			defer active.Add(r.Context(), -1, attributes)
			next.ServeHTTP(writer, r)
			duration.Record(r.Context(), time.Since(startedAt).Seconds(), metric.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route(r)),
				attribute.String("http.response.status_code", strconv.Itoa(writer.status)),
			))
		})
	}, nil
}

func route(r *http.Request) string {
	pattern := chi.RouteContext(r.Context()).RoutePattern()
	if pattern == "" {
		return r.URL.Path
	}
	return pattern
}
