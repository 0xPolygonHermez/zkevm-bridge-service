-- +migrate Up
ALTER TABLE mt.root
DROP COLUMN IF EXISTS deposit_cnt;

-- +migrate Down
ALTER TABLE mt.root
ADD COLUMN deposit_cnt BIGINT;

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
