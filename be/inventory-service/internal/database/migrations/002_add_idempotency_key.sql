-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS idempotency_key (
    token      VARCHAR(64) PRIMARY KEY NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires_at
    ON idempotency_key (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_idempotency_expires_at;
DROP TABLE IF EXISTS idempotency_key;
-- +goose StatementEnd
