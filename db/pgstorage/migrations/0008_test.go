package migrations_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

type migrationTest0008 struct{}

func (m migrationTest0008) InsertData(db *sql.DB) error {
	block := "INSERT INTO sync.block (id, block_num, block_hash, parent_hash, network_id, received_at) VALUES(2, 2803824, decode('27474F16174BBE50C294FE13C190B92E42B2368A6D4AEB8A4A015F52816296C3','hex'), decode('C9B5033799ADF3739383A0489EFBE8A0D4D5E4478778A4F4304562FD51AE4C07','hex'), 1, '0001-01-01 01:00:00.000');"
	if _, err := db.Exec(block); err != nil {
		return err
	}
	insertDeposit := "INSERT INTO sync.deposit(leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, id, ready_for_claim) " +
		"VALUES(0, 0, 123, decode('0000000000000000000000000000000000000000','hex'), '10000000000000000000', 456, decode('C949254D682D8C9AD5682521675B8F43B102AEC4','hex'), 2, 0, decode('C2D6575EA98EB55E36B5AC6E11196800362594458A4B3143DB50E4995CB2422E','hex'), decode('','hex'), 1, true);"
	if _, err := db.Exec(insertDeposit); err != nil {
		return err
	}
	insertDeposit2 := "INSERT INTO sync.deposit(leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, id, ready_for_claim) VALUES(0, 0, 0, decode('0000000000000000000000000000000000000000','hex'), '10000000000000000000', 1, decode('C949254D682D8C9AD5682521675B8F43B102AEC4','hex'), 2, 1, decode('C2D6575EA98EB55E36B5AC6E11196800362594458A4B3143DB50E4995CB2422E','hex'), decode('','hex'), 2, true);"
	if _, err := db.Exec(insertDeposit2); err != nil {
		return err
	}
	insertDeposit3 := "INSERT INTO sync.deposit(leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, id, ready_for_claim) VALUES(0, 0, 0, decode('0000000000000000000000000000000000000000','hex'), '10000000000000000000', 1, decode('C949254D682D8C9AD5682521675B8F43B102AEC4','hex'), 2, 2, decode('C2D6575EA98EB55E36B5AC6E11196800362594458A4B3143DB50E4995CB2422E','hex'), decode('','hex'), 3, true);"
	if _, err := db.Exec(insertDeposit3); err != nil {
		return err
	}

	insertClaim := "INSERT INTO sync.Claim (network_id, index, orig_net, orig_addr, amount, dest_addr, block_id, tx_hash) VALUES(1, 3, 1234, decode('0000000000000000000000000000000000000000','hex'), '300000000000000000', decode('14567C0DCF79C20FE1A21E36EC975D1775A1905C','hex'), 2, decode('A9505DB7D7EDD08947F12F2B1F7898148FFB43D80BCB977B78161EF14173D575','hex'));"
	if _, err := db.Exec(insertClaim); err != nil {
		return err
	}

	insertTokenWrapped := "INSERT INTO sync.token_wrapped (network_id, orig_net, orig_token_addr,wrapped_token_addr,  block_id) " +
		"VALUES(1, 1234, decode('0000000000000000000000000000000000000000','hex'),decode('0000000000000000000000000000000000000000','hex'), 2);"
	if _, err := db.Exec(insertTokenWrapped); err != nil {
		return err
	}
	return nil
}

