-- +migrate Up
DELETE FROM sync.exit_root WHERE block_id = 0; -- This will clean up old and unnecessary values
ALTER TABLE sync.exit_root ADD COLUMN network_id INTEGER NOT NULL DEFAULT 0;

-- +migrate Down
ALTER TABLE sync.exit_root DROP COLUMN network_id;
