-- +migrate Up
CREATE SCHEMA sync;

-- History
CREATE TABLE sync.block
(
    id          SERIAL PRIMARY KEY,   
    block_num   BIGINT,
    block_hash  BYTEA NOT NULL,
    parent_hash BYTEA,
    network_id  INTEGER,

    received_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE sync.exit_root
(
    block_id             BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num            BIGINT NOT NULL,
    global_exit_root_num BIGINT,
    mainnet_exit_root    BYTEA,
    rollup_exit_root     BYTEA,
    network_id           INTEGER
);

CREATE TABLE sync.batch
(
    batch_num            BIGINT,
    batch_hash           BYTEA,
    block_id             BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num            BIGINT NOT NULL,
    sequencer            BYTEA,
    aggregator           BYTEA,
    consolidated_tx_hash BYTEA,
    header               jsonb,
    uncles               jsonb,
    chain_id             BIGINT,
    global_exit_root     BYTEA,
    network_id           INTEGER,

    received_at     TIMESTAMP WITH TIME ZONE NOT NULL,
    consolidated_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY(batch_num, network_id)
);

CREATE TABLE sync.deposit
(
    orig_net    INTEGER,
    token_addr  BYTEA NOT NULL,
    amount      VARCHAR,
    dest_net    INTEGER NOT NULL,
    dest_addr   BYTEA NOT NULL,
    block_id    BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num   BIGINT NOT NULL,
    deposit_cnt BIGINT,
    PRIMARY KEY (orig_net, deposit_cnt)
);

CREATE TABLE sync.claim
(
    index       BIGINT PRIMARY KEY, -- deposit count
    orig_net    integer,
    token_addr  BYTEA NOT NULL,
    amount      VARCHAR,
    dest_net    INTEGER NOT NULL,
    dest_addr   BYTEA NOT NULL,
    block_id    BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num   BIGINT NOT NULL
);

CREATE TABLE sync.token_wrapped
(
    item_id            SERIAL PRIMARY KEY,
    orig_net           integer,
    orig_token_addr    BYTEA NOT NULL,
    dest_net           INTEGER NOT NULL,
    wrapped_token_addr BYTEA NOT NULL,
    block_id           BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num          BIGINT NOT NULL
);