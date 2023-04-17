-- +migrate Down

ALTER TABLE mt.rht DROP COLUMN IF EXISTS root_id;
ALTER TABLE mt.root DROP COLUMN IF EXISTS id;

ALTER TABLE mt.root DROP CONSTRAINT IF EXISTS root_pkey;
ALTER TABLE mt.rht DROP CONSTRAINT IF EXISTS rht_pkey;

ALTER TABLE mt.root ADD CONSTRAINT root_pkey PRIMARY KEY (deposit_cnt, network);
ALTER TABLE mt.rht ADD CONSTRAINT rht_pkey PRIMARY KEY (key);

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

ALTER TABLE mt.root DROP CONSTRAINT IF EXISTS root_pkey;
ALTER TABLE mt.rht DROP CONSTRAINT IF EXISTS rht_pkey;

ALTER TABLE mt.rht DROP COLUMN IF EXISTS root_id;
ALTER TABLE mt.root DROP COLUMN IF EXISTS id;

ALTER TABLE mt.root ADD COLUMN id SERIAL PRIMARY KEY;
ALTER TABLE mt.rht ADD COLUMN root_id BIGINT NOT NULL DEFAULT 1 CONSTRAINT rht_root_id_fkey REFERENCES mt.root (id) ON DELETE CASCADE;
ALTER TABLE mt.rht ALTER COLUMN root_id DROP DEFAULT;

-- +migrate StatementBegin
DO $$
DECLARE
	rt RECORD;
	pkey BYTEA;
	pvalue BYTEA [];
BEGIN
	FOR rt IN SELECT * FROM mt.root 
	LOOP
		IF rt.deposit_cnt > 0 THEN
			rt.deposit_cnt = rt.deposit_cnt - 1;
		END IF;
		pkey = rt.root;
		FOR i IN reverse 31..0
		LOOP
			UPDATE mt.rht SET root_id = rt.id WHERE key = pkey RETURNING value INTO pvalue;
			
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