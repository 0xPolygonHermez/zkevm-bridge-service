package pgstorage

import (
	"context"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGetDepositsXLayer(t *testing.T) {
	data := `INSERT INTO sync.block
	(id, block_num, block_hash, parent_hash, network_id, received_at)
	VALUES(1, 1, decode('5C7831','hex'), decode('5C7830','hex'), 0, '1970-01-01 01:00:00.000');
	INSERT INTO sync.block
	(id, block_num, block_hash, parent_hash, network_id, received_at)
	VALUES(2, 2, decode('5C7832','hex'), decode('5C7831','hex'), 0, '1970-01-01 01:00:00.000');

	INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, dest_contract_addr)
	VALUES (1, 0, 0, '\xca3faf8a0e99b136394286569f95f04127cb2087', 0, 1, '\x23335657622dcc27bb1914e51cdc30871d6d04d3', 1, 1, '\xf6ff1541a10bb49be5a38da488690f9fc8a97021e245ec1bc5190fec8a64909a',
	        '\x00000000000000000000000023335657622dcc27bb1914e51cdc30871d6d04d300000000000000000000000000000000000000000000000000000000000f4240',
	        '\x74b7f16337b8972027f6196a17a631ac6de26d22');
	INSERT INTO sync.deposit (leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata)
	VALUES (0, 0, 0, '\x0000000000000000000000000000000000000000', 100000000, 1, '\x23335657622dcc27bb1914e51cdc30871d6d04d3', 1, 2, '\xf6ff1541a10bb49be5a38da488690f9fc8a97021e245ec1bc5190fec8a64909a',
		'\x00000000000000000000000023335657622dcc27bb1914e51cdc30871d6d04d300000000000000000000000000000000000000000000000000000000000f4240');`

	dbCfg := NewConfigFromEnv()
	ctx := context.Background()
	err := InitOrReset(dbCfg)
	require.NoError(t, err)

	store, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)
	_, err = store.Exec(ctx, data)
	require.NoError(t, err)

	utils.InitRollupNetworkId(1)
	addr := "0x23335657622dcc27bb1914e51cdc30871d6d04d3"

	deposits, err := store.GetDepositsXLayer(ctx, addr, 25, 0, []common.Address{common.HexToAddress("0xca3faf8a0e99b136394286569f95f04127cb2087")}, nil)
	for _, d := range deposits {
		t.Logf("deposit: [%+v]", d)
	}
	require.NoError(t, err)
	require.Len(t, deposits, 2)
}
