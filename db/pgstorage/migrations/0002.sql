-- +migrate Up

CREATE SCHEMA merkletree;
CREATE SCHEMA bridgetree;

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

CREATE TABLE bridgetree.root_track 
(
    index BIGINT PRIMARY KEY,
    global_root BYTEA NOT NULL,
    roots BYTEA[]
);