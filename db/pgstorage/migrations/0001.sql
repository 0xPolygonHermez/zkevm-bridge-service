-- +migrate Down
DROP SCHEMA IF EXISTS syncv2 CASCADE;
DROP SCHEMA IF EXISTS mtv2 CASCADE;

-- +migrate Up
CREATE SCHEMA syncv2;
CREATE SCHEMA mtv2;

-- History
CREATE TABLE syncv2.block
(
    id          SERIAL PRIMARY KEY,   
    block_num   BIGINT,
    block_hash  BYTEA NOT NULL,
    parent_hash BYTEA,
    network_id  INTEGER,

    received_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- insert the block with block_id = 0 for the trusted exit root table
INSERT INTO syncv2.block (id, block_hash, received_at) VALUES (0, '\\x0', to_timestamp(0));

CREATE TABLE syncv2.exit_root
(
    id                      SERIAL,
    block_id                BIGINT REFERENCES syncv2.block (id) ON DELETE CASCADE,
    global_exit_root        BYTEA,
    exit_roots              BYTEA[],
    PRIMARY KEY (id),
    CONSTRAINT UC UNIQUE (block_id, global_exit_root)
);

CREATE TABLE syncv2.batch
(
    batch_num            BIGINT PRIMARY KEY,
    sequencer            BYTEA,
    raw_tx_data          BYTEA, 
    global_exit_root     BYTEA,
    timestamp            TIMESTAMP WITH TIME ZONE
);

CREATE TABLE syncv2.verified_batch
(
    batch_num   BIGINT PRIMARY KEY REFERENCES syncv2.batch (batch_num),
    aggregator  BYTEA,
    tx_hash     BYTEA,
    block_id    BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE
);

CREATE TABLE syncv2.forced_batch
(
    batch_num            BIGINT,
    block_id             BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    forced_batch_num     BIGINT,
    sequencer            BYTEA,
    global_exit_root     BYTEA,
    raw_tx_data          BYTEA
);

CREATE TABLE syncv2.deposit
(
    leaf_type   INTEGER,
    network_id  INTEGER,
    orig_net    INTEGER,
    orig_addr  BYTEA NOT NULL,
    amount      VARCHAR,
    dest_net    INTEGER NOT NULL,
    dest_addr   BYTEA NOT NULL,
    block_id    BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    deposit_cnt BIGINT,
    tx_hash     BYTEA NOT NULL,
    metadata    BYTEA NOT NULL,
    PRIMARY KEY (network_id, deposit_cnt)
);

CREATE TABLE syncv2.claim
(
    network_id  INTEGER NOT NULL,
    index       BIGINT, -- deposit count
    orig_net    INTEGER,
    orig_addr  BYTEA NOT NULL,
    amount      VARCHAR,
    dest_addr   BYTEA NOT NULL,
    block_id    BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    tx_hash     BYTEA NOT NULL,
    PRIMARY KEY (network_id, index)
);

CREATE TABLE syncv2.token_wrapped
(
    network_id         INTEGER NOT NULL,
    orig_net           INTEGER,
    orig_token_addr    BYTEA NOT NULL,
    wrapped_token_addr BYTEA NOT NULL,
    block_id           BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    name               VARCHAR,
    symbol             VARCHAR,
    decimals           INTEGER
);

CREATE TABLE mtv2.rht 
(
    key BYTEA PRIMARY KEY,
    value BYTEA[]
);

CREATE TABLE mtv2.root
(
    root        BYTEA,
    deposit_cnt BIGINT,
    network     INTEGER,
    PRIMARY KEY(deposit_cnt, network)
);