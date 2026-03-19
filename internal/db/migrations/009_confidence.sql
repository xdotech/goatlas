-- +goose Up
ALTER TABLE function_calls ADD COLUMN IF NOT EXISTS confidence REAL DEFAULT 0.5;
ALTER TABLE interface_impls ADD COLUMN IF NOT EXISTS confidence REAL DEFAULT 0.85;

-- +goose Down
ALTER TABLE function_calls DROP COLUMN IF EXISTS confidence;
ALTER TABLE interface_impls DROP COLUMN IF EXISTS confidence;
