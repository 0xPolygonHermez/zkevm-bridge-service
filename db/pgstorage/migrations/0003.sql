-- +migrate Down

ALTER TABLE mt.rht DROP COLUMN IF EXISTS deposit_id;
ALTER TABLE mt.root DROP COLUMN IF EXISTS deposit_id;
ALTER TABLE sync.deposit DROP COLUMN IF EXISTS id;

ALTER TABLE sync.deposit ADD CONSTRAINT deposit_pkey PRIMARY KEY (network_id, deposit_cnt);

DROP INDEX IF EXISTS mt.rht_key_idx;
DROP TABLE IF EXISTS sync.monitored_txs;

ALTER TABLE sync.deposit DROP COLUMN ready_for_claim;

ALTER TABLE sync.block DROP CONSTRAINT block_hash_unique;

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

ALTER TABLE
    sync.deposit
ADD
    COLUMN ready_for_claim BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE
    sync.block
ADD
    CONSTRAINT block_hash_unique UNIQUE (block_hash);

CREATE TABLE sync.monitored_txs (
    id BIGINT PRIMARY KEY,
    block_id BIGINT REFERENCES sync.block (id) ON DELETE CASCADE,
    from_addr BYTEA NOT NULL,
    to_addr BYTEA,
    nonce BIGINT NOT NULL,
    value VARCHAR,
    data BYTEA,
    gas BIGINT NOT NULL,
    status VARCHAR NOT NULL,
    history BYTEA [],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

UPDATE mt.root SET deposit_cnt = deposit_cnt - 1;

UPDATE
    sync.deposit
SET
    ready_for_claim = true
WHERE
    deposit_cnt <= (
        SELECT
            deposit_cnt
        FROM
            mt.root
        WHERE
            root = (
                SELECT
                    exit_roots [1]
                FROM
                    sync.exit_root
                WHERE
                    block_id = 0
                ORDER BY
                    id DESC
                LIMIT
                    1
            )
            AND network = 0
    )
    AND network_id = 0;

UPDATE
    sync.deposit
SET
    ready_for_claim = true
WHERE
    deposit_cnt <= (
        SELECT
            deposit_cnt
        FROM
            mt.root
        WHERE
            root = (
                SELECT
                    exit_roots [2]
                FROM
                    sync.exit_root
                WHERE
                    block_id > 0
                ORDER BY
                    id DESC
                LIMIT
                    1
            )
            AND network = 1
    )
    AND network_id != 0;


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
CREATE INDEX IF NOT EXISTS rht_key_idx ON mt.rht(key);