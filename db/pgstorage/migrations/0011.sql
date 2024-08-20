-- +migrate Up

ALTER TABLE sync.claim DROP CONSTRAINT claim_pkey;
ALTER TABLE sync.claim ADD PRIMARY KEY (index, rollup_index, network_id);

-- +migrate Down

ALTER TABLE sync.claim DROP CONSTRAINT claim_pkey;
ALTER TABLE sync.claim ADD PRIMARY KEY (network_id, index);