package migrations_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// This migration creates a different proof table dropping all the information.

type migrationTest0003 struct{}

func (m migrationTest0003) InsertData(db *sql.DB) error {
	// Insert initial blocks and deposits to rht table
	var blockID uint64
	if err := db.QueryRow("INSERT INTO sync.block (block_num, block_hash, parent_hash, network_id, received_at) VALUES ($1, $2, $3, $4, $5) RETURNING id", 1,
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0, time.Now()).Scan(&blockID); err != nil {
		return err
	}
	const addDeposit = "INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)"
	if _, err := db.Exec(addDeposit,
		0, 0, 0, common.FromHex("0x0000000000000000000000000000000000000000"),
		"1000000", 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), blockID, 0,
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		[]byte{}); err != nil {
		return err
	}

	return nil
}

func (m migrationTest0003) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// Insert a monitored tx
	_, err := db.Exec(`INSERT INTO sync.monitored_txs 
	(id, block_id, from_addr, to_addr, nonce, value, data, gas, status, history, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`, 0, 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, "10000", []byte{}, 5000000, "crerated", nil, time.Now(), time.Now())
	assert.NoError(t, err)
	// Insert a new Deposit
	_, err = db.Exec("INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, ready_for_claim) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)",
		0, 0, 0, common.FromHex("0x0000000000000000000000000000000000000000"),
		"1000000", 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, 1,
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		[]byte{}, true)
	assert.NoError(t, err)
	// Get a Deposit count
	const getDeposit = "SELECT COUNT(*) FROM sync.deposit WHERE ready_for_claim = true"
	var depositCount int
	err = db.QueryRow(getDeposit).Scan(&depositCount)
	assert.NoError(t, err)
	assert.Equal(t, depositCount, 1)
	// Insert a block
	var blockID uint64
	err = db.QueryRow(`WITH block_id AS 
	(INSERT INTO sync.block (block_num, block_hash, parent_hash, network_id, received_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (block_hash) DO NOTHING RETURNING id)
	SELECT * from block_id
	UNION ALL
	SELECT id FROM sync.block WHERE block_hash = $2;`, 1,
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0, time.Now()).Scan(&blockID)
	assert.NoError(t, err)
	assert.Equal(t, blockID, uint64(1))

	indexes := []string{"rht_key_idx"}

	// Check indexes adding
	for _, idx := range indexes {
		// getIndex
		const getIndex = `SELECT count(*) FROM pg_indexes WHERE indexname = $1;`
		row := db.QueryRow(getIndex, idx)
		var result int
		assert.NoError(t, row.Scan(&result))
		assert.Equal(t, 1, result)
	}
}

func (m migrationTest0003) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	// Insert a block
	var blockID uint64
	err := db.QueryRow(`INSERT INTO sync.block (block_num, block_hash, parent_hash, network_id, received_at) 
	VALUES ($1, $2, $3, $4, $5) RETURNING id;`, 1,
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0, time.Now()).Scan(&blockID)
	assert.NoError(t, err)
	assert.Equal(t, blockID, uint64(3))

	indexes := []string{"rht_key_idx"}

	// Check indexes removing
	for _, idx := range indexes {
		// getIndex
		const getIndex = `SELECT count(*) FROM pg_indexes WHERE indexname = $1;`
		row := db.QueryRow(getIndex, idx)
		var result int
		assert.NoError(t, row.Scan(&result))
		assert.Equal(t, 0, result)
	}
}

func TestMigration0003(t *testing.T) {
	runMigrationTest(t, 3, migrationTest0003{})
}
