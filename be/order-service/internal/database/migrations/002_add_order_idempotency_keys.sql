-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS order_idempotency_keys (
    user_id    UUID NOT NULL,
    key        VARCHAR(64) NOT NULL,
    order_id   UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (user_id, key)
);

CREATE INDEX IF NOT EXISTS idx_order_idempotency_keys_expires
    ON order_idempotency_keys (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS order_idempotency_keys;
-- +goose StatementEnd
