package synchronizer

import (
	context "context"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	cfgTypes "github.com/0xPolygonHermez/zkevm-node/config/types"
	rpcTypes "github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mocks struct {
	Etherman    *ethermanMock
	BridgeCtrl  *bridgectrlMock
	Storage     *storageMock
	DbTx        *dbTxMock
	ZkEVMClient *zkEVMClientMock
}

func TestSyncGer(t *testing.T) {
	setupMocks := func(m *mocks) Synchronizer {
		genBlockNumber := uint64(123456)
		cfg := Config{
			SyncInterval:  cfgTypes.Duration{Duration: 1 * time.Second},
			SyncChunkSize: 10,
		}
		ctx := mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil })
		m.Etherman.On("GetNetworkID", ctx).Return(uint(0), nil)
		m.Storage.On("GetLatestL1SyncedExitRoot", context.Background(), nil).Return(&etherman.GlobalExitRoot{}, gerror.ErrStorageNotFound)
		chEvent := make(chan *etherman.GlobalExitRoot)
		chSynced := make(chan uint)
		sync, err := NewSynchronizer(m.Storage, m.BridgeCtrl, m.Etherman, m.ZkEVMClient, genBlockNumber, chEvent, chSynced, cfg)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case <-chEvent:
					t.Log("New GER received")
				case netID := <-chSynced:
					t.Log("Synced networkID: ", netID)
				case <-context.Background().Done():
					return
				}
			}
		}()

		parentHash := common.HexToHash("0x111")
		ethHeader := &types.Header{Number: big.NewInt(1), ParentHash: parentHash}
		ethBlock := types.NewBlockWithHeader(ethHeader)
		lastBlock := &etherman.Block{BlockHash: ethBlock.Hash(), BlockNumber: ethBlock.Number().Uint64()}
		var networkID uint = 0

		m.Storage.
			On("GetLastBlock", ctx, networkID, nil).
			Return(lastBlock, nil).
			Once()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock.BlockNumber).
			Return(ethBlock, nil).
			Once()

		var n *big.Int
		m.Etherman.
			On("HeaderByNumber", ctx, n).
			Return(ethHeader, nil).
			Once()

		globalExitRoot := etherman.GlobalExitRoot{
			BlockID: 1,
			ExitRoots: []common.Hash{
				common.HexToHash("0xc14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58865"),
				common.HexToHash("0xd14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58866"),
			},
			GlobalExitRoot: common.HexToHash("0xb14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58864"),
		}

		ethermanBlock := etherman.Block{
			BlockHash:       ethBlock.Hash(),
			GlobalExitRoots: []etherman.GlobalExitRoot{globalExitRoot},
			NetworkID:       0,
		}
		blocks := []etherman.Block{ethermanBlock}
		order := map[common.Hash][]etherman.Order{
			ethBlock.Hash(): {
				{
					Name: etherman.GlobalExitRootsOrder,
					Pos:  0,
				},
			},
		}

		fromBlock := ethBlock.NumberU64() + 1
		toBlock := fromBlock + cfg.SyncChunkSize

		m.Etherman.
			On("GetRollupInfoByBlockRange", ctx, fromBlock, &toBlock).
			Return(blocks, order, nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		m.Storage.
			On("AddBlock", ctx, &blocks[0], m.DbTx).
			Return(uint64(1), nil).
			Once()

		m.Storage.
			On("AddGlobalExitRoot", ctx, &blocks[0].GlobalExitRoots[0], m.DbTx).
			Return(nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Run(func(args mock.Arguments) { sync.Stop() }).
			Return(nil).
			Once()

		rpcResponse := &rpcTypes.Batch{
			GlobalExitRoot:  common.HexToHash("0xb14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58861"),
			MainnetExitRoot: common.HexToHash("0xc14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58862"),
			RollupExitRoot:  common.HexToHash("0xd14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58863"),
		}
		m.ZkEVMClient.
			On("BatchNumber", ctx).
			Return(uint64(1), nil).
			Once()

		m.ZkEVMClient.
			On("BatchByNumber", ctx, big.NewInt(1)).
			Return(rpcResponse, nil).
			Once()

		ger := &etherman.GlobalExitRoot{
			GlobalExitRoot: rpcResponse.GlobalExitRoot,
			ExitRoots: []common.Hash{
				rpcResponse.MainnetExitRoot,
				rpcResponse.RollupExitRoot,
			},
		}

		m.Storage.
			On("AddTrustedGlobalExitRoot", ctx, ger, nil).
			Return(false, nil).
			Once()

		return sync
	}

	m := mocks{
		Etherman:    newEthermanMock(t),
		BridgeCtrl:  newBridgectrlMock(t),
		Storage:     newStorageMock(t),
		DbTx:        newDbTxMock(t),
		ZkEVMClient: newZkEVMClientMock(t),
	}

	// start synchronizing
	t.Run("Sync Ger test", func(t *testing.T) {
		sync := setupMocks(&m)
		err := sync.Sync()
		require.NoError(t, err)
	})
}
