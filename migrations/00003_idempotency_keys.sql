-- +goose Up
CREATE TABLE idempotency_keys (
    key UUID PRIMARY KEY,
    request_hash BYTEA NOT NULL,
    trip_id UUID REFERENCES trips(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys (expires_at);

-- +goose Down
DROP TABLE idempotency_keys;
