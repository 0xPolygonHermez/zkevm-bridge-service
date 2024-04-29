-- +migrate Up


CREATE TABLE IF NOT EXISTS sync.monitored_txs_group
(
    group_id       int8 PRIMARY KEY,
    status        VARCHAR NOT NULL, -- Status of the group
    deposit_ids   int8[], -- Deposit IDs in the group
    -- Num of retries done for this group
    num_retries   int4 NOT NULL,
    compressed_tx_data BYTEA NULL,
    claim_tx_history VARCHAR NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_log VARCHAR NULL
);

ALTER TABLE sync.monitored_txs
ADD COLUMN IF NOT EXISTS group_id BIGINT DEFAULT NULL;
ALTER TABLE sync.monitored_txs
ADD COLUMN IF NOT EXISTS global_exit_root BYTEA NOT NULL DEFAULT '\x0000000000000000000000000000000000000000';
-- ADD CONSTRAINT fk_group_id FOREIGN KEY (group_id) REFERENCES sync.monitored_txs_group(group_id) ON DELETE CASCADE;


-- +migrate Down
ALTER TABLE sync.monitored_txs
DROP COLUMN IF EXISTS group_id;
ALTER TABLE sync.monitored_txs
DROP COLUMN IF EXISTS global_exit_root;

DROP TABLE IF EXISTS sync.monitored_txs_group;