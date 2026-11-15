package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestPanicIsLoggedAndMarksSpanAsError(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(t.Context())
		otel.SetTracerProvider(previousProvider)
	})

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	metricMiddleware, err := Metrics()
	if err != nil {
		t.Fatalf("create metrics middleware: %v", err)
	}

	handler := Tracing()(Logging(logger)(metricMiddleware(Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("want status 500, got %d", response.Code)
	}
	if response.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("unexpected content type: %s", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(logs.String(), `"status":500`) {
		t.Fatalf("request log must contain status 500: %s", logs.String())
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("want one span, got %d", len(spans))
	}
	if spans[0].Status().Code != codes.Error {
		t.Fatalf("span must have error status, got %s", spans[0].Status().Code)
	}
}
