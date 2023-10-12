-- +migrate Down
DROP SCHEMA IF EXISTS common;

-- +migrate Up
CREATE SCHEMA IF NOT EXISTS common;
CREATE TABLE IF NOT EXISTS common.main_coins
(
    id          SERIAL PRIMARY KEY,
    symbol      TEXT NOT NULL,
    name        TEXT NOT NULL,
    decimals    INTEGER NOT NULL DEFAULT 0,
    address     BYTEA NOT NULL,
    chain_id    INTEGER NOT NULL,
    network_id  INTEGER NOT NULL,
    logo_link   TEXT NOT NULL DEFAULT '',
    is_deleted   BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INTEGER NOT NULL DEFAULT 0,
    create_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    modify_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);