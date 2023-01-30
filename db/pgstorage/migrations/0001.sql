-- +migrate Down
DROP SCHEMA IF EXISTS sync CASCADE;

DROP SCHEMA IF EXISTS mt CASCADE;

-- +migrate Up
CREATE SCHEMA sync;

CREATE SCHEMA mt;

-- History
CREATE TABLE sync.block (
    id SERIAL PRIMARY KEY,
    block_num BIGINT,
    block_hash BYTEA NOT NULL,
    parent_hash BYTEA,
    network_id INTEGER,
    received_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- insert the block with block_id = 0 for the trusted exit root table
INSERT INTO
    sync.block (id, block_hash, received_at)
VALUES
    (0, '\\x0', to_timestamp(0));

CREATE TABLE sync.exit_root (
    id SERIAL,
    block_id BIGINT REFERENCES sync.block (id) ON DELETE CASCADE,
    global_exit_root BYTEA,
    exit_roots BYTEA [],
    PRIMARY KEY (id),
    CONSTRAINT UC UNIQUE (block_id, global_exit_root)
);

CREATE TABLE sync.batch (
    batch_num BIGINT PRIMARY KEY,
    sequencer BYTEA,
    raw_tx_data BYTEA,
    global_exit_root BYTEA,
    timestamp TIMESTAMP WITH TIME ZONE
);

CREATE TABLE sync.verified_batch (
    batch_num BIGINT PRIMARY KEY REFERENCES sync.batch (batch_num),
    aggregator BYTEA,
    tx_hash BYTEA,
    block_id BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE
);

CREATE TABLE sync.forced_batch (
    batch_num BIGINT,
    block_id BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    forced_batch_num BIGINT,
    sequencer BYTEA,
    global_exit_root BYTEA,
    raw_tx_data BYTEA
);

CREATE TABLE sync.deposit (
    leaf_type INTEGER,
    network_id INTEGER,
    orig_net INTEGER,
    orig_addr BYTEA NOT NULL,
    amount VARCHAR,
    dest_net INTEGER NOT NULL,
    dest_addr BYTEA NOT NULL,
    block_id BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    deposit_cnt BIGINT,
    tx_hash BYTEA NOT NULL,
    metadata BYTEA NOT NULL,
    PRIMARY KEY (network_id, deposit_cnt)
);

CREATE TABLE sync.claim (
    network_id INTEGER NOT NULL,
    index BIGINT,
    -- deposit count
    orig_net INTEGER,
    orig_addr BYTEA NOT NULL,
    amount VARCHAR,
    dest_addr BYTEA NOT NULL,
    block_id BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    tx_hash BYTEA NOT NULL,
    PRIMARY KEY (network_id, index)
);

CREATE TABLE sync.token_wrapped (
    network_id INTEGER NOT NULL,
    orig_net INTEGER,
    orig_token_addr BYTEA NOT NULL,
    wrapped_token_addr BYTEA NOT NULL,
    block_id BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    name VARCHAR,
    symbol VARCHAR,
    decimals INTEGER
);

CREATE TABLE mt.root (
    id SERIAL PRIMARY KEY,
    root BYTEA,
    deposit_cnt BIGINT,
    network INTEGER
);

CREATE TABLE mt.rht (
    root_id BIGINT NOT NULL REFERENCES mt.root (id) ON DELETE CASCADE,
    key BYTEA,
    value BYTEA []
);

-- Create indexes
CREATE INDEX IF NOT EXISTS root_network_idx ON mt.root(root, network);
CREATE INDEX IF NOT EXISTS deposit_idx ON mt.root(deposit_cnt);
CREATE INDEX IF NOT EXISTS block_idx ON sync.exit_root(block_id);
CREATE INDEX IF NOT EXISTS root_idx ON mt.root(root);
CREATE INDEX IF NOT EXISTS exit_roots_idx ON sync.exit_root(exit_roots);