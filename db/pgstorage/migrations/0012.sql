-- +migrate Up
DELETE FROM sync.exit_root WHERE block_id = 0; -- This will clean up old and unnecessary values
ALTER TABLE sync.exit_root ADD COLUMN network_id INTEGER NOT NULL DEFAULT 0;

ALTER TABLE IF EXISTS sync.exit_root DROP CONSTRAINT IF EXISTS UC;
ALTER TABLE IF EXISTS sync.exit_root ADD CONSTRAINT UC UNIQUE(block_id, global_exit_root, network_id);

-- +migrate Down
ALTER TABLE sync.exit_root DROP COLUMN network_id;

ALTER TABLE IF EXISTS sync.exit_root DROP CONSTRAINT IF EXISTS UC;
ALTER TABLE IF EXISTS sync.exit_root ADD CONSTRAINT UC UNIQUE(block_id, global_exit_root);
