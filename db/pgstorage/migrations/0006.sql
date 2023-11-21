-- +migrate Up
ALTER TABLE mt.rht ADD primary key(key, deposit_id);
ALTER TABLE mt.root ADD primary key(deposit_id);
ALTER TABLE sync.token_wrapped ADD primary key(network_id,orig_net,orig_token_addr);
DROP TABLE IF EXISTS mt.rht_temp;

CREATE INDEX IF NOT EXISTS claim_block_id ON sync.claim USING btree (block_id);
CREATE INDEX IF NOT EXISTS deposit_block_id ON sync.deposit USING btree (block_id);
CREATE INDEX IF NOT EXISTS token_wrapped_block_id ON sync.token_wrapped USING btree (block_id);

ALTER TABLE sync.monitored_txs
DROP COLUMN IF EXISTS block_id;
ALTER TABLE sync.monitored_txs
RENAME COLUMN id TO deposit_id;

-- +migrate Down
ALTER TABLE mt.rht DROP CONSTRAINT rht_pkey;
ALTER TABLE mt.root DROP CONSTRAINT root_pkey;
ALTER TABLE sync.token_wrapped DROP CONSTRAINT token_wrapped_pkey;
CREATE TABLE IF NOT EXISTS mt.rht_temp AS (SELECT key, min(value), max(deposit_id) FROM mt.rht GROUP BY key HAVING count(key) > 1);

DROP INDEX IF EXISTS sync.claim_block_id;
DROP INDEX IF EXISTS sync.deposit_block_id;
DROP INDEX IF EXISTS sync.token_wrapped_block_id;

ALTER TABLE sync.monitored_txs
ADD COLUMN block_id BIGINT DEFAULT 0 REFERENCES sync.block (id) ON DELETE CASCADE;
ALTER TABLE sync.monitored_txs
RENAME COLUMN deposit_id TO id;
