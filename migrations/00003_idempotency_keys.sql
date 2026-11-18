-- +goose Up
CREATE TABLE idempotency_keys (
    key UUID PRIMARY KEY,
    request_hash BYTEA NOT NULL,
    trip_id UUID REFERENCES trips(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE idempotency_keys;
