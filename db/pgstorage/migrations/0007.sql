-- +migrate Down
DROP TABLE IF EXISTS sync.block_test CASCADE;
DROP TABLE IF EXISTS sync.exit_root_test CASCADE;
DROP TABLE IF EXISTS sync.deposit_test CASCADE;
DROP TABLE IF EXISTS sync.claim_test CASCADE;
DROP TABLE IF EXISTS sync.token_wrapped_test CASCADE;
DROP TABLE IF EXISTS sync.monitored_txs_test CASCADE;
DROP TABLE IF EXISTS mt.rht_test CASCADE;
DROP TABLE IF EXISTS mt.root_test CASCADE;
DROP TABLE IF EXISTS common.main_coins_test CASCADE;

-- +migrate Up
CREATE TABLE IF NOT EXISTS sync.block_test (LIKE sync.block INCLUDING ALL);
CREATE TABLE IF NOT EXISTS sync.exit_root_test (LIKE sync.exit_root INCLUDING ALL);
CREATE TABLE IF NOT EXISTS sync.deposit_test (LIKE sync.deposit INCLUDING ALL);
CREATE TABLE IF NOT EXISTS sync.claim_test (LIKE sync.claim INCLUDING ALL);
CREATE TABLE IF NOT EXISTS sync.token_wrapped_test (LIKE sync.token_wrapped INCLUDING ALL);
CREATE TABLE IF NOT EXISTS sync.monitored_txs_test (LIKE sync.monitored_txs INCLUDING ALL);
CREATE TABLE IF NOT EXISTS mt.rht_test (LIKE mt.rht INCLUDING ALL);
CREATE TABLE IF NOT EXISTS mt.root_test (LIKE mt.root INCLUDING ALL);
CREATE TABLE IF NOT EXISTS common.main_coins_test (LIKE common.main_coins INCLUDING ALL);

ALTER TABLE sync.exit_root_test ADD CONSTRAINT exit_root_block_id_fkey FOREIGN KEY (block_id) REFERENCES sync.block_test (id) ON DELETE CASCADE;
ALTER TABLE sync.deposit_test ADD CONSTRAINT deposit_block_id_fkey FOREIGN KEY (block_id) REFERENCES sync.block_test (id) ON DELETE CASCADE;
ALTER TABLE sync.claim_test ADD CONSTRAINT claim_block_id_fkey FOREIGN KEY (block_id) REFERENCES sync.block_test (id) ON DELETE CASCADE;
ALTER TABLE sync.token_wrapped_test ADD CONSTRAINT claim_block_id_fkey FOREIGN KEY (block_id) REFERENCES sync.block_test (id) ON DELETE CASCADE;
ALTER TABLE sync.monitored_txs_test ADD CONSTRAINT monitored_txs_block_id_fkey FOREIGN KEY (block_id) REFERENCES sync.block_test (id) ON DELETE CASCADE;
ALTER TABLE mt.rht_test ADD CONSTRAINT rht_deposit_id_fkey FOREIGN KEY (deposit_id) REFERENCES sync.deposit_test (id) ON DELETE CASCADE;
ALTER TABLE mt.root_test ADD CONSTRAINT root_deposit_id_fkey FOREIGN KEY (deposit_id) REFERENCES sync.deposit_test (id) ON DELETE CASCADE;