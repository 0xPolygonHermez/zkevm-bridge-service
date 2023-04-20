package synchronizer

import (
	context "context"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	cfgTypes "github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
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

func TestTrustedStateReorg(t *testing.T) {
	type testCase struct {
		Name            string
		getTrustedBatch func(*mocks, etherman.SequencedBatch) *etherman.Batch
	}

	setupMocks := func(m *mocks, tc *testCase) Synchronizer {
		genBlockNumber := uint64(123456)
		cfg := Config{
			SyncInterval:  cfgTypes.Duration{Duration: 1 * time.Second},
			SyncChunkSize: 10,
		}
		ctx := mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil })
		m.Etherman.On("GetNetworkID", ctx).Return(uint(0), nil)
		m.Storage.On("GetLatestL1SyncedExitRoot", context.Background(), nil).Return(&etherman.GlobalExitRoot{}, gerror.ErrStorageNotFound)
		chEvent := make(chan *etherman.GlobalExitRoot)
		sync, err := NewSynchronizer(m.Storage, m.BridgeCtrl, m.Etherman, m.ZkEVMClient, genBlockNumber, chEvent, cfg)
		require.NoError(t, err)

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

		sequencedBatch := etherman.SequencedBatch{
			BatchNumber: uint64(1),
			Sequencer:   common.HexToAddress("0x222"),
			TxHash:      common.HexToHash("0x333"),
			PolygonZkEVMBatchData: polygonzkevm.PolygonZkEVMBatchData{
				Transactions:       []byte{},
				GlobalExitRoot:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				Timestamp:          uint64(time.Now().Unix()),
				MinForcedTimestamp: 0,
			},
		}

		ethermanBlock := etherman.Block{
			BlockHash:        ethBlock.Hash(),
			SequencedBatches: [][]etherman.SequencedBatch{{sequencedBatch}},
			NetworkID:        0,
		}
		blocks := []etherman.Block{ethermanBlock}
		order := map[common.Hash][]etherman.Order{
			ethBlock.Hash(): {
				{
					Name: etherman.SequenceBatchesOrder,
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

		block := &etherman.Block{
			ID:               0,
			BlockNumber:      ethermanBlock.BlockNumber,
			BlockHash:        ethermanBlock.BlockHash,
			ParentHash:       ethermanBlock.ParentHash,
			NetworkID:        0,
			ReceivedAt:       ethermanBlock.ReceivedAt,
			SequencedBatches: ethermanBlock.SequencedBatches,
		}

		m.Storage.
			On("AddBlock", ctx, block, m.DbTx).
			Return(uint64(1), nil).
			Once()

		trustedBatch := tc.getTrustedBatch(m, sequencedBatch)

		m.Storage.
			On("GetBatchByNumber", ctx, sequencedBatch.BatchNumber, m.DbTx).
			Return(trustedBatch, nil).
			Once()

		m.Storage.
			On("ResetTrustedState", ctx, sequencedBatch.BatchNumber-1, m.DbTx).
			Return(nil).
			Once()

		b := &etherman.Batch{
			BatchNumber:    sequencedBatch.BatchNumber,
			Coinbase:       sequencedBatch.Sequencer,
			BatchL2Data:    sequencedBatch.Transactions,
			Timestamp:      time.Unix(int64(sequencedBatch.Timestamp), 0),
			GlobalExitRoot: sequencedBatch.GlobalExitRoot,
		}

		m.Storage.
			On("AddBatch", ctx, b, m.DbTx).
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

	testCases := []testCase{
		{
			Name: "Transactions are different",
			getTrustedBatch: func(m *mocks, sequencedBatch etherman.SequencedBatch) *etherman.Batch {
				return &etherman.Batch{
					BatchL2Data:    []byte{1},
					GlobalExitRoot: sequencedBatch.GlobalExitRoot,
					Timestamp:      time.Unix(int64(sequencedBatch.Timestamp), 0),
					Coinbase:       sequencedBatch.Sequencer,
				}
			},
		},
		{
			Name: "Global Exit Root is different",
			getTrustedBatch: func(m *mocks, sequencedBatch etherman.SequencedBatch) *etherman.Batch {
				return &etherman.Batch{
					BatchL2Data:    sequencedBatch.Transactions,
					GlobalExitRoot: common.HexToHash("0x999888777"),
					Timestamp:      time.Unix(int64(sequencedBatch.Timestamp), 0),
					Coinbase:       sequencedBatch.Sequencer,
				}
			},
		},
		{
			Name: "Timestamp is different",
			getTrustedBatch: func(m *mocks, sequencedBatch etherman.SequencedBatch) *etherman.Batch {
				return &etherman.Batch{
					BatchL2Data:    sequencedBatch.Transactions,
					GlobalExitRoot: sequencedBatch.GlobalExitRoot,
					Timestamp:      time.Unix(int64(0), 0),
					Coinbase:       sequencedBatch.Sequencer,
				}
			},
		},
		{
			Name: "Coinbase is different",
			getTrustedBatch: func(m *mocks, sequencedBatch etherman.SequencedBatch) *etherman.Batch {
				return &etherman.Batch{
					BatchL2Data:    sequencedBatch.Transactions,
					GlobalExitRoot: sequencedBatch.GlobalExitRoot,
					Timestamp:      time.Unix(int64(sequencedBatch.Timestamp), 0),
					Coinbase:       common.HexToAddress("0x999888777"),
				}
			},
		},
	}

	m := mocks{
		Etherman:    newEthermanMock(t),
		BridgeCtrl:  newBridgectrlMock(t),
		Storage:     newStorageMock(t),
		DbTx:        newDbTxMock(t),
		ZkEVMClient: newZkEVMClientMock(t),
	}

	// start synchronizing
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testCase := tc
			sync := setupMocks(&m, &testCase)
			err := sync.Sync()
			require.NoError(t, err)
		})
	}
}
