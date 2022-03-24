-- +migrate Up

CREATE SCHEMA merkletree;

CREATE TABLE merkletree.rht 
(
    key BYTEA PRIMARY KEY,
    value BYTEA[],
    depth CHAR NOT NULL,
    network CHAR NOT NULL,
    -- To parse uint8 to char in database, we are using one-index in the database
    deposit_cnt BIGINT NOT NULL
);
