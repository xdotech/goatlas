-- +goose Up
ALTER TABLE symbols ADD COLUMN IF NOT EXISTS embedded_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE symbols DROP COLUMN IF EXISTS embedded_at;
