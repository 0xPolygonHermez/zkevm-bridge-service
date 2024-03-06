-- +migrate Up
ALTER TABLE sync.claim
ALTER COLUMN network_id TYPE BIGINT;

ALTER TABLE sync.deposit
ALTER COLUMN network_id TYPE BIGINT;

ALTER TABLE sync.block
ALTER COLUMN network_id TYPE BIGINT;

ALTER TABLE sync.token_wrapped
ALTER COLUMN network_id TYPE BIGINT;

-- +migrate Down
ALTER TABLE sync.claim
ALTER COLUMN network_id TYPE INTEGER;

ALTER TABLE sync.deposit
ALTER COLUMN network_id TYPE INTEGER;

ALTER TABLE sync.block
ALTER COLUMN network_id TYPE INTEGER;

ALTER TABLE sync.token_wrapped
ALTER COLUMN network_id TYPE INTEGER;