package migrations_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// this migration changes length of the token name
type migrationTest0010 struct{}

func (m migrationTest0010) InsertData(db *sql.DB) error {
	addBlocks := `
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(1, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b20', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50fe', '2024-03-11 02:52:23.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(2, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b21', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f1', '2024-03-11 02:52:24.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(3, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b22', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f2', '2024-03-11 02:52:25.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(4, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b23', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f3', '2024-03-11 02:52:26.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(5, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b24', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f4', '2024-03-11 02:52:27.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(6, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b25', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f5', '2024-03-11 02:52:28.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(7, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b26', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f6', '2024-03-11 02:52:29.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(8, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b27', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f7', '2024-03-11 02:52:30.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(9, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b28', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f8', '2024-03-11 02:52:31.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(10, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b29', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50f9', '2024-03-11 02:52:32.000', 0);
	INSERT INTO sync.block
	(block_num, block_hash, parent_hash, received_at, network_id)
	VALUES(11, '0x013be63487a53c874614dd1ae0434cf211e393b2e386c8fde74da203b5469b2a', '0x0328698ebeda498df8c63040e2a4771d24722ab2c1e8291226b9215c7eec50fa', '2024-03-11 02:52:33.000', 0);
	INSERT INTO sync.claim
	(network_id, "index", orig_net, orig_addr, amount, dest_addr, block_id, tx_hash, rollup_index, mainnet_flag)
	VALUES(1, 0, 0, decode('0000000000000000000000000000000000000000','hex'), '100000000000000000', decode('F35960302A07022ABA880DFFAEC2FDD64D5BF1C1','hex'), 2, decode('3E24EC7286B5138DE66E8B2B854EE957579B2651B3A454AD32C55A985364FAFF','hex'), 0, false);
	INSERT INTO sync.deposit
	(leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, id, ready_for_claim)
	VALUES(0, 1, 0, decode('0000000000000000000000000000000000000000','hex'), '2000000000000000', 0, decode('2536C2745AC4A584656A830F7BDCD329C94E8F30','hex'), 3, 1, decode('1E615900D623C9291992C79ED156A950BE7DA69B8E58A67DC6F2BCDE2EB236FC','hex'), decode('','hex'), 2, true);
	INSERT INTO sync.token_wrapped
	(network_id, orig_net, orig_token_addr, wrapped_token_addr, block_id, "name", symbol, decimals)
	VALUES(1, 0, decode('5AA6D983DECB146A5810BB28CCD2554F29176AB6','hex'), decode('6014E48D6C37CD0953E86F511CF04DDD7C37029D','hex'), 5, 'ToniToken', 'TRM', 18);
	INSERT INTO mt.rollup_exit
	(id, leaf, rollup_id, root, block_id)
	VALUES(1, decode('4C907345C62B48529CE718F3A32E8BE63A3AE02831386A638419C6CBE6606558','hex'), 1, decode('3CAF4160ABD2C2160305420728BDFECC882456DCA00247FEAFC2C00ADA3E19E0','hex'), 6);
	INSERT INTO sync.exit_root
	(id, block_id, global_exit_root, exit_roots)
	VALUES(1, 8, decode('AD3228B676F7D3CD4284A5443F17F1962B36E491B30A40B2405849E597BA5FB5','hex'), '{decode(''5C7830303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030'',''hex''),decode(''5C7830303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030'',''hex'')}');
	`
	if _, err := db.Exec(addBlocks); err != nil {
		return err
	}
	blockCount := `SELECT count(*) FROM sync.block`
	var count int
	err := db.QueryRow(blockCount).Scan(&count)
	if err != nil {
		return err
	}
	if count != 12 {
		return fmt.Errorf("error: initial wrong number of blocks: %d", count)
	}
	return nil
}

func (m migrationTest0010) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	blockCount := `SELECT count(*) FROM sync.block`
	var count int
	err := db.QueryRow(blockCount).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 6, count)
}

func (m migrationTest0010) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
}

func TestMigration0010(t *testing.T) {
	runMigrationTest(t, 10, migrationTest0010{})
}
