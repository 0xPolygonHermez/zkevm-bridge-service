-- +migrate Down
DROP SCHEMA IF EXISTS sync CASCADE;
DROP SCHEMA IF EXISTS merkletree CASCADE;

-- +migrate Up
CREATE SCHEMA sync;
CREATE SCHEMA merkletree;

-- History

CREATE TABLE merkletree.rht 
(
    key BYTEA PRIMARY KEY,
    value BYTEA[]
);

CREATE TABLE merkletree.root
(
    root BYTEA,
    network CHAR NOT NULL,
    deposit_cnt BIGINT NOT NULL,
    PRIMARY KEY(root, network)
);


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
    block_id                BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num               BIGINT NOT NULL,
    global_exit_root_num    BIGINT,
    global_exit_root_l2_num BIGINT,
    mainnet_exit_root       BYTEA,
    rollup_exit_root        BYTEA
);

CREATE TABLE sync.batch
(
    batch_num            BIGINT,
    tx_hash              BYTEA,
    block_id             BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
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

CREATE TABLE sync.deposit
(
    network_id  INTEGER,
    orig_net    INTEGER,
    token_addr  BYTEA NOT NULL,
    amount      VARCHAR,
    dest_net    INTEGER NOT NULL,
    dest_addr   BYTEA NOT NULL,
    block_id    BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num   BIGINT NOT NULL,
    deposit_cnt BIGINT,
    tx_hash     BYTEA NOT NULL,
    PRIMARY KEY (network_id, deposit_cnt)
);

CREATE TABLE sync.claim
(
    network_id  INTEGER NOT NULL,
    index       BIGINT, -- deposit count
    orig_net    integer,
    token_addr  BYTEA NOT NULL,
    amount      VARCHAR,
    dest_addr   BYTEA NOT NULL,
    block_id    BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num   BIGINT NOT NULL,
    tx_hash     BYTEA NOT NULL,
    PRIMARY KEY (network_id, index)
);

CREATE TABLE sync.token_wrapped
(
    item_id            SERIAL PRIMARY KEY,
    network_id         INTEGER NOT NULL,
    orig_net           integer,
    orig_token_addr    BYTEA NOT NULL,
    wrapped_token_addr BYTEA NOT NULL,
    block_id           BIGINT NOT NULL REFERENCES sync.block (id) ON DELETE CASCADE,
    block_num          BIGINT NOT NULL
);