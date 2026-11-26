-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS message TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders
    DROP COLUMN IF EXISTS message;
-- +goose StatementEnd