func (m migrationTest0008) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	queryDepositCount := "select orig_net,dest_net from sync.deposit where id = 1;"
	row := db.QueryRow(queryDepositCount)
	var origNet, destNet uint32
	assert.NoError(t, row.Scan(&origNet, &destNet))
	assert.Equal(t, uint32(123), origNet)
	assert.Equal(t, uint32(456), destNet)
	var err error
	insertDeposit4 := "INSERT INTO sync.deposit(leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, id, ready_for_claim)" +
		" VALUES(0, 0, 4294967295, decode('0000000000000000000000000000000000000000','hex'), '10000000000000000000', 4294967295, decode('C949254D682D8C9AD5682521675B8F43B102AEC4','hex'), 2, 2, decode('C2D6575EA98EB55E36B5AC6E11196800362594458A4B3143DB50E4995CB2422E','hex'), decode('','hex'), 4, true);"
	_, err = db.Exec(insertDeposit4)
	assert.NoError(t, err)

	queryDepositCount4 := "select orig_net,dest_net from sync.deposit where id = 4;"
	row = db.QueryRow(queryDepositCount4)
	assert.NoError(t, row.Scan(&origNet, &destNet))
	assert.Equal(t, uint32(4294967295), origNet)
	assert.Equal(t, uint32(4294967295), destNet)

	// The new deposit need to be deleted because downgrade process will fail
	_, err = db.Exec("DELETE FROM sync.deposit WHERE id = 4;")
	assert.NoError(t, err)

	queryClaim := "select orig_net from sync.Claim where index = 3;"
	row = db.QueryRow(queryClaim)
	assert.NoError(t, row.Scan(&origNet))
	assert.Equal(t, uint32(1234), origNet)

	insertClaim := "INSERT INTO sync.Claim (network_id, index, orig_net, orig_addr, amount, dest_addr, block_id, tx_hash) " +
		"VALUES(1, 4, 4294967295, decode('0000000000000000000000000000000000000000','hex'), '300000000000000000', decode('14567C0DCF79C20FE1A21E36EC975D1775A1905C','hex'), 2, decode('A9505DB7D7EDD08947F12F2B1F7898148FFB43D80BCB977B78161EF14173D575','hex'));"
	_, err = db.Exec(insertClaim)
	assert.NoError(t, err)

	row = db.QueryRow("select orig_net from sync.Claim where index = 4;")
	assert.NoError(t, row.Scan(&origNet))
	assert.Equal(t, uint32(4294967295), origNet)
	// The new deposit need to be deleted because downgrade process will fail
	_, err = db.Exec("DELETE FROM sync.Claim WHERE index = 4;")
	assert.NoError(t, err)

	insertTokenWrapped := "INSERT INTO sync.token_wrapped (network_id, orig_net, orig_token_addr,wrapped_token_addr,  block_id) " +
		"VALUES(2, 4294967295, decode('0000000000000000000000000000000000000000','hex'),decode('0000000000000000000000000000000000000000','hex'), 2);"
	_, err = db.Exec(insertTokenWrapped)
	assert.NoError(t, err)
	row = db.QueryRow("select orig_net from sync.token_wrapped where network_id = 2;")
	assert.NoError(t, row.Scan(&origNet))
	assert.Equal(t, uint32(4294967295), origNet)

	// The new deposit need to be deleted because downgrade process will fail
	_, err = db.Exec("DELETE FROM sync.token_wrapped WHERE network_id = 2;")
	assert.NoError(t, err)
}

func (m migrationTest0008) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	queryDepositCount := "select orig_net,dest_net from sync.deposit where id = 1;"
	var err error
	var origNet, destNet int

	// Due the change of type the first call returns an error and discard cache, so next call will work
	// Discard Error: "cached plan must not change result type" "0A000"
	row := db.QueryRow(queryDepositCount)
	err = row.Scan(&origNet, &destNet)
	assert.Error(t, err) // This error is "cached plan must not change result type" "0A000"

	row = db.QueryRow(queryDepositCount)
	err = row.Scan(&origNet, &destNet)
	assert.NoError(t, err)
	assert.Equal(t, int(123), origNet)
	assert.Equal(t, int(456), destNet)

	insertDeposit4 := "INSERT INTO sync.deposit(leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, id, ready_for_claim)" +
		" VALUES(0, 0, 4294967295, decode('0000000000000000000000000000000000000000','hex'), '10000000000000000000', 4294967295, decode('C949254D682D8C9AD5682521675B8F43B102AEC4','hex'), 2, 2, decode('C2D6575EA98EB55E36B5AC6E11196800362594458A4B3143DB50E4995CB2422E','hex'), decode('','hex'), 4, true);"
	_, err = db.Exec(insertDeposit4)
	assert.Error(t, err)

	queryClaim := "select orig_net from sync.Claim where index = 3;"
	_ = db.QueryRow(queryClaim) // Discard Error: "cached plan must not change result type" "0A000"
	row = db.QueryRow(queryClaim)
	assert.NoError(t, row.Scan(&origNet))
	assert.Equal(t, int(1234), origNet)
}

func TestMigration0008(t *testing.T) {
	runMigrationTest(t, 8, migrationTest0008{})
}
