-- +migrate Down
DROP INDEX IF EXISTS mt.rht_key_idx;

DROP TABLE IF EXISTS sync.monitored_txs;

ALTER TABLE
    sync.deposit DROP COLUMN ready_for_claim;

ALTER TABLE
    sync.block DROP CONSTRAINT block_hash_unique;

ALTER TABLE
    mt.rht
ADD
    PRIMARY KEY (key);

-- +migrate Up
ALTER TABLE
    mt.rht DROP CONSTRAINT rht_pkey;

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

-- +migrate StatementBegin
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

-- +migrate StatementEnd

-- Create indexes
CREATE INDEX IF NOT EXISTS rht_key_idx ON mt.rht(key);