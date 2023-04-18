package migrations_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// This migration creates a different proof table dropping all the information.

type migrationTest0005 struct{}

func (m migrationTest0005) InsertData(db *sql.DB) error {
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

	// Insert initial nodes to rht table
	const addNode = "INSERT INTO mt.rht (key, value, deposit_id) VALUES ($1, $2, $3)"
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
	if _, err := db.Exec(addNode, common.FromHex("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"), [][]byte{
		common.FromHex("0xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344"),
		common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000"),
	}, 1); err != nil {
		return err
	}

	start := common.FromHex("0xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344")
	for i := 0; i < 30; i++ {
		left := utils.GenerateRandomHash()
		if _, err := db.Exec(addNode, start, [][]byte{
			left[:],
			common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000"),
		}, 1); err != nil {
			return err
		}
		start = left[:]
	}

	return nil
}

func (m migrationTest0005) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// Check the get_nodes function
	const getNodes = "SELECT left_hash, right_hash FROM mt.get_nodes($1, $2);"
	root := common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
	rows, err := db.Query(getNodes, 0, root)
	assert.NoError(t, err)
	defer rows.Close()
	count := 0
	for rows.Next() {
		var left, right []byte
		assert.NoError(t, rows.Scan(&left, &right))
		count++
		if count == 2 {
			assert.Equal(t, common.FromHex("0x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d"), right)
		}
	}

	rows, err = db.Query(getNodes, 1<<31, root)
	assert.NoError(t, err)
	defer rows.Close()
	count = 0
	for rows.Next() {
		var left, right []byte
		assert.NoError(t, rows.Scan(&left, &right))
		count++
		if count == 2 {
			assert.Equal(t, common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000"), right)
		}
	}

	functions := []string{"get_nodes"}
	// Check function adding
	for _, fn := range functions {
		// getProc
		const getProc = `SELECT count(*) FROM pg_proc WHERE proname = $1;`
		row := db.QueryRow(getProc, fn)
		var result int
		assert.NoError(t, row.Scan(&result))
		assert.Equal(t, 1, result)
	}
}

func (m migrationTest0005) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	functions := []string{"get_nodes"}
	// Check function removing
	for _, fn := range functions {
		// getProc
		const getProc = `SELECT count(*) FROM pg_proc WHERE proname = $1;`
		row := db.QueryRow(getProc, fn)
		var result int
		assert.NoError(t, row.Scan(&result))
		assert.Equal(t, 0, result)
	}
}

func TestMigration0005(t *testing.T) {
	runMigrationTest(t, 5, migrationTest0005{})
}
