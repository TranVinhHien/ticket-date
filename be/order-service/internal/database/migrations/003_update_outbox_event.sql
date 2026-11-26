-- +goose Up
-- +goose StatementBegin
ALTER TABLE outbox_event
    ADD COLUMN IF NOT EXISTS retry_count SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_error TEXT,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_outbox_event_status_updated
    ON outbox_event (status, updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_outbox_event_status_updated;
ALTER TABLE outbox_event
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS retry_count;
-- +goose StatementEnd
