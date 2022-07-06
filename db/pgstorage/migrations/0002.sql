-- +migrate Down
DROP SCHEMA IF EXISTS syncv2 CASCADE;
DROP SCHEMA IF EXISTS mtv2 CASCADE;

-- +migrate Up
CREATE SCHEMA syncv2;
CREATE SCHEMA mtv2;

-- History

CREATE TABLE mtv2.rht 
(
    key BYTEA PRIMARY KEY,
    value BYTEA[]
);

CREATE TABLE mtv2.root
(
    root       BYTEA,
    deposit_id BIGINT PRIMARY KEY REFERENCES syncv2.deposit (id) ON DELETE CASCADE,
);

CREATE TABLE syncv2.block
(
    id          SERIAL PRIMARY KEY,   
    block_num   BIGINT,
    block_hash  BYTEA NOT NULL,
    parent_hash BYTEA,
    network_id  INTEGER,

    received_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE syncv2.exit_root
(
    block_id                BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    global_exit_root_num    BIGINT,
    global_exit_root        BYTEA,
    mainnet_root_id         BIGINT,
    rollup_root_id          BIGINT
);

CREATE TABLE syncv2.batch
(
    batch_num            BIGINT,
    tx_hash              BYTEA,
    block_id             BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    block_num            BIGINT NOT NULL,
    sequencer            BYTEA,
    aggregator           BYTEA,
    consolidated_tx_hash BYTEA,
    uncles               jsonb,
    chain_id             BIGINT,
    global_exit_root     BYTEA,
    network_id           INTEGER,

    received_at     TIMESTAMP WITH TIME ZONE NOT NULL,
    consolidated_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY(batch_num, network_id)
);

CREATE TABLE syncv2.deposit
(
    id          SERIAL PRIMARY KEY,   
    network_id  INTEGER,
    orig_net    INTEGER,
    token_addr  BYTEA NOT NULL,
    amount      VARCHAR,
    dest_net    INTEGER NOT NULL,
    dest_addr   BYTEA NOT NULL,
    block_id    BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    block_num   BIGINT NOT NULL,
    deposit_cnt BIGINT,
    tx_hash     BYTEA NOT NULL,
);

CREATE TABLE syncv2.claim
(
    network_id  INTEGER NOT NULL,
    index       BIGINT, -- deposit count
    orig_net    integer,
    token_addr  BYTEA NOT NULL,
    amount      VARCHAR,
    dest_addr   BYTEA NOT NULL,
    block_id    BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    block_num   BIGINT NOT NULL,
    tx_hash     BYTEA NOT NULL,
    PRIMARY KEY (network_id, index)
);

CREATE TABLE syncv2.token_wrapped
(
    item_id            SERIAL PRIMARY KEY,
    network_id         INTEGER NOT NULL,
    orig_net           integer,
    orig_token_addr    BYTEA NOT NULL,
    wrapped_token_addr BYTEA NOT NULL,
    block_id           BIGINT NOT NULL REFERENCES syncv2.block (id) ON DELETE CASCADE,
    block_num          BIGINT NOT NULL
);