package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	ErrTransport = errors.New("push transport error")
	ErrClient    = errors.New("push service client error")
	ErrServer    = errors.New("push service server error")
)

type HTTPStatusError struct {
	StatusCode int
	kind       error
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("push service returned HTTP %d", e.StatusCode)
}

func (e *HTTPStatusError) Unwrap() error { return e.kind }

type Client struct {
	url    string
	client *http.Client
}

func New(baseURL string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("push service URL is empty")
	}
	if timeout <= 0 {
		return nil, errors.New("push request timeout must be positive")
	}
	return &Client{
		url: baseURL + "/api/v1/push",
		client: &http.Client{
			Timeout:   timeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport.(*http.Transport).Clone()),
		},
	}, nil
}

func (c *Client) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	payload := struct {
		RequestID   string            `json:"request_id"`
		RecipientID uuid.UUID         `json:"recipient_id"`
		Kind        string            `json:"kind"`
		Data        map[string]string `json:"data"`
	}{
		RequestID:   uuid.NewString(),
		RecipientID: driverID,
		Kind:        "REQUEST_POSITION",
		Data:        map[string]string{"trip_id": tripID.String()},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode push request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTransport, err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))

	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		return nil
	case response.StatusCode >= 400 && response.StatusCode < 500:
		return &HTTPStatusError{StatusCode: response.StatusCode, kind: ErrClient}
	case response.StatusCode >= 500:
		return &HTTPStatusError{StatusCode: response.StatusCode, kind: ErrServer}
	default:
		return &HTTPStatusError{StatusCode: response.StatusCode, kind: ErrServer}
	}
}

func (c *Client) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}
