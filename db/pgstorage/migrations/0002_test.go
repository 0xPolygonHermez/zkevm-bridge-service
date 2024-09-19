package migrations_test

import (
	"database/sql"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// This migration creates a different proof table dropping all the information.

type migrationTest0002 struct{}

func (m migrationTest0002) InsertData(db *sql.DB) error {
	// Insert initial nodes to rht table
	const addNode = "INSERT INTO mtv2.rht (key, value) VALUES ($1, $2)"
	if _, err := db.Exec(addNode, common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), [][]byte{
		common.FromHex("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		common.FromHex("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
	}); err != nil {
		return err
	}
	if _, err := db.Exec(addNode, common.FromHex("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"), [][]byte{
		common.FromHex("0xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344"),
		common.FromHex("0x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d"),
	}); err != nil {
		return err
	}
	// Insert initial root
	if _, err := db.Exec("INSERT INTO mtv2.root (root, deposit_cnt, network) VALUES ($1, $2, $3)",
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0, 0); err != nil {
		return err
	}
	return nil
}

func (m migrationTest0002) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// Insert a new root
	const addRoot = "INSERT INTO mt.root (root, deposit_cnt, network) VALUES ($1, $2, $3) RETURNING id"
	var rootID int
	err := db.QueryRow(addRoot, common.FromHex("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"), 1, 0).Scan(&rootID)
	assert.NoError(t, err)

	// Insert a new node to the rht table
	const addNode = "INSERT INTO mt.rht (key, value, root_id) VALUES ($1, $2, $3)"
	_, err = db.Exec(addNode, common.FromHex("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"), [][]byte{
		common.FromHex("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		common.FromHex("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
	}, rootID)
	assert.NoError(t, err)

	indexes := []string{"root_network_idx", "deposit_idx", "block_idx", "root_idx", "exit_roots_idx"}
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

func (m migrationTest0002) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	// Add a root to the the new root table
	const addNewRoot = "INSERT INTO mt.root (root, deposit_cnt, network) VALUES ($1, $2, $3)"
	_, err := db.Exec(addNewRoot, common.FromHex("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"), 1, 0)
	assert.Error(t, err)

	// Insert an old root to reproduce the duplicate root_pkey error
	const addOldRoot = "INSERT INTO mtv2.root (root, deposit_cnt, network) VALUES ($1, $2, $3)"
	_, err = db.Exec(addOldRoot, common.FromHex("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"), 1, 0)
	assert.Error(t, err)

	// Insert an new root to reproduce the duplicate key error
	_, err = db.Exec(addOldRoot, common.FromHex("0xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9"), 2, 0)
	assert.NoError(t, err)

	indexes := []string{"root_network_idx", "deposit_idx", "block_idx", "root_idx", "exit_roots_idx"}
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

func TestMigration0002(t *testing.T) {
	runMigrationTest(t, 2, migrationTest0002{})
}
