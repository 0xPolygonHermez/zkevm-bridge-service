-- +migrate Down

ALTER TABLE mt.rht DROP COLUMN IF EXISTS deposit_id;
ALTER TABLE mt.root DROP COLUMN IF EXISTS deposit_id;
ALTER TABLE sync.deposit DROP COLUMN IF EXISTS id;

ALTER TABLE sync.deposit ADD CONSTRAINT deposit_pkey PRIMARY KEY (network_id, deposit_cnt);

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

UPDATE mt.root SET deposit_cnt = deposit_cnt + 1;

DROP INDEX IF EXISTS mt.rht_key_idx;

-- +migrate Up

ALTER TABLE mt.rht DROP COLUMN IF EXISTS root_id;
ALTER TABLE mt.root DROP COLUMN IF EXISTS id;

ALTER TABLE sync.deposit DROP CONSTRAINT IF EXISTS deposit_pkey;
ALTER TABLE sync.deposit ADD COLUMN id SERIAL PRIMARY KEY;

ALTER TABLE mt.root ADD COLUMN deposit_id BIGINT NOT NULL DEFAULT 1 CONSTRAINT root_deposit_id_fkey REFERENCES sync.deposit (id) ON DELETE CASCADE;
ALTER TABLE mt.root ALTER COLUMN deposit_id DROP DEFAULT;
UPDATE mt.root AS r SET deposit_id = d.id FROM sync.deposit AS d WHERE d.deposit_cnt = r.deposit_cnt AND d.network_id = r.network;

ALTER TABLE mt.rht ADD COLUMN deposit_id BIGINT NOT NULL DEFAULT 1 CONSTRAINT rht_deposit_id_fkey REFERENCES sync.deposit (id) ON DELETE CASCADE;
ALTER TABLE mt.rht ALTER COLUMN deposit_id DROP DEFAULT;

UPDATE mt.root SET deposit_cnt = deposit_cnt - 1;

-- Create indexes
CREATE INDEX IF NOT EXISTS rht_key_idx ON mt.rht(key);

-- Delete duplicates
CREATE TABLE mt.rht_temp AS (SELECT key, min(value), max(deposit_id) FROM mt.rht GROUP BY key HAVING count(key) > 1);
DELETE FROM mt.rht where key in (select key FROM mt.rht_temp);
INSERT INTO mt.rht(key, value, deposit_id)  (SELECT b.key, b.min, b.max FROM mt.rht_temp b);

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
