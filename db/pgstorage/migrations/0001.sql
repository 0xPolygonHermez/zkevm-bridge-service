-- +migrate Up
CREATE SCHEMA state;

-- History
CREATE TABLE state.block
(
    block_num   BIGINT PRIMARY KEY,
    block_hash  BYTEA                       NOT NULL,
    parent_hash BYTEA,

    received_at TIMESTAMP WITH TIME ZONE NOT NULL
);

