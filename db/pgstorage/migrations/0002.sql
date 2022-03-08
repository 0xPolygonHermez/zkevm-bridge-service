-- +migrate Up

CREATE SCHEMA merkletree;

CREATE TABLE merkletree.rht 
(
    key BYTEA PRIMARY KEY,
    value BYTEA[],
    network CHAR NOT NULL
);

CREATE TABLE merkletree.root_track 
(
    index BIGINT PRIMARY KEY,
    root BYTEA NOT NULL,
    network CHAR NOT NULL
);
