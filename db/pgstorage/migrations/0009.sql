-- +migrate Up

ALTER TABLE sync.deposit
ADD COLUMN IF NOT EXISTS origin_rollup_id BIGINT DEFAULT 0;

ALTER TABLE sync.claim DROP CONSTRAINT claim_pkey;
ALTER TABLE sync.claim ADD PRIMARY KEY (index, rollup_index, mainnet_flag);

-- +migrate Down

ALTER TABLE sync.claim DROP CONSTRAINT claim_pkey;
ALTER TABLE sync.claim ADD PRIMARY KEY (network_id, index);

ALTER TABLE sync.deposit
DROP COLUMN IF EXISTS origin_rollup_id;