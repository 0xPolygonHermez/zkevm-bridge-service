-- +migrate Up
CREATE SCHEMA sync;

-- History
CREATE TABLE sync.block
(
    block_num   BIGINT PRIMARY KEY,
    block_hash  BYTEA NOT NULL,
    parent_hash BYTEA,

    received_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE sync.l2_block
(
    block_num   BIGINT PRIMARY KEY,
    block_hash  BYTEA NOT NULL,
    parent_hash BYTEA,

    received_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE sync.exit_root
(
    block_num            BIGINT NOT NULL REFERENCES sync.block (block_num) ON DELETE CASCADE,
    global_exit_root_num BIGINT,
    mainnet_exit_root    BYTEA,
    rollup_exit_root     BYTEA
);

CREATE TABLE sync.batch
(
    batch_num            BIGINT PRIMARY KEY,
    batch_hash           BYTEA,
    block_num            BIGINT NOT NULL REFERENCES sync.block (block_num) ON DELETE CASCADE,
    sequencer            BYTEA,
    aggregator           BYTEA,
    consolidated_tx_hash BYTEA,
    header               jsonb,
    uncles               jsonb,
    chain_id             BIGINT,
    global_exit_root     BYTEA,

    received_at     TIMESTAMP WITH TIME ZONE NOT NULL,
    consolidated_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE sync.deposit
(
    orig_net    integer,
    token_addr  BYTEA NOT NULL,
    amount      DECIMAL(78, 0),
    dest_net    integer,
    dest_addr   BYTEA NOT NULL,
    block_num   BIGINT NOT NULL REFERENCES sync.block (block_num) ON DELETE CASCADE,
    deposit_cnt BIGINT PRIMARY KEY
);

CREATE TABLE sync.l2_deposit
(
    orig_net    integer,
    token_addr  BYTEA NOT NULL,
    amount      DECIMAL(78, 0),
    dest_net    integer,
    dest_addr   BYTEA NOT NULL,
    l2_block_num   BIGINT NOT NULL REFERENCES sync.l2_block (block_num) ON DELETE CASCADE,
    deposit_cnt BIGINT PRIMARY KEY
);

CREATE TABLE sync.claim
(
    index       BIGINT PRIMARY KEY
    orig_net    integer,
    token_addr  BYTEA NOT NULL,
    amount      DECIMAL(78, 0),
    dest_addr   BYTEA NOT NULL,
    block_num   BIGINT NOT NULL REFERENCES sync.block (block_num) ON DELETE CASCADE,
);

CREATE TABLE sync.l2_claim
(
    index       BIGINT PRIMARY KEY
    orig_net    integer,
    token_addr  BYTEA NOT NULL,
    amount      DECIMAL(78, 0),
    dest_addr   BYTEA NOT NULL,
    l2_block_num   BIGINT NOT NULL REFERENCES sync.l2_block (block_num) ON DELETE CASCADE,
);

CREATE TABLE sync.token_wrapped
(
    item_id            SERIAL PRIMARY KEY
    orig_net           integer,
    orig_token_addr    BYTEA NOT NULL,
    wrapped_token_addr BYTEA NOT NULL,
    block_num          BIGINT NOT NULL REFERENCES sync.block (block_num) ON DELETE CASCADE,
);

CREATE TABLE sync.l2_token_wrapped
(
    item_id            SERIAL PRIMARY KEY
    orig_net           integer,
    orig_token_addr    BYTEA NOT NULL,
    wrapped_token_addr BYTEA NOT NULL,
    l2_block_num          BIGINT NOT NULL REFERENCES sync.l2_block (block_num) ON DELETE CASCADE,
);