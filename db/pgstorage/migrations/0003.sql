-- +migrate Down
DROP TABLE IF EXISTS sync.monitored_txs;

ALTER TABLE
    sync.deposit DROP COLUMN ready_for_claim;

-- +migrate Up
ALTER TABLE
    sync.deposit
ADD
    COLUMN ready_for_claim BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE sync.monitored_txs (
    id VARCHAR PRIMARY KEY,
    block_id BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    from_addr VARCHAR NOT NULL,
    to_addr VARCHAR,
    nonce DECIMAL(78, 0) NOT NULL,
    value DECIMAL(78, 0),
    data VARCHAR,
    gas DECIMAL(78, 0) NOT NULL,
    status VARCHAR NOT NULL,
    history VARCHAR [],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
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