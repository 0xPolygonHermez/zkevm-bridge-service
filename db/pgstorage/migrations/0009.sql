-- +migrate Down

ALTER TABLE sync.deposit DROP COLUMN IF EXISTS ready_time;

-- +migrate Up

ALTER TABLE sync.deposit ADD COLUMN ready_time TIMESTAMP WITH TIME ZONE;