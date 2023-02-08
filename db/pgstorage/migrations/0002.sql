-- +migrate Down

ALTER TABLE mt.rht DROP COLUMN IF EXISTS deposit_id;
ALTER TABLE mt.root DROP COLUMN IF EXISTS deposit_id;
ALTER TABLE sync.deposit DROP COLUMN IF EXISTS id;

ALTER TABLE sync.deposit ADD CONSTRAINT deposit_pkey PRIMARY KEY (network_id, deposit_cnt);

DROP INDEX IF EXISTS mt.root_network_idx;
DROP INDEX IF EXISTS mt.deposit_idx;
DROP INDEX IF EXISTS sync.block_idx;
DROP INDEX IF EXISTS mt.root_idx;
DROP INDEX IF EXISTS sync.exit_roots_idx;

ALTER SCHEMA mt RENAME TO mtv2;
ALTER SCHEMA sync RENAME TO syncv2;

-- +migrate Up
DROP SCHEMA IF EXISTS sync CASCADE;
DROP SCHEMA IF EXISTS mt CASCADE;

ALTER SCHEMA mtv2 RENAME TO mt;
ALTER SCHEMA syncv2 RENAME TO sync;

ALTER TABLE sync.deposit DROP CONSTRAINT IF EXISTS deposit_pkey;
ALTER TABLE sync.deposit ADD COLUMN id SERIAL PRIMARY KEY;

ALTER TABLE mt.root ADD COLUMN deposit_id BIGINT NOT NULL DEFAULT 1 CONSTRAINT root_deposit_id_fkey REFERENCES sync.deposit (id) ON DELETE CASCADE;
ALTER TABLE mt.root ALTER COLUMN deposit_id DROP DEFAULT;
UPDATE mt.root AS r SET deposit_id = d.id FROM sync.deposit AS d WHERE d.deposit_cnt = r.deposit_cnt AND d.network_id = r.network;

ALTER TABLE mt.rht ADD COLUMN deposit_id BIGINT NOT NULL DEFAULT 1 CONSTRAINT rht_deposit_id_fkey REFERENCES sync.deposit (id) ON DELETE CASCADE;
ALTER TABLE mt.rht ALTER COLUMN deposit_id DROP DEFAULT;

-- +migrate StatementBegin
DO $$
DECLARE
	rt RECORD;
	pkey BYTEA;
	pvalue BYTEA [];
BEGIN
	FOR rt IN SELECT * FROM mt.root 
	LOOP
		pkey = rt.root;
		FOR i IN reverse 31..0
		LOOP
			UPDATE mt.rht SET deposit_id = rt.deposit_id WHERE key = pkey RETURNING value INTO pvalue;
			
			IF rt.deposit_cnt & (1 << i) > 0 THEN
				pkey = pvalue[2];
			ELSE
				pkey = pvalue[1];
			END IF;
		END LOOP;
	END LOOP;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- Create indexes
CREATE INDEX IF NOT EXISTS root_network_idx ON mt.root(root, network);
CREATE INDEX IF NOT EXISTS deposit_idx ON mt.root(deposit_cnt);
CREATE INDEX IF NOT EXISTS block_idx ON sync.exit_root(block_id);
CREATE INDEX IF NOT EXISTS root_idx ON mt.root(root);
CREATE INDEX IF NOT EXISTS exit_roots_idx ON sync.exit_root(exit_roots);