-- +migrate Down

ALTER TABLE sync.deposit DROP COLUMN IF EXISTS dest_contract_addr;

-- +migrate Up

ALTER TABLE sync.deposit ADD COLUMN IF NOT EXISTS dest_contract_addr BYTEA NOT NULL DEFAULT '\x0000000000000000000000000000000000000000';
