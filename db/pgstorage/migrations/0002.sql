-- +migrate Up

CREATE SCHEMA merkletree;

CREATE TABLE merkletree.rht 
(
    key BYTEA PRIMARY KEY,
    value BYTEA[],
    depth CHAR NOT NULL,
    network CHAR NOT NULL,
    deposit_cnt BIGINT NOT NULL
);
