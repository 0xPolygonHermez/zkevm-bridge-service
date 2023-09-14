-- +migrate Up
ALTER TABLE mt.rht ADD primary key(key, deposit_id);
ALTER TABLE mt.root ADD primary key(deposit_id);
ALTER TABLE sync.token_wrapped ADD primary key(block_id);
DROP TABLE IF EXISTS mt.rht_temp;

-- +migrate Down
ALTER TABLE mt.rht DROP CONSTRAINT rht_pkey;
ALTER TABLE mt.root DROP CONSTRAINT root_pkey;
ALTER TABLE sync.token_wrapped DROP CONSTRAINT token_wrapped_pkey;
CREATE TABLE IF NOT EXISTS mt.rht_temp AS (SELECT key, min(value), max(deposit_id) FROM mt.rht GROUP BY key HAVING count(key) > 1);
