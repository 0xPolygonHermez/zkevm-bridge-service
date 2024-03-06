-- +migrate Up
ALTER TABLE sync.claim
ALTER COLUMN network_id TYPE BIGINT;

-- +migrate Down
ALTER TABLE sync.claim
ALTER COLUMN network_id TYPE INTEGER;