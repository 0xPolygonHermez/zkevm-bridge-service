package pgstorage

import (
	"context"
	"testing"
	"time"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLeaves(t *testing.T) {
	data := `INSERT INTO sync.block
	(id, block_num, block_hash, parent_hash, network_id, received_at)
	VALUES(1, 1, decode('5C7831','hex'), decode('5C7830','hex'), 0, '1970-01-01 01:00:00.000');
	INSERT INTO sync.block
	(id, block_num, block_hash, parent_hash, network_id, received_at)
	VALUES(2, 2, decode('5C7832','hex'), decode('5C7831','hex'), 0, '1970-01-01 01:00:00.000');
	
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('A4BFA0908DC7B06D98DA4309F859023D6947561BC19BC00D77F763DEA1A0B9F5','hex'), 1, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25721','hex'), 1);
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('315FEE1AA202BF4A6BD0FDE560C89BE90B6E6E2AAF92DC5E8D118209ABC3410F','hex'), 2, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25721','hex'), 1);
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('B598CE65AA15C08DDA126A2985BA54F0559EAAC562BB43BA430C7344261FBC5D','hex'), 3, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25721','hex'), 1);
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('E6585BDF74B6A46B9EDE8B1B877E1232FB79EE93106C4DB8FFD49CF1685BF242','hex'), 4, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25721','hex'), 1);
	
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('A4BFA0908DC7B06D98DA4309F859023D6947561BC19BC00D77F763DEA1A0B9F6','hex'), 1, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25722','hex'), 2);
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('E6585BDF74B6A46B9EDE8B1B877E1232FB79EE93106C4DB8FFD49CF1685BF243','hex'), 4, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25722','hex'), 2);
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('B598CE65AA15C08DDA126A2985BA54F0559EAAC562BB43BA430C7344261FBC5E','hex'), 3, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25722','hex'), 2);
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('315FEE1AA202BF4A6BD0FDE560C89BE90B6E6E2AAF92DC5E8D118209ABC34110','hex'), 2, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25722','hex'), 1);
	`
	dbCfg := NewConfigFromEnv()
	ctx := context.Background()
	err := InitOrReset(dbCfg)
	require.NoError(t, err)

	store, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)
	_, err = store.Exec(ctx, data)
	require.NoError(t, err)

	leaves, err := store.GetLatestRollupExitLeaves(ctx, nil)
	require.NoError(t, err)

	for _, l := range leaves {
		log.Debugf("leaves: %+v", l)
	}
	assert.Equal(t, "0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f6", leaves[0].Leaf.String())
	assert.Equal(t, uint64(5), leaves[0].ID)
	assert.Equal(t, uint64(2), leaves[0].BlockID)
	assert.Equal(t, uint(1), leaves[0].RollupId)
	assert.Equal(t, "0x42d3339fe8eb57770953423f20a029e778a707e8d58aaf110b40d5eb4dd25722", leaves[0].Root.String())

	assert.Equal(t, "0x315fee1aa202bf4a6bd0fde560c89be90b6e6e2aaf92dc5e8d118209abc34110", leaves[1].Leaf.String())
	assert.Equal(t, uint64(8), leaves[1].ID)
	assert.Equal(t, uint64(1), leaves[1].BlockID)
	assert.Equal(t, uint(2), leaves[1].RollupId)
	assert.Equal(t, "0x42d3339fe8eb57770953423f20a029e778a707e8d58aaf110b40d5eb4dd25722", leaves[1].Root.String())

	assert.Equal(t, "0xb598ce65aa15c08dda126a2985ba54f0559eaac562bb43ba430c7344261fbc5e", leaves[2].Leaf.String())
	assert.Equal(t, uint64(7), leaves[2].ID)
	assert.Equal(t, uint64(2), leaves[2].BlockID)
	assert.Equal(t, uint(3), leaves[2].RollupId)
	assert.Equal(t, "0x42d3339fe8eb57770953423f20a029e778a707e8d58aaf110b40d5eb4dd25722", leaves[2].Root.String())

	assert.Equal(t, "0xe6585bdf74b6a46b9ede8b1b877e1232fb79ee93106c4db8ffd49cf1685bf243", leaves[3].Leaf.String())
	assert.Equal(t, uint64(6), leaves[3].ID)
	assert.Equal(t, uint64(2), leaves[3].BlockID)
	assert.Equal(t, uint(4), leaves[3].RollupId)
	assert.Equal(t, "0x42d3339fe8eb57770953423f20a029e778a707e8d58aaf110b40d5eb4dd25722", leaves[3].Root.String())
}

func TestIsRollupExitRoot(t *testing.T) {
	data := `INSERT INTO sync.block
	(id, block_num, block_hash, parent_hash, network_id, received_at)
	VALUES(1, 1, decode('5C7831','hex'), decode('5C7830','hex'), 0, '1970-01-01 01:00:00.000');
	
	INSERT INTO mt.rollup_exit
	(leaf, rollup_id, root, block_id)
	VALUES(decode('A4BFA0908DC7B06D98DA4309F859023D6947561BC19BC00D77F763DEA1A0B9F5','hex'), 1, decode('42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25721','hex'), 1);
	`
	root := common.HexToHash("0x42D3339FE8EB57770953423F20A029E778A707E8D58AAF110B40D5EB4DD25721")
	dbCfg := NewConfigFromEnv()
	ctx := context.Background()
	err := InitOrReset(dbCfg)
	require.NoError(t, err)

	store, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	exist, err := store.IsRollupExitRoot(ctx, root, nil)
	require.NoError(t, err)
	assert.Equal(t, false, exist)

	_, err = store.Exec(ctx, data)
	require.NoError(t, err)

	exist, err = store.IsRollupExitRoot(ctx, root, nil)
	require.NoError(t, err)

	assert.Equal(t, true, exist)
}

func createStore(t *testing.T) *PostgresStorage {
	dbCfg := NewConfigFromEnv()

	err := InitOrReset(dbCfg)
	require.NoError(t, err)

	store, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)
	return store
}

func TestAddMonitoredTxs(t *testing.T) {
	store := createStore(t)
	ctx := context.Background()
	to := common.HexToAddress("0x123")
	groupID := uint64(1)
	monitoredTx := ctmtypes.MonitoredTx{
		To:        &to,
		Nonce:     1,
		Value:     nil,
		Data:      []byte("data"),
		Gas:       100,
		GasPrice:  nil,
		Status:    ctmtypes.MonitoredTxStatusCreated,
		History:   map[common.Hash]bool{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		GroupID:   &groupID,
	}
	err := store.AddClaimTx(ctx, monitoredTx, nil)
	require.NoError(t, err)
}

func TestAddMonitoredTxsGroup(t *testing.T) {
	store := createStore(t)
	ctx := context.Background()
	group := ctmtypes.MonitoredTxGroupDBEntry{
		GroupID:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.AddMonitoredTxsGroup(ctx, &group, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), group.GroupID)
}
