-- +migrate Up
ALTER TABLE sync.deposit 
ALTER COLUMN orig_net TYPE BIGINT,
ALTER COLUMN dest_net TYPE BIGINT;

ALTER TABLE sync.claim 
ALTER COLUMN orig_net TYPE BIGINT;

ALTER TABLE sync.token_wrapped
ALTER COLUMN orig_net TYPE BIGINT;


-- +migrate Down
ALTER TABLE sync.deposit
ALTER COLUMN orig_net TYPE INTEGER,
ALTER COLUMN dest_net TYPE INTEGER;

ALTER TABLE sync.claim 
ALTER COLUMN orig_net TYPE INTEGER;

ALTER TABLE sync.token_wrapped
ALTER COLUMN orig_net TYPE INTEGER;