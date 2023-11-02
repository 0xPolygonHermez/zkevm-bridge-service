-- +migrate Up
ALTER TABLE mt.root
DROP COLUMN IF EXISTS deposit_cnt;

CREATE TABLE IF NOT EXISTS mt.rollup_exit
(
	id        BIGSERIAL PRIMARY KEY,
    leaf      BYTEA,
    rollup_id BIGINT,
	root      BYTEA,
	block_num BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE
);

-- +migrate Down
ALTER TABLE mt.root
ADD COLUMN deposit_cnt BIGINT;

DROP TABLE IF EXISTS mt.rollup_exit;

-- +migrate StatementBegin
DO $$
DECLARE
	rt RECORD;
BEGIN
	FOR rt IN SELECT * FROM mt.root 
	LOOP
		UPDATE mt.root SET deposit_cnt = (SELECT deposit_cnt FROM sync.deposit WHERE id = rt.deposit_id) WHERE deposit_id = rt.deposit_id;
	END LOOP;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd
