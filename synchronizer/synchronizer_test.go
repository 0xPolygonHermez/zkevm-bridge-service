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
		parentCtx := context.Background()
		sync, err := NewSynchronizer(parentCtx, m.Storage, m.BridgeCtrl, m.Etherman, m.ZkEVMClient, genBlockNumber, chEvent, chSynced, cfg)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case <-chEvent:
					t.Log("New GER received")
				case netID := <-chSynced:
					t.Log("Synced networkID: ", netID)
				case <-parentCtx.Done():
					t.Log("Stopping parentCtx...")
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

func TestReorg(t *testing.T) {
	setupMocks := func(m *mocks) Synchronizer {
		genBlockNumber := uint64(0)
		cfg := Config{
			SyncInterval:  cfgTypes.Duration{Duration: 1 * time.Second},
			SyncChunkSize: 10,
		}
		ctx := mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil })
	 	parentContext := context.Background()
		m.Etherman.On("GetNetworkID", ctx).Return(uint(0), nil)
		m.Storage.On("GetLatestL1SyncedExitRoot", ctx, nil).Return(&etherman.GlobalExitRoot{}, gerror.ErrStorageNotFound).Once()
		m.Storage.On("IsLxLyActivated", ctx, nil).Return(true, nil).Once()
		chEvent := make(chan *etherman.GlobalExitRoot)
		chSynced := make(chan uint)
		sync, err := NewSynchronizer(parentContext, m.Storage, m.BridgeCtrl, m.Etherman, m.ZkEVMClient, genBlockNumber, chEvent, chSynced, cfg)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case <-chEvent:
					t.Log("New GER received")
				case netID := <-chSynced:
					t.Log("Synced networkID: ", netID)
				case <-parentContext.Done():
					t.Log("Stopping parentCtx...")
					return
				}
			}
		}()
		parentHash := common.HexToHash("0x111")
		ethHeader0 := &types.Header{Number: big.NewInt(0), ParentHash: parentHash}
		ethBlock0 := types.NewBlockWithHeader(ethHeader0)
		ethHeader1bis := &types.Header{Number: big.NewInt(1), ParentHash: ethBlock0.Hash(), Time: 10, GasUsed: 20, Root: common.HexToHash("0x234")}
		ethBlock1bis := types.NewBlockWithHeader(ethHeader1bis)
		ethHeader2bis := &types.Header{Number: big.NewInt(2), ParentHash: ethBlock1bis.Hash()}
		ethBlock2bis := types.NewBlockWithHeader(ethHeader2bis)
		ethHeader3bis := &types.Header{Number: big.NewInt(3), ParentHash: ethBlock2bis.Hash()}
		ethBlock3bis := types.NewBlockWithHeader(ethHeader3bis)
		ethHeader1 := &types.Header{Number: big.NewInt(1), ParentHash: ethBlock0.Hash()}
		ethBlock1 := types.NewBlockWithHeader(ethHeader1)
		ethHeader2 := &types.Header{Number: big.NewInt(2), ParentHash: ethBlock1.Hash()}
		ethBlock2 := types.NewBlockWithHeader(ethHeader2)
		ethHeader3 := &types.Header{Number: big.NewInt(3), ParentHash: ethBlock2.Hash()}
		ethBlock3 := types.NewBlockWithHeader(ethHeader3)

		lastBlock0 := &etherman.Block{BlockHash: ethBlock0.Hash(), BlockNumber: ethBlock0.Number().Uint64(), ParentHash: ethBlock0.ParentHash()}
		lastBlock1 := &etherman.Block{BlockHash: ethBlock1.Hash(), BlockNumber: ethBlock1.Number().Uint64(), ParentHash: ethBlock1.ParentHash()}
		var networkID uint = 0

		m.Storage.
			On("GetLastBlock", ctx, networkID, nil).
			Return(lastBlock1, nil).
			Once()

		var n *big.Int
		m.Etherman.
			On("HeaderByNumber", ctx, n).
			Return(ethHeader3bis, nil).
			Once()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock1.BlockNumber).
			Return(ethBlock1, nil).
			Once()

		ti := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)

		ethermanBlock1bis := etherman.Block{
			BlockNumber: 1,
			ReceivedAt:  ti,
			BlockHash:   ethBlock1bis.Hash(),
			ParentHash:  ethBlock1bis.ParentHash(),
		}
		ethermanBlock2bis := etherman.Block{
			BlockNumber: 2,
			ReceivedAt:  ti,
			BlockHash:   ethBlock2bis.Hash(),
			ParentHash:  ethBlock2bis.ParentHash(),
		}
		blocks := []etherman.Block{ethermanBlock1bis, ethermanBlock2bis}
		order := map[common.Hash][]etherman.Order{}

		fromBlock := ethBlock1.NumberU64()
		toBlock := fromBlock + cfg.SyncChunkSize
		if toBlock > ethBlock3.NumberU64() {
			toBlock = ethBlock3.NumberU64()
		}
		m.Etherman.
			On("GetRollupInfoByBlockRange", mock.Anything, fromBlock, &toBlock).
			Return(blocks, order, nil).
			Once()

		var depth uint64 = 1
		stateBlock0 := &etherman.Block{
			BlockNumber: ethBlock0.NumberU64(),
			BlockHash:   ethBlock0.Hash(),
			ParentHash:  ethBlock0.ParentHash(),
			ReceivedAt:  ti,
		}
		m.Storage.
			On("GetPreviousBlock", ctx, networkID, depth, nil).
			Return(stateBlock0, nil).
			Once()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock0.BlockNumber).
			Return(ethBlock0, nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		m.Storage.
			On("Reset", ctx, ethBlock0.NumberU64(), networkID, m.DbTx).
			Return(nil).
			Once()

		depositCnt := 1
		m.Storage.
			On("GetNumberDeposits", ctx, networkID, ethBlock0.NumberU64(), m.DbTx).
			Return(uint64(depositCnt), nil).
			Once()

		m.BridgeCtrl.
			On("ReorgMT", ctx, uint(depositCnt), networkID, m.DbTx).
			Return(nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Return(nil).
			Once()

		m.Etherman.
			On("HeaderByNumber", ctx, n).
			Return(ethHeader3bis, nil).
			Twice()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock0.BlockNumber).
			Return(ethBlock0, nil).
			Once()

		ethermanBlock0 := etherman.Block{
			BlockNumber: 0,
			ReceivedAt:  ti,
			BlockHash:   ethBlock0.Hash(),
			ParentHash:  ethBlock0.ParentHash(),
		}
		ethermanBlock3bis := etherman.Block{
			BlockNumber: 3,
			ReceivedAt:  ti,
			BlockHash:   ethBlock3bis.Hash(),
			ParentHash:  ethBlock3bis.ParentHash(),
		}
		fromBlock = 0
		blocks2 := []etherman.Block{ethermanBlock0, ethermanBlock1bis, ethermanBlock2bis, ethermanBlock3bis}
		m.Etherman.
			On("GetRollupInfoByBlockRange", mock.Anything, fromBlock, &toBlock).
			Return(blocks2, order, nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		stateBlock1bis := &etherman.Block{
			BlockNumber: ethermanBlock1bis.BlockNumber,
			BlockHash:   ethermanBlock1bis.BlockHash,
			ParentHash:  ethermanBlock1bis.ParentHash,
			ReceivedAt:  ethermanBlock1bis.ReceivedAt,
		}
		m.Storage.
			On("AddBlock", ctx, stateBlock1bis, m.DbTx).
			Return(uint64(1), nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Return(nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		stateBlock2bis := &etherman.Block{
			BlockNumber: ethermanBlock2bis.BlockNumber,
			BlockHash:   ethermanBlock2bis.BlockHash,
			ParentHash:  ethermanBlock2bis.ParentHash,
			ReceivedAt:  ethermanBlock2bis.ReceivedAt,
		}
		m.Storage.
			On("AddBlock", ctx, stateBlock2bis, m.DbTx).
			Return(uint64(2), nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Return(nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		stateBlock3bis := &etherman.Block{
			BlockNumber: ethermanBlock3bis.BlockNumber,
			BlockHash:   ethermanBlock3bis.BlockHash,
			ParentHash:  ethermanBlock3bis.ParentHash,
			ReceivedAt:  ethermanBlock3bis.ReceivedAt,
		}
		m.Storage.
			On("AddBlock", ctx, stateBlock3bis, m.DbTx).
			Return(uint64(3), nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Return(nil).
			Once()

		ger := common.HexToHash("0x01")
		m.ZkEVMClient.
			On("GetLatestGlobalExitRoot", ctx).
			Return(ger, nil).
			Once()

		exitRoots := &rpcTypes.ExitRoots{
			MainnetExitRoot: common.Hash{},
			RollupExitRoot: common.Hash{},
		}
		m.ZkEVMClient.
			On("ExitRootsByGER", ctx, ger).
			Return(exitRoots, nil).
			Once()

		fullGer := &etherman.GlobalExitRoot{
			GlobalExitRoot: ger,
			ExitRoots: []common.Hash{
				exitRoots.MainnetExitRoot,
				exitRoots.RollupExitRoot,
			},
		}
		m.Storage.
			On("AddTrustedGlobalExitRoot", ctx, fullGer, nil).
			Return(true, nil).
			Run(func(args mock.Arguments) {
				sync.Stop()
			}).
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

func TestLatestSyncedBlockEmpty(t *testing.T) {
	setupMocks := func(m *mocks) Synchronizer {
		genBlockNumber := uint64(0)
		cfg := Config{
			SyncInterval:  cfgTypes.Duration{Duration: 1 * time.Second},
			SyncChunkSize: 10,
		}
		ctx := mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil })
	 	parentContext := context.Background()
		m.Etherman.On("GetNetworkID", ctx).Return(uint(0), nil)
		m.Storage.On("GetLatestL1SyncedExitRoot", ctx, nil).Return(&etherman.GlobalExitRoot{}, gerror.ErrStorageNotFound).Once()
		m.Storage.On("IsLxLyActivated", ctx, nil).Return(true, nil).Once()
		chEvent := make(chan *etherman.GlobalExitRoot)
		chSynced := make(chan uint)
		sync, err := NewSynchronizer(parentContext, m.Storage, m.BridgeCtrl, m.Etherman, m.ZkEVMClient, genBlockNumber, chEvent, chSynced, cfg)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case <-chEvent:
					t.Log("New GER received")
				case netID := <-chSynced:
					t.Log("Synced networkID: ", netID)
				case <-parentContext.Done():
					t.Log("Stopping parentCtx...")
					return
				}
			}
		}()
		parentHash := common.HexToHash("0x111")
		ethHeader0 := &types.Header{Number: big.NewInt(0), ParentHash: parentHash}
		ethBlock0 := types.NewBlockWithHeader(ethHeader0)
		ethHeader1 := &types.Header{Number: big.NewInt(1), ParentHash: ethBlock0.Hash()}
		ethBlock1 := types.NewBlockWithHeader(ethHeader1)
		ethHeader2 := &types.Header{Number: big.NewInt(2), ParentHash: ethBlock1.Hash()}
		ethBlock2 := types.NewBlockWithHeader(ethHeader2)
		ethHeader3 := &types.Header{Number: big.NewInt(3), ParentHash: ethBlock2.Hash()}
		ethBlock3 := types.NewBlockWithHeader(ethHeader3)

		lastBlock0 := &etherman.Block{BlockHash: ethBlock0.Hash(), BlockNumber: ethBlock0.Number().Uint64(), ParentHash: ethBlock0.ParentHash()}
		lastBlock1 := &etherman.Block{BlockHash: ethBlock1.Hash(), BlockNumber: ethBlock1.Number().Uint64(), ParentHash: ethBlock1.ParentHash()}
		var networkID uint = 0

		m.Storage.
			On("GetLastBlock", ctx, networkID, nil).
			Return(lastBlock1, nil).
			Once()

		var n *big.Int
		m.Etherman.
			On("HeaderByNumber", ctx, n).
			Return(ethHeader3, nil).
			Once()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock1.BlockNumber).
			Return(ethBlock1, nil).
			Once()

		blocks := []etherman.Block{}
		order := map[common.Hash][]etherman.Order{}

		fromBlock := ethBlock1.NumberU64()
		toBlock := fromBlock + cfg.SyncChunkSize
		if toBlock > ethBlock3.NumberU64() {
			toBlock = ethBlock3.NumberU64()
		}
		m.Etherman.
			On("GetRollupInfoByBlockRange", ctx, fromBlock, &toBlock).
			Return(blocks, order, nil).
			Once()

		ti := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)
		var depth uint64 = 1
		stateBlock0 := &etherman.Block{
			BlockNumber: ethBlock0.NumberU64(),
			BlockHash:   ethBlock0.Hash(),
			ParentHash:  ethBlock0.ParentHash(),
			ReceivedAt:  ti,
		}
		m.Storage.
			On("GetPreviousBlock", ctx, networkID, depth, nil).
			Return(stateBlock0, nil).
			Once()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock0.BlockNumber).
			Return(ethBlock0, nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		m.Storage.
			On("Reset", ctx, ethBlock0.NumberU64(), networkID, m.DbTx).
			Return(nil).
			Once()

		depositCnt := 1
		m.Storage.
			On("GetNumberDeposits", ctx, networkID, ethBlock0.NumberU64(), m.DbTx).
			Return(uint64(depositCnt), nil).
			Once()

		m.BridgeCtrl.
			On("ReorgMT", ctx, uint(depositCnt), networkID, m.DbTx).
			Return(nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Return(nil).
			Once()

		m.Etherman.
			On("HeaderByNumber", ctx, n).
			Return(ethHeader3, nil).
			Twice()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock0.BlockNumber).
			Return(ethBlock0, nil).
			Once()

		ethermanBlock0 := etherman.Block{
			BlockNumber: 0,
			ReceivedAt:  ti,
			BlockHash:   ethBlock0.Hash(),
			ParentHash:  ethBlock0.ParentHash(),
		}
		blocks = []etherman.Block{ethermanBlock0}
		fromBlock = 0
		m.Etherman.
			On("GetRollupInfoByBlockRange", ctx, fromBlock, &toBlock).
			Return(blocks, order, nil).
			Once()

		ger := common.HexToHash("0x01")
		m.ZkEVMClient.
			On("GetLatestGlobalExitRoot", ctx).
			Return(ger, nil).
			Once()

		exitRoots := &rpcTypes.ExitRoots{
			MainnetExitRoot: common.Hash{},
			RollupExitRoot: common.Hash{},
		}
		m.ZkEVMClient.
			On("ExitRootsByGER", ctx, ger).
			Return(exitRoots, nil).
			Once()

		fullGer := &etherman.GlobalExitRoot{
			GlobalExitRoot: ger,
			ExitRoots: []common.Hash{
				exitRoots.MainnetExitRoot,
				exitRoots.RollupExitRoot,
			},
		}
		m.Storage.
			On("AddTrustedGlobalExitRoot", ctx, fullGer, nil).
			Return(true, nil).
			Run(func(args mock.Arguments) {
				sync.Stop()
			}).
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

func TestRegularReorg(t *testing.T) {
	setupMocks := func(m *mocks) Synchronizer {
		genBlockNumber := uint64(0)
		cfg := Config{
			SyncInterval:  cfgTypes.Duration{Duration: 1 * time.Second},
			SyncChunkSize: 10,
		}
		ctx := mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil })
	 	parentContext := context.Background()
		m.Etherman.On("GetNetworkID", ctx).Return(uint(0), nil)
		m.Storage.On("GetLatestL1SyncedExitRoot", ctx, nil).Return(&etherman.GlobalExitRoot{}, gerror.ErrStorageNotFound).Once()
		m.Storage.On("IsLxLyActivated", ctx, nil).Return(true, nil).Once()
		chEvent := make(chan *etherman.GlobalExitRoot)
		chSynced := make(chan uint)
		sync, err := NewSynchronizer(parentContext, m.Storage, m.BridgeCtrl, m.Etherman, m.ZkEVMClient, genBlockNumber, chEvent, chSynced, cfg)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case <-chEvent:
					t.Log("New GER received")
				case netID := <-chSynced:
					t.Log("Synced networkID: ", netID)
				case <-parentContext.Done():
					t.Log("Stopping parentCtx...")
					return
				}
			}
		}()
		parentHash := common.HexToHash("0x111")
		ethHeader0 := &types.Header{Number: big.NewInt(0), ParentHash: parentHash}
		ethBlock0 := types.NewBlockWithHeader(ethHeader0)
		ethHeader1bis := &types.Header{Number: big.NewInt(1), ParentHash: ethBlock0.Hash(), Time: 10, GasUsed: 20, Root: common.HexToHash("0x234")}
		ethBlock1bis := types.NewBlockWithHeader(ethHeader1bis)
		ethHeader2bis := &types.Header{Number: big.NewInt(2), ParentHash: ethBlock1bis.Hash()}
		ethBlock2bis := types.NewBlockWithHeader(ethHeader2bis)
		ethHeader1 := &types.Header{Number: big.NewInt(1), ParentHash: ethBlock0.Hash()}
		ethBlock1 := types.NewBlockWithHeader(ethHeader1)
		ethHeader2 := &types.Header{Number: big.NewInt(2), ParentHash: ethBlock1.Hash()}
		ethBlock2 := types.NewBlockWithHeader(ethHeader2)

		lastBlock0 := &etherman.Block{BlockHash: ethBlock0.Hash(), BlockNumber: ethBlock0.Number().Uint64(), ParentHash: ethBlock0.ParentHash()}
		lastBlock1 := &etherman.Block{BlockHash: ethBlock1.Hash(), BlockNumber: ethBlock1.Number().Uint64(), ParentHash: ethBlock1.ParentHash()}
		var networkID uint = 0

		m.Storage.
			On("GetLastBlock", ctx, networkID, nil).
			Return(lastBlock1, nil).
			Once()

		var n *big.Int
		m.Etherman.
			On("HeaderByNumber", ctx, n).
			Return(ethHeader2bis, nil).
			Once()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock1.BlockNumber).
			Return(ethBlock1bis, nil).
			Once()


		ti := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)
		var depth uint64 = 1
		stateBlock0 := &etherman.Block{
			BlockNumber: ethBlock0.NumberU64(),
			BlockHash:   ethBlock0.Hash(),
			ParentHash:  ethBlock0.ParentHash(),
			ReceivedAt:  ti,
		}

		m.Storage.
			On("GetPreviousBlock", ctx, networkID, depth, nil).
			Return(stateBlock0, nil).
			Once()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock0.BlockNumber).
			Return(ethBlock0, nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		m.Storage.
			On("Reset", ctx, ethBlock0.NumberU64(), networkID, m.DbTx).
			Return(nil).
			Once()

		depositCnt := 1
		m.Storage.
			On("GetNumberDeposits", ctx, networkID, ethBlock0.NumberU64(), m.DbTx).
			Return(uint64(depositCnt), nil).
			Once()

		m.BridgeCtrl.
			On("ReorgMT", ctx, uint(depositCnt), networkID, m.DbTx).
			Return(nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Return(nil).
			Once()

		m.Etherman.
			On("HeaderByNumber", ctx, n).
			Return(ethHeader2bis, nil).
			Twice()

		m.Etherman.
			On("EthBlockByNumber", ctx, lastBlock0.BlockNumber).
			Return(ethBlock0, nil).
			Once()

		ethermanBlock0 := etherman.Block{
			BlockNumber: 0,
			ReceivedAt:  ti,
			BlockHash:   ethBlock0.Hash(),
			ParentHash:  ethBlock0.ParentHash(),
		}
		ethermanBlock1bis := etherman.Block{
			BlockNumber: 1,
			ReceivedAt:  ti,
			BlockHash:   ethBlock1bis.Hash(),
			ParentHash:  ethBlock1bis.ParentHash(),
		}
		ethermanBlock2bis := etherman.Block{
			BlockNumber: 2,
			ReceivedAt:  ti,
			BlockHash:   ethBlock2bis.Hash(),
			ParentHash:  ethBlock2bis.ParentHash(),
		}
		blocks := []etherman.Block{ethermanBlock0, ethermanBlock1bis, ethermanBlock2bis}
		order := map[common.Hash][]etherman.Order{}

		fromBlock := ethBlock0.NumberU64()
		toBlock := fromBlock + cfg.SyncChunkSize
		if toBlock > ethBlock2.NumberU64() {
			toBlock = ethBlock2.NumberU64()
		}
		m.Etherman.
			On("GetRollupInfoByBlockRange", ctx, fromBlock, &toBlock).
			Return(blocks, order, nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		stateBlock1bis := &etherman.Block{
			BlockNumber: ethermanBlock1bis.BlockNumber,
			BlockHash:   ethermanBlock1bis.BlockHash,
			ParentHash:  ethermanBlock1bis.ParentHash,
			ReceivedAt:  ethermanBlock1bis.ReceivedAt,
		}
		m.Storage.
			On("AddBlock", ctx, stateBlock1bis, m.DbTx).
			Return(uint64(1), nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Return(nil).
			Once()

		m.Storage.
			On("BeginDBTransaction", ctx).
			Return(m.DbTx, nil).
			Once()

		stateBlock2bis := &etherman.Block{
			BlockNumber: ethermanBlock2bis.BlockNumber,
			BlockHash:   ethermanBlock2bis.BlockHash,
			ParentHash:  ethermanBlock2bis.ParentHash,
			ReceivedAt:  ethermanBlock2bis.ReceivedAt,
		}
		m.Storage.
			On("AddBlock", ctx, stateBlock2bis, m.DbTx).
			Return(uint64(2), nil).
			Once()

		m.Storage.
			On("Commit", ctx, m.DbTx).
			Return(nil).
			Once()

		ger := common.HexToHash("0x01")
		m.ZkEVMClient.
			On("GetLatestGlobalExitRoot", ctx).
			Return(ger, nil).
			Once()

		exitRoots := &rpcTypes.ExitRoots{
			MainnetExitRoot: common.Hash{},
			RollupExitRoot: common.Hash{},
		}
		m.ZkEVMClient.
			On("ExitRootsByGER", ctx, ger).
			Return(exitRoots, nil).
			Once()

		fullGer := &etherman.GlobalExitRoot{
			GlobalExitRoot: ger,
			ExitRoots: []common.Hash{
				exitRoots.MainnetExitRoot,
				exitRoots.RollupExitRoot,
			},
		}
		m.Storage.
			On("AddTrustedGlobalExitRoot", ctx, fullGer, nil).
			Return(true, nil).
			Run(func(args mock.Arguments) {
				sync.Stop()
			}).
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