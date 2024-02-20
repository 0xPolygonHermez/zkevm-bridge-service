-- +migrate Up

ALTER TABLE sync.deposit
ADD COLUMN IF NOT EXISTS origin_rollup_id BIGINT DEFAULT 0;

-- +migrate Down

ALTER TABLE sync.deposit
DROP COLUMN IF EXISTS origin_rollup_id;