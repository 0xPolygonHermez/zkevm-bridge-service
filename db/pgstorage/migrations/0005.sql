-- +migrate Down

CREATE TABLE IF NOT EXISTS sync.batch
(
    batch_num            BIGINT PRIMARY KEY,
    sequencer            BYTEA,
    raw_tx_data          BYTEA, 
    global_exit_root     BYTEA,
    timestamp            TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS sync.verified_batch
(
    batch_num   BIGINT PRIMARY KEY REFERENCES sync.batch (batch_num),
    aggregator  BYTEA,
    tx_hash     BYTEA,
    block_id    BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sync.forced_batch
(
    batch_num            BIGINT,
    block_id             BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    forced_batch_num     BIGINT,
    sequencer            BYTEA,
    global_exit_root     BYTEA,
    raw_tx_data          BYTEA
);

-- +migrate Up

DROP TABLE IF EXISTS sync.verified_batch;
DROP TABLE IF EXISTS sync.forced_batch;
DROP TABLE IF EXISTS sync.batch;
