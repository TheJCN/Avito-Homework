package create

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/TheJCN/Avito-Course/api"
	positiondomain "github.com/TheJCN/Avito-Course/internal/domain/position"
)

var ErrInvalidRequest = errors.New("invalid request")

type Handler struct{ usecase usecase }

func New(usecase usecase) *Handler { return &Handler{usecase: usecase} }

func (h *Handler) Handle(ctx context.Context, request *http.Request, tripID uuid.UUID) (positiondomain.Position, error) {
	data, err := readData(request)
	if err != nil {
		return positiondomain.Position{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	return h.usecase.Execute(ctx, tripID, data.Coordinates.Latitude, data.Coordinates.Longitude, data.RecordedAt)
}

func readData(request *http.Request) (api.PositionData, error) {
	var data api.PositionData
	body, err := io.ReadAll(io.LimitReader(request.Body, 1048577))
	if err != nil || len(body) > 1048576 {
		return data, errors.New("invalid request body")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return data, err
	}
	if len(fields) != 2 || empty(fields["coordinates"]) || empty(fields["recorded_at"]) {
		return data, errors.New("missing or unknown fields")
	}
	var coordinates map[string]json.RawMessage
	if err := json.Unmarshal(fields["coordinates"], &coordinates); err != nil || len(coordinates) != 2 || empty(coordinates["latitude"]) || empty(coordinates["longitude"]) {
		return data, errors.New("invalid coordinates")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return data, err
	}
	return data, nil
}

func empty(value json.RawMessage) bool {
	return len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}
