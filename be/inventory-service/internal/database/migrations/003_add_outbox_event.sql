-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS outbox_event (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate_id VARCHAR(64) NOT NULL,
    event_type   VARCHAR(64) NOT NULL,
    payload      TEXT NOT NULL,
    status       SMALLINT NOT NULL DEFAULT 0,
    retry_count  SMALLINT NOT NULL DEFAULT 0,
    last_error   TEXT,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT idx_outbox_event_aggregate_type
        UNIQUE (aggregate_id, event_type)
);

CREATE INDEX IF NOT EXISTS idx_outbox_event_status_created
    ON outbox_event (status, created_at);

CREATE INDEX IF NOT EXISTS idx_outbox_event_status_updated
    ON outbox_event (status, updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_outbox_event_status_updated;
DROP INDEX IF EXISTS idx_outbox_event_status_created;
DROP TABLE IF EXISTS outbox_event;
-- +goose StatementEnd
