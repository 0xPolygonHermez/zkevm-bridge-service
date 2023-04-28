package migrations_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// This migration creates a different proof table dropping all the information.

type migrationTest0004 struct{}

func (m migrationTest0004) InsertData(db *sql.DB) error {
	// Insert initial block, deposit and root
	if _, err := db.Exec("INSERT INTO sync.block (block_num, block_hash, parent_hash, network_id, received_at) VALUES ($1, $2, $3, $4, $5)", 1, common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0,
		time.Now()); err != nil {
		return err
	}
	if _, err := db.Exec("INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		0, 0, 0, common.FromHex("0x0000000000000000000000000000000000000000"),
		"1000000", 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, 0,
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		[]byte{}); err != nil {
		return err
	}
	if _, err := db.Exec("INSERT INTO mt.root (root, deposit_cnt, network, deposit_id) VALUES ($1, $2, $3, $4)",
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 1, 0, 1); err != nil {
		return err
	}

	return nil
}

func (m migrationTest0004) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// Insert a new Deposit
	var depositID uint64
	err := db.QueryRow("INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id",
		0, 0, 0, common.FromHex("0x0000000000000000000000000000000000000000"),
		"1000000", 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, 1,
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		[]byte{}).Scan(&depositID)
	assert.NoError(t, err)
	// Insert a new node to the rht table
	const addNode = "INSERT INTO mt.rht (key, value, deposit_id) VALUES ($1, $2, $3)"
	_, err = db.Exec(addNode, common.FromHex("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"), [][]byte{
		common.FromHex("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		common.FromHex("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
	}, depositID)
	assert.NoError(t, err)

	// Insert a monitored tx
	_, err = db.Exec(`INSERT INTO sync.monitored_txs 
	(id, block_id, from_addr, to_addr, nonce, value, data, gas, status, history, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`, 0, 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, "10000", []byte{}, 5000000, "crerated", nil, time.Now(), time.Now())
	assert.NoError(t, err)
	// Insert a new Deposit
	_, err = db.Exec("INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, ready_for_claim) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)",
		0, 0, 0, common.FromHex("0x0000000000000000000000000000000000000000"),
		"1000000", 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, 2,
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
}

func (m migrationTest0004) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	// Insert a monitored tx
	_, err := db.Exec(`INSERT INTO sync.monitored_txs 
		(id, block_id, from_addr, to_addr, nonce, value, data, gas, status, history, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`, 0, 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, "10000", []byte{}, 5000000, "crerated", nil, time.Now(), time.Now())
	assert.Error(t, err)
}

func TestMigration0004(t *testing.T) {
	runMigrationTest(t, 4, migrationTest0004{})
}
