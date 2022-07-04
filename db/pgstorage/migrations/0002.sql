-- +migrate Up

CREATE SCHEMA merkletree;

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
