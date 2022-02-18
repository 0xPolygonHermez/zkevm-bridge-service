-- +migrate Up
CREATE SCHEMA sync;

-- History
CREATE TABLE sync.block
(
    block_num   BIGINT PRIMARY KEY,
    block_hash  BYTEA                       NOT NULL,
    parent_hash BYTEA,

    received_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE sync.deposit
(
    orig_net   integer,
    token_addr BYTEA NOT NULL,
    amount DECIMAL(78, 0),
    dest_net integer,
    dest_addr BYTEA NOT NULL,
    block_num BIGINT,
    deposit_cnt BIGINT PRIMARY KEY
);