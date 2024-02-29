package migrations_test

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type migrationTest0009 struct{}

const (
	originalDepositSQL = `
		INSERT INTO sync.claim (block_id, network_id, index, mainnet_flag, rollup_index, orig_addr, dest_addr, tx_hash)
		VALUES(69, 0, 1, false, 0, decode('00','hex'), decode('00','hex'), decode('00','hex'));
	` // Rollup 1 to L1
	conflictingDeposit = `
		INSERT INTO sync.claim (block_id, network_id, index, mainnet_flag, rollup_index, orig_addr, dest_addr, tx_hash) 
		VALUES(69, 0, 1, false, 1, decode('00','hex'), decode('00','hex'), decode('00','hex'));
	` // Rollup 2 to L1
)

func (m migrationTest0009) InsertData(db *sql.DB) error {
	block := "INSERT INTO sync.block (id, block_num, block_hash, parent_hash, network_id, received_at) VALUES(69, 2803824, decode('27474F16174BBE50C294FE13C190B92E42B2368A6D4AEB8A4A015F52816296C3','hex'), decode('C9B5033799ADF3739383A0489EFBE8A0D4D5E4478778A4F4304562FD51AE4C07','hex'), 1, '0001-01-01 01:00:00.000');"
	if _, err := db.Exec(block); err != nil {
		return err
	}

	if _, err := db.Exec(originalDepositSQL); err != nil {
		return err
	}
	_, err := db.Exec(conflictingDeposit)
	if err == nil || !strings.Contains(err.Error(), "ERROR: duplicate key value violates unique constraint \"claim_pkey\" (SQLSTATE 23505)") {
		return errors.New("should violate primary key")
	}

	return nil
}

func (m migrationTest0009) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// check that original row still in there
	selectClaim := `SELECT block_id, network_id, index, mainnet_flag, rollup_index FROM sync.claim;`
	row := db.QueryRow(selectClaim)
	var (
		block_id, network_id, index, rollup_index int
		mainnet_flag                              bool
	)
	assert.NoError(t, row.Scan(&block_id, &network_id, &index, &mainnet_flag, &rollup_index))
	assert.Equal(t, 69, block_id)
	assert.Equal(t, 0, network_id)
	assert.Equal(t, 1, index)
	assert.Equal(t, false, mainnet_flag)
	assert.Equal(t, 0, rollup_index)

	// Add deposit that originally would have caused pkey violation
	_, err := db.Exec(conflictingDeposit)
	assert.NoError(t, err)

	// Remove conflicting deposit so it's possible to run the migration down
	_, err = db.Exec("DELETE FROM sync.claim WHERE rollup_index = 1;")
	assert.NoError(t, err)

}

func (m migrationTest0009) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	// check that original row still in there
	selectClaim := `SELECT block_id, network_id, index, mainnet_flag, rollup_index FROM sync.claim;`
	row := db.QueryRow(selectClaim)
	var (
		block_id, network_id, index, rollup_index int
		mainnet_flag                              bool
	)
	assert.NoError(t, row.Scan(&block_id, &network_id, &index, &mainnet_flag, &rollup_index))
	assert.Equal(t, 69, block_id)
	assert.Equal(t, 0, network_id)
	assert.Equal(t, 1, index)
	assert.Equal(t, false, mainnet_flag)
	assert.Equal(t, 0, rollup_index)
}

func TestMigration0009(t *testing.T) {
	runMigrationTest(t, 9, migrationTest0009{})
}
