-- +migrate Down
ALTER TABLE syncv2.claim 
RENAME COLUMN orig_addr TO token_addr;

-- +migrate Up
ALTER TABLE syncv2.claim 
RENAME COLUMN token_addr TO orig_addr;
