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
		m.Storage.On("GetLatestL1SyncedExitRoot", ctx, nil).Return(&etherman.GlobalExitRoot{}, gerror.ErrStorageNotFound).Once()
		m.Storage.On("IsLxLyActivated", ctx, nil).Return(true, nil).Once()
		chEvent := make(chan *etherman.GlobalExitRoot)
		chSynced := make(chan uint)
		sync, err := NewSynchronizer(context.Background(), m.Storage, m.BridgeCtrl, m.Etherman, m.ZkEVMClient, genBlockNumber, chEvent, chSynced, cfg)
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
		ethHeader0 := &types.Header{Number: big.NewInt(0), ParentHash: parentHash}
		ethHeader1 := &types.Header{Number: big.NewInt(1), ParentHash: ethHeader0.Hash()}
		ethBlock0 := types.NewBlockWithHeader(ethHeader0)
		ethBlock1 := types.NewBlockWithHeader(ethHeader1)
		lastBlock := &etherman.Block{BlockHash: ethBlock0.Hash(), BlockNumber: ethBlock0.Number().Uint64()}
		var networkID uint = 0

		m.Storage.
			On("GetLastBlock", ctx, networkID, nil).
			Return(lastBlock, nil)

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock.BlockNumber).
			Return(ethBlock0, nil).
			Once()

		var n *big.Int
		m.Etherman.
			On("HeaderByNumber", ctx, n).
			Return(ethHeader1, nil).
			Once()

		globalExitRoot := etherman.GlobalExitRoot{
			BlockID: 1,
			ExitRoots: []common.Hash{
				common.HexToHash("0xc14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58865"),
				common.HexToHash("0xd14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58866"),
			},
			GlobalExitRoot: common.HexToHash("0xb14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58864"),
		}
		ethermanBlock0 := etherman.Block{
			BlockHash:       ethBlock0.Hash(),
			NetworkID:       0,
		}
		ethermanBlock1 := etherman.Block{
			BlockNumber:     ethBlock0.NumberU64(),
			BlockHash:       ethBlock1.Hash(),
			GlobalExitRoots: []etherman.GlobalExitRoot{globalExitRoot},
			NetworkID:       0,
		}
		blocks := []etherman.Block{ethermanBlock0, ethermanBlock1}
		order := map[common.Hash][]etherman.Order{
			ethBlock1.Hash(): {
				{
					Name: etherman.GlobalExitRootsOrder,
					Pos:  0,
				},
			},
		}

		fromBlock := ethBlock0.NumberU64()
		toBlock := fromBlock + cfg.SyncChunkSize
		if toBlock > ethBlock1.NumberU64() {
			toBlock = ethBlock1.NumberU64()
		}
		m.Etherman.
			On("GetRollupInfoByBlockRange", ctx, fromBlock, &toBlock).
			Return(blocks, order, nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		m.Storage.
			On("AddBlock", ctx, &blocks[1], m.DbTx).
			Return(uint64(1), nil).
			Once()

		m.Storage.
			On("AddGlobalExitRoot", ctx, &blocks[1].GlobalExitRoots[0], m.DbTx).
			Return(nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Run(func(args mock.Arguments) { sync.Stop() }).
			Return(nil).
			Once()

		m.Storage.
			On("GetLatestL1SyncedExitRoot", ctx, nil).
			Return(&blocks[1].GlobalExitRoots[0], nil).
			Once()

		g := common.HexToHash("0xb14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58861")

		m.ZkEVMClient.
			On("GetLatestGlobalExitRoot", ctx).
			Return(g, nil).
			Once()

		exitRootResponse := &rpcTypes.ExitRoots{
			MainnetExitRoot: common.HexToHash("0xc14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58862"),
			RollupExitRoot:  common.HexToHash("0xd14c74e4dddf25627a745f46cae6ac98782e2783c3ccc28107c8210e60d58863"),
		}
		m.ZkEVMClient.
			On("ExitRootsByGER", ctx, g).
			Return(exitRootResponse, nil).
			Once()

		ger := &etherman.GlobalExitRoot{
			GlobalExitRoot: g,
			ExitRoots: []common.Hash{
				exitRootResponse.MainnetExitRoot,
				exitRootResponse.RollupExitRoot,
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
