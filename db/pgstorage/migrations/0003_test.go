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
	if _, err := db.Exec("INSERT INTO mt.root (root, deposit_cnt, network) VALUES ($1, $2, $3)",
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 1, 0); err != nil {
		return err
	}

	// Insert initial nodes to rht table
	const addNode = "INSERT INTO mt.rht (key, value, root_id) VALUES ($1, $2, $3)"
	if _, err := db.Exec(addNode, common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), [][]byte{
		common.FromHex("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		common.FromHex("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
	}, 1); err != nil {
		return err
	}
	if _, err := db.Exec(addNode, common.FromHex("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"), [][]byte{
		common.FromHex("0xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344"),
		common.FromHex("0x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d"),
	}, 1); err != nil {
		return err
	}
	return nil
}

func (m migrationTest0003) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// Insert a new Deposit
	var depositID uint64
	err := db.QueryRow("INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id",
		0, 0, 0, common.FromHex("0x0000000000000000000000000000000000000000"),
		"1000000", 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, 1,
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		[]byte{}).Scan(&depositID)
	assert.NoError(t, err)
	// Insert a new root
	const addRoot = "INSERT INTO mt.root (root, deposit_cnt, network, deposit_id) VALUES ($1, $2, $3, $4)"
	_, err = db.Exec(addRoot, common.FromHex("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"), 1, 0, depositID)
	assert.NoError(t, err)
	// Insert a new node to the rht table
	const addNode = "INSERT INTO mt.rht (key, value, deposit_id) VALUES ($1, $2, $3)"
	_, err = db.Exec(addNode, common.FromHex("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"), [][]byte{
		common.FromHex("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		common.FromHex("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
	}, depositID)
	assert.NoError(t, err)

	// Insert a new Deposit
	_, err = db.Exec("INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		0, 0, 0, common.FromHex("0x0000000000000000000000000000000000000000"),
		"1000000", 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), 1, 2,
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		[]byte{})
	assert.NoError(t, err)
	// Check the root deposit count
	const getRoot = "SELECT MAX(deposit_cnt), COUNT(*) FROM mt.root"
	var maxDepositCnt, rootCount int
	err = db.QueryRow(getRoot).Scan(&maxDepositCnt, &rootCount)
	assert.NoError(t, err)
	assert.Equal(t, maxDepositCnt, 1)
	assert.Equal(t, rootCount, 2)

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
	// Check the root deposit count
	const getRoot = "SELECT MAX(deposit_cnt), COUNT(*) FROM mt.root"
	var maxDepositCnt, rootCount int
	err := db.QueryRow(getRoot).Scan(&maxDepositCnt, &rootCount)
	assert.NoError(t, err)
	assert.Equal(t, maxDepositCnt, 2)
	assert.Equal(t, rootCount, 2)

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
