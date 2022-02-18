-- +migrate Up

CREATE SCHEMA merkletree;

CREATE TABLE merkletree.mainnet 
(
    key BYTEA PRIMARY KEY,
    value BYTEA NOT NULL
);

CREATE TABLE merkletree.rollup 
(
    key BYTEA PRIMARY KEY,
    value BYTEA NOT NULL
);