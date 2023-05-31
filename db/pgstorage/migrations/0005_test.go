package migrations_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// This migration creates a different proof table dropping all the information.

type migrationTest0005 struct{}

func (m migrationTest0005) InsertData(db *sql.DB) error {
	// Insert initial block, batch, verifyBatch and forcedBatch
	if _, err := db.Exec("INSERT INTO sync.block (block_num, block_hash, parent_hash, network_id, received_at) VALUES ($1, $2, $3, $4, $5)", 1, common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
		common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0,
		time.Now()); err != nil {
		return err
	}
	if _, err := db.Exec("INSERT INTO sync.batch (batch_num, sequencer, global_exit_root, timestamp) VALUES ($1, $2, $3, $4)",
		1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"), time.Now()); err != nil {
		return err
	}
	if _, err := db.Exec("INSERT INTO sync.verified_batch (batch_num, aggregator, tx_hash, block_id) VALUES ($1, $2, $3, $4)",
		1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0); err != nil {
		return err
	}
	if _, err := db.Exec("INSERT INTO sync.forced_batch (batch_num, block_id, forced_batch_num, sequencer, global_exit_root, raw_tx_data) VALUES ($1, $2, $3, $4, $5, $6)",
		1, 0, 1, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f56f7a8c96")); err != nil {
		return err
	}

	return nil
}

func (m migrationTest0005) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// Insert batch, verifyBatch and forcedBatch
	_, err := db.Exec("INSERT INTO sync.batch (batch_num, sequencer, global_exit_root, timestamp) VALUES ($1, $2, $3, $4)",
		3, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"), time.Now())
	assert.Error(t, err)
	_, err = db.Exec("INSERT INTO sync.verified_batch (batch_num, aggregator, tx_hash, block_id) VALUES ($1, $2, $3, $4)",
		3, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0)
	assert.Error(t, err)
	_, err = db.Exec("INSERT INTO sync.forced_batch (batch_num, block_id, forced_batch_num, sequencer, global_exit_root, raw_tx_data) VALUES ($1, $2, $3, $4, $5, $6)",
		3, 0, 3, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f56f7a8c96"))
	assert.Error(t, err)
}

func (m migrationTest0005) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	// Insert batch, verifyBatch and forcedBatch
	_, err := db.Exec("INSERT INTO sync.batch (batch_num, sequencer, global_exit_root, timestamp) VALUES ($1, $2, $3, $4)",
		2, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"), time.Now())
	assert.NoError(t, err)
	_, err = db.Exec("INSERT INTO sync.verified_batch (batch_num, aggregator, tx_hash, block_id) VALUES ($1, $2, $3, $4)",
		2, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), 0)
	assert.NoError(t, err)
	_, err = db.Exec("INSERT INTO sync.forced_batch (batch_num, block_id, forced_batch_num, sequencer, global_exit_root, raw_tx_data) VALUES ($1, $2, $3, $4, $5, $6)",
		2, 0, 2, common.FromHex("0x6B175474E89094C44Da98b954EedeAC495271d0F"), common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5"),
		common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f56f7a8c96"))
	assert.NoError(t, err)
}

func TestMigration0005(t *testing.T) {
	runMigrationTest(t, 5, migrationTest0005{})
}
