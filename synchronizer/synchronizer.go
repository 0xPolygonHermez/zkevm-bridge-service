package synchronizer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/sequencer/broadcast/pb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Synchronizer connects L1 and L2
type Synchronizer interface {
	Sync() error
	Stop()
}

// ClientSynchronizer connects L1 and L2
type ClientSynchronizer struct {
	etherMan        ethermanInterface
	bridgeCtrl      bridgectrlInterface
	storage         storageInterface
	ctx             context.Context
	cancelCtx       context.CancelFunc
	genBlockNumber  uint64
	cfg             Config
	networkID       uint
	chExitRootEvent chan bool
	broadcastClient pb.BroadcastServiceClient
	synced          bool
}

// NewSynchronizer creates and initializes an instance of Synchronizer
func NewSynchronizer(
	storage interface{},
	bridge bridgectrlInterface,
	ethMan ethermanInterface,
	broadcastClient pb.BroadcastServiceClient,
	genBlockNumber uint64,
	chExitRootEvent chan bool,
	cfg Config) (Synchronizer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	networkID, err := ethMan.GetNetworkID(ctx)
	if err != nil {
		log.Fatal("error getting networkID. Error: ", err)
	}

	if networkID == 0 {
		return &ClientSynchronizer{
			bridgeCtrl:      bridge,
			storage:         storage.(storageInterface),
			etherMan:        ethMan,
			ctx:             ctx,
			cancelCtx:       cancel,
			genBlockNumber:  genBlockNumber,
			cfg:             cfg,
			networkID:       networkID,
			chExitRootEvent: chExitRootEvent,
			broadcastClient: broadcastClient,
		}, nil
	}
	return &ClientSynchronizer{
		bridgeCtrl:     bridge,
		storage:        storage.(storageInterface),
		etherMan:       ethMan,
		ctx:            ctx,
		cancelCtx:      cancel,
		genBlockNumber: genBlockNumber,
		cfg:            cfg,
		networkID:      networkID,
	}, nil
}

var waitDuration = time.Duration(0)

// Sync function will read the last state synced and will continue from that point.
// Sync() will read blockchain events to detect rollup updates
func (s *ClientSynchronizer) Sync() error {
	// If there is no lastEthereumBlock means that sync from the beginning is necessary. If not, it continues from the retrieved ethereum block
	// Get the latest synced block. If there is no block on db, use genesis block
	log.Infof("NetworkID: %d, Synchronization started", s.networkID)
	dbTx, err := s.storage.BeginDBTransaction(s.ctx)
	if err != nil {
		log.Fatalf("networkID: %d, error creating db transaction to get latest block", s.networkID)
	}
	lastBlockSynced, err := s.storage.GetLastBlock(s.ctx, s.networkID, dbTx)
	if err != nil {
		if err == gerror.ErrStorageNotFound {
			log.Warnf("networkID: %d, error getting the latest ethereum block. No data stored. Setting genesis block. Error: %w", s.networkID, err)
			lastBlockSynced = &etherman.Block{
				BlockNumber: s.genBlockNumber,
				NetworkID:   s.networkID,
			}
			log.Warnf("networkID: %d, error getting the latest block. No data stored. Setting genesis block: %+v. Error: %s",
				s.networkID, lastBlockSynced, err.Error())
		} else {
			log.Fatalf("networkID: %d, unexpected error getting the latest block. Error: %s", s.networkID, err.Error())
		}
	}
	err = s.storage.Commit(s.ctx, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error committing dbTx, err: %s", s.networkID, err.Error())
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state. RollbackErr: %s, err: %s",
				s.networkID, rollbackErr.Error(), err.Error())
		}
		log.Fatalf("networkID: %d, error committing dbTx, err: %s", s.networkID, err.Error())
	}
	for {
		select {
		case <-s.ctx.Done():
			log.Debug("synchronizer ctx done. NetworkID: ", s.networkID)
			return nil
		case <-time.After(waitDuration):
			//Sync L1Blocks
			if lastBlockSynced, err = s.syncBlocks(lastBlockSynced); err != nil {
				log.Warn("error syncing blocks: ", err)
				if s.ctx.Err() != nil {
					continue
				}
			}
			if !s.synced {
				// Check latest Block
				header, err := s.etherMan.HeaderByNumber(s.ctx, nil)
				if err != nil {
					log.Warnf("networkID: %d, error getting latest block from. Error: %s", s.networkID, err.Error())
					continue
				}
				lastKnownBlock := header.Number
				if lastBlockSynced.BlockNumber == lastKnownBlock.Uint64() {
					waitDuration = s.cfg.SyncInterval.Duration
					s.synced = true
				}
				if lastBlockSynced.BlockNumber > lastKnownBlock.Uint64() {
					log.Fatalf("networkID: %d, error: latest Synced BlockNumber is higher than the latest Proposed in the network", s.networkID)
				}
			} else { // Sync Trusted GlobalExitRoots if L1 is synced
				if s.networkID != 0 {
					continue
				}
				log.Infof("networkID: %d, Virtual state is synced, getting trusted state", s.networkID)
				err = s.syncTrustedState()
				if err != nil {
					log.Errorf("networkID: %d, error getting current trusted state", s.networkID)
				}
			}
		}
	}
}

// Stop function stops the synchronizer
func (s *ClientSynchronizer) Stop() {
	s.cancelCtx()
}

func (s *ClientSynchronizer) syncTrustedState() error {
	lastBatch, err := s.broadcastClient.GetLastBatch(s.ctx, &emptypb.Empty{})
	if err != nil {
		log.Errorf("networkID: %d, error getting latest batch from grpc. Error: %w", s.networkID, err)
		return err
	}
	ger := &etherman.GlobalExitRoot{
		GlobalExitRoot: common.HexToHash(lastBatch.GlobalExitRoot),
		ExitRoots: []common.Hash{
			common.HexToHash(lastBatch.MainnetExitRoot),
			common.HexToHash(lastBatch.RollupExitRoot),
		},
	}
	isUpdated, err := s.storage.AddTrustedGlobalExitRoot(s.ctx, ger, nil)
	if err != nil {
		log.Error("networkID: %d, error storing latest trusted globalExitRoot. Error: %w", s.networkID, err)
		return err
	}
	if isUpdated {
		s.chExitRootEvent <- true
	}
	return nil
}

// This function syncs the node from a specific block to the latest
func (s *ClientSynchronizer) syncBlocks(lastBlockSynced *etherman.Block) (*etherman.Block, error) {
	log.Debugf("NetworkID: %d, before checkReorg. lastBlockSynced: %+v", s.networkID, lastBlockSynced)
	// This function will read events fromBlockNum to latestEthBlock. Check reorg to be sure that everything is ok.
	block, err := s.checkReorg(lastBlockSynced)
	if err != nil {
		log.Errorf("networkID: %d, error checking reorgs. Retrying... Err: %s", s.networkID, err.Error())
		return lastBlockSynced, fmt.Errorf("networkID: %d, error checking reorgs", s.networkID)
	}
	if block != nil {
		err = s.resetState(block.BlockNumber)
		if err != nil {
			log.Errorf("networkID: %d, error resetting the state to a previous block. Retrying... Error: %s", s.networkID, err.Error())
			return lastBlockSynced, fmt.Errorf("networkID: %d, error resetting the state to a previous block", s.networkID)
		}
		return block, nil
	}
	log.Debugf("NetworkID: %d, after checkReorg: no reorg detected", s.networkID)
	// Call the blockchain to retrieve data
	header, err := s.etherMan.HeaderByNumber(s.ctx, nil)
	if err != nil {
		return lastBlockSynced, err
	}
	lastKnownBlock := header.Number

	var fromBlock uint64
	if lastBlockSynced.BlockNumber > 0 {
		fromBlock = lastBlockSynced.BlockNumber + 1
	}

	for {
		toBlock := fromBlock + s.cfg.SyncChunkSize

		log.Debugf("NetworkID: %d, Getting bridge info from block %d to block %d", s.networkID, fromBlock, toBlock)
		// This function returns the rollup information contained in the ethereum blocks and an extra param called order.
		// Order param is a map that contains the event order to allow the synchronizer store the info in the same order that is readed.
		// Name can be defferent in the order struct. For instance: Batches or Name:NewSequencers. This name is an identifier to check
		// if the next info that must be stored in the db is a new sequencer or a batch. The value pos (position) tells what is the
		// array index where this value is.
		blocks, order, err := s.etherMan.GetRollupInfoByBlockRange(s.ctx, fromBlock, &toBlock)
		if err != nil {
			return lastBlockSynced, err
		}
		s.processBlockRange(blocks, order)
		if len(blocks) > 0 {
			lastBlockSynced = &blocks[len(blocks)-1]
			for i := range blocks {
				log.Debug("NetworkID: ", s.networkID, ", Position: ", i, ". BlockNumber: ", blocks[i].BlockNumber, ". BlockHash: ", blocks[i].BlockHash)
			}
		}
		fromBlock = toBlock + 1

		if lastKnownBlock.Cmp(new(big.Int).SetUint64(toBlock)) < 1 {
			waitDuration = s.cfg.SyncInterval.Duration
			s.synced = true
			break
		}
		if len(blocks) == 0 { // If there is no events in the checked blocks range and lastKnownBlock > fromBlock.
			// Store the latest block of the block range. Get block info and process the block
			fb, err := s.etherMan.EthBlockByNumber(s.ctx, toBlock)
			if err != nil {
				return lastBlockSynced, err
			}
			b := etherman.Block{
				BlockNumber: fb.NumberU64(),
				BlockHash:   fb.Hash(),
				ParentHash:  fb.ParentHash(),
				ReceivedAt:  time.Unix(int64(fb.Time()), 0),
			}
			s.processBlockRange([]etherman.Block{b}, order)

			lastBlockSynced = &b
			log.Debugf("NetworkID: %d, Storing empty block. BlockNumber: %d. BlockHash: %s",
				s.networkID, b.BlockNumber, b.BlockHash.String())
		}
	}

	return lastBlockSynced, nil
}

func (s *ClientSynchronizer) processBlockRange(blocks []etherman.Block, order map[common.Hash][]etherman.Order) {
	// New info has to be included into the db using the state
	for i := range blocks {
		// Begin db transaction
		dbTx, err := s.storage.BeginDBTransaction(s.ctx)
		if err != nil {
			log.Fatalf("networkID: %d, error creating db transaction to store block. BlockNumber: %d",
				s.networkID, blocks[i].BlockNumber)
		}
		// Add block information
		blocks[i].NetworkID = s.networkID
		blockID, err := s.storage.AddBlock(s.ctx, &blocks[i], dbTx)
		if err != nil {
			rollbackErr := s.storage.Rollback(s.ctx, dbTx)
			if rollbackErr != nil {
				log.Fatalf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %s, err: %s",
					s.networkID, blocks[i].BlockNumber, rollbackErr.Error(), err.Error())
			}
			log.Fatalf("networkID: %d, error storing block. BlockNumber: %d, error: %s",
				s.networkID, blocks[i].BlockNumber, err.Error())
		}
		for _, element := range order[blocks[i].BlockHash] {
			switch element.Name {
			case etherman.SequenceBatchesOrder:
				s.processSequenceBatches(blocks[i].SequencedBatches[element.Pos], blockID, blocks[i].BlockNumber, dbTx)
			case etherman.ForcedBatchesOrder:
				s.processForcedBatch(blocks[i].ForcedBatches[element.Pos], blockID, dbTx)
			case etherman.GlobalExitRootsOrder:
				s.processGlobalExitRoot(blocks[i].GlobalExitRoots[element.Pos], blockID, dbTx)
			case etherman.SequenceForceBatchesOrder:
				s.processSequenceForceBatches(blocks[i].SequencedForceBatches[element.Pos], blocks[i], dbTx)
			case etherman.TrustedVerifyBatchOrder:
				s.processTrustedVerifyBatch(blocks[i].VerifiedBatches[element.Pos], blockID, blocks[i].BlockNumber, dbTx)
			case etherman.DepositsOrder:
				s.processDeposit(blocks[i].Deposits[element.Pos], blockID, dbTx)
			case etherman.ClaimsOrder:
				s.processClaim(blocks[i].Claims[element.Pos], blockID, dbTx)
			case etherman.TokensOrder:
				s.processTokenWrapped(blocks[i].Tokens[element.Pos], blockID, dbTx)
			}
		}
		err = s.storage.Commit(s.ctx, dbTx)
		if err != nil {
			log.Errorf("networkID: %d, error committing state to store block. BlockNumber: %d, err: %s",
				s.networkID, blocks[i].BlockNumber, err.Error())
			rollbackErr := s.storage.Rollback(s.ctx, dbTx)
			if rollbackErr != nil {
				log.Fatalf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %s, err: %s",
					s.networkID, blocks[i].BlockNumber, rollbackErr.Error(), err.Error())
			}
			log.Fatalf("networkID: %d, error committing state to store block. BlockNumber: %d, err: %s",
				s.networkID, blocks[i].BlockNumber, err.Error())
		}
	}
}

// This function allows reset the state until an specific ethereum block
func (s *ClientSynchronizer) resetState(blockNumber uint64) error {
	log.Debugf("NetworkID: %d. Reverting synchronization to block: %d", s.networkID, blockNumber)
	dbTx, err := s.storage.BeginDBTransaction(s.ctx)
	if err != nil {
		log.Errorf("networkID: %d, Error starting a db transaction to reset the state. Error: %s", s.networkID, err.Error())
		return err
	}
	err = s.storage.Reset(s.ctx, blockNumber, s.networkID, dbTx)
	if err != nil {
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %s, error : %s",
				s.networkID, blockNumber, rollbackErr.Error(), err.Error())
			return rollbackErr
		}
		log.Errorf("networkID: %d, error resetting the state. Error: %s", s.networkID, err.Error())
		return err
	}
	depositCnt, err := s.storage.GetNumberDeposits(s.ctx, s.networkID, blockNumber, dbTx)
	if err != nil {
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %s, error : %s",
				s.networkID, blockNumber, rollbackErr.Error(), err.Error())
			return rollbackErr
		}
		log.Error("networkID: %d, error getting GetNumberDeposits. Error: %s", s.networkID, err.Error())
		return err
	}

	err = s.bridgeCtrl.ReorgMT(uint(depositCnt), s.networkID, dbTx)
	if err != nil {
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %s, error : %s",
				s.networkID, blockNumber, rollbackErr.Error(), err.Error())
			return rollbackErr
		}
		log.Error("networkID: %s, error resetting ReorgMT the state. Error: %s", s.networkID, err.Error())
		return err
	}
	err = s.storage.Commit(s.ctx, dbTx)
	if err != nil {
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %s, error : %s",
				s.networkID, blockNumber, rollbackErr.Error(), err.Error())
			return rollbackErr
		}
		log.Errorf("networkID: %d, error committing the resetted state. Error: %s", s.networkID, err.Error())
		return err
	}

	return nil
}

/*
This function will check if there is a reorg.
As input param needs the last ethereum block synced. Retrieve the block info from the blockchain
to compare it with the stored info. If hash and hash parent matches, then no reorg is detected and return a nil.
If hash or hash parent don't match, reorg detected and the function will return the block until the sync process
must be reverted. Then, check the previous ethereum block synced, get block info from the blockchain and check
hash and has parent. This operation has to be done until a match is found.
*/
func (s *ClientSynchronizer) checkReorg(latestBlock *etherman.Block) (*etherman.Block, error) {
	// This function only needs to worry about reorgs if some of the reorganized blocks contained rollup info.
	latestBlockSynced := *latestBlock
	var depth uint64
	for {
		block, err := s.etherMan.EthBlockByNumber(s.ctx, latestBlock.BlockNumber)
		if err != nil {
			log.Errorf("networkID: %d, error getting latest block synced from blockchain. Block: %d, error: %s",
				s.networkID, latestBlock.BlockNumber, err.Error())
			return nil, err
		}
		if block.NumberU64() != latestBlock.BlockNumber {
			err = fmt.Errorf("networkID: %d, wrong ethereum block retrieved from blockchain. Block numbers don't match."+
				" BlockNumber stored: %d. BlockNumber retrieved: %d", s.networkID, latestBlock.BlockNumber, block.NumberU64())
			log.Error("error: ", err)
			return nil, err
		}
		// Compare hashes
		if (block.Hash() != latestBlock.BlockHash || block.ParentHash() != latestBlock.ParentHash) && latestBlock.BlockNumber > s.genBlockNumber {
			log.Debug("NetworkID: ", s.networkID, ", [checkReorg function] => latestBlockNumber: ", latestBlock.BlockNumber)
			log.Debug("NetworkID: ", s.networkID, ", [checkReorg function] => latestBlockHash: ", latestBlock.BlockHash)
			log.Debug("NetworkID: ", s.networkID, ", [checkReorg function] => latestBlockHashParent: ", latestBlock.ParentHash)
			log.Debug("NetworkID: ", s.networkID, ", [checkReorg function] => BlockNumber: ", latestBlock.BlockNumber, block.NumberU64())
			log.Debug("NetworkID: ", s.networkID, ", [checkReorg function] => BlockHash: ", block.Hash())
			log.Debug("NetworkID: ", s.networkID, ", [checkReorg function] => BlockHashParent: ", block.ParentHash())
			depth++
			log.Debug("NetworkID: ", s.networkID, ", REORG: Looking for the latest correct block. Depth: ", depth)
			// Reorg detected. Getting previous block
			dbTx, err := s.storage.BeginDBTransaction(s.ctx)
			if err != nil {
				log.Fatalf("networkID: %d, error creating db transaction to get previous blocks", s.networkID)
			}
			latestBlock, err = s.storage.GetPreviousBlock(s.ctx, s.networkID, depth, dbTx)
			errC := s.storage.Commit(s.ctx, dbTx)
			if errC != nil {
				log.Errorf("networkID: %d, error committing dbTx, err: %s", s.networkID, errC.Error())
				rollbackErr := s.storage.Rollback(s.ctx, dbTx)
				if rollbackErr != nil {
					log.Fatalf("networkID: %d, error rolling back state. RollbackErr: %s, err: %s",
						s.networkID, rollbackErr.Error(), errC.Error())
				}
				log.Fatalf("networkID: %d, error committing dbTx, err: %s", s.networkID, errC.Error())
			}
			if errors.Is(err, gerror.ErrStorageNotFound) {
				log.Warnf("networkID: %d, error checking reorg: previous block not found in db: %s", s.networkID, err.Error())
				return &etherman.Block{}, nil
			} else if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	if latestBlockSynced.BlockHash != latestBlock.BlockHash {
		log.Debugf("NetworkID: %d, reorg detected in block: %d", s.networkID, latestBlockSynced.BlockNumber)
		return latestBlock, nil
	}
	log.Debugf("NetworkID: %d, no reorg detected", s.networkID)
	return nil, nil
}

func (s *ClientSynchronizer) checkTrustedState(batch etherman.Batch, dbTx pgx.Tx) (bool, error) {
	// First get trusted batch from db
	tBatch, err := s.storage.GetBatchByNumber(s.ctx, batch.BatchNumber, dbTx)
	if err != nil {
		return false, err
	}
	//Compare virtual state with trusted state
	if hex.EncodeToString(batch.BatchL2Data) == hex.EncodeToString(tBatch.BatchL2Data) &&
		batch.GlobalExitRoot == tBatch.GlobalExitRoot &&
		batch.Timestamp == tBatch.Timestamp &&
		batch.Coinbase == tBatch.Coinbase {
		return true, nil
	}
	return false, nil
}

func (s *ClientSynchronizer) processSequenceBatches(sequencedBatches []etherman.SequencedBatch, blockID, blockNumber uint64, dbTx pgx.Tx) {
	for _, sbatch := range sequencedBatches {
		batch := etherman.Batch{
			BatchNumber:    sbatch.BatchNumber,
			GlobalExitRoot: sbatch.GlobalExitRoot,
			Timestamp:      time.Unix(int64(sbatch.Timestamp), 0),
			Coinbase:       sbatch.Sequencer,
			BatchL2Data:    sbatch.Transactions,
		}
		// ForcedBatches must be processed
		if sbatch.MinForcedTimestamp > 0 {
			// Read forcedBatches from db
			forcedBatches, err := s.storage.GetNextForcedBatches(s.ctx, 1, dbTx)
			if err != nil {
				log.Errorf("networkID: %d, error getting forcedBatches. BatchNumber: %d", s.networkID, batch.BatchNumber)
				rollbackErr := s.storage.Rollback(s.ctx, dbTx)
				if rollbackErr != nil {
					log.Fatalf("NetworkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %s, error: %w",
						s.networkID, batch.BatchNumber, blockNumber, rollbackErr.Error(), err)
				}
				log.Fatalf("networkID: %d, error getting forcedBatches. BatchNumber: %d, BlockNumber: %d, error: %w",
					s.networkID, batch.BatchNumber, blockNumber, err)
			}
			if len(forcedBatches) == 0 {
				log.Errorf("networkID: %d, error: empty forcedBatches array read from db. BatchNumber: %d", s.networkID, batch.BatchNumber)
				rollbackErr := dbTx.Rollback(s.ctx)
				if rollbackErr != nil {
					log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %s, error: %w", s.networkID, batch.BatchNumber, blockNumber, rollbackErr.Error(), err)
				}
				log.Fatal("networkID: %d, error: empty forcedBatches array read from db. BatchNumber: %d", s.networkID, batch.BatchNumber)
			}
			if uint64(forcedBatches[0].ForcedAt.Unix()) != sbatch.MinForcedTimestamp ||
				forcedBatches[0].GlobalExitRoot != sbatch.GlobalExitRoot ||
				common.Bytes2Hex(forcedBatches[0].RawTxsData) != common.Bytes2Hex(sbatch.Transactions) {
				log.Errorf("networkID: %d, error: forcedBatch received doesn't match with the next expected forcedBatch stored in db. Expected: %+v, Synced: %+v", s.networkID, forcedBatches, sbatch)
				rollbackErr := dbTx.Rollback(s.ctx)
				if rollbackErr != nil {
					log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %w", s.networkID, sbatch.BatchNumber, blockNumber, rollbackErr)
				}
				log.Fatalf("networkID: %d, error: forcedBatch received doesn't match with the next expected forcedBatch stored in db. Expected: %+v, Synced: %+v", s.networkID, forcedBatches, sbatch)
			}

			// Store batchNumber in forced_batch table
			err = s.storage.AddBatchNumberInForcedBatch(s.ctx, forcedBatches[0].ForcedBatchNumber, batch.BatchNumber, dbTx)
			if err != nil {
				log.Errorf("networkID: %d, error adding the batchNumber to forcedBatch in processSequenceBatches. BlockNumber: %d",
					s.networkID, blockNumber)
				rollbackErr := s.storage.Rollback(s.ctx, dbTx)
				if rollbackErr != nil {
					log.Fatalf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %s, error : %w",
						s.networkID, blockNumber, rollbackErr.Error(), err)
				}
				log.Fatalf("networkID: %d, error adding the batchNumber to forcedBatch in processSequenceBatches. BlockNumber: %d, error: %w",
					s.networkID, blockNumber, err)
			}
		}

		// Call the check trusted state method to compare trusted and virtual state
		status, err := s.checkTrustedState(batch, dbTx)
		if err != nil {
			if errors.Is(err, gerror.ErrStorageNotFound) {
				log.Debugf("NetworkID: %d, BatchNumber: %d, not found in trusted state. Storing it...", s.networkID, batch.BatchNumber)
				// If it is not found, store batch
				err = s.storage.AddBatch(s.ctx, &batch, dbTx)
				if err != nil {
					log.Errorf("networkID: %d, error storing batch. BatchNumber: %d, BlockNumber: %d, error: %w",
						s.networkID, batch.BatchNumber, blockNumber, err)
					rollbackErr := s.storage.Rollback(s.ctx, dbTx)
					if rollbackErr != nil {
						log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %s, error: %w",
							s.networkID, batch.BatchNumber, blockNumber, rollbackErr.Error(), err)
					}
					log.Fatalf("networkID: %d, error storing batch. BatchNumber: %d, BlockNumber: %d, error: %w",
						s.networkID, batch.BatchNumber, blockNumber, err)
				}
				status = true
			} else {
				rollbackErr := s.storage.Rollback(s.ctx, dbTx)
				if rollbackErr != nil {
					log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %s, error: %w",
						s.networkID, batch.BatchNumber, blockNumber, rollbackErr.Error(), err)
				}
				log.Fatalf("networkID: %d, error checking trusted state. Error: %w", s.networkID, err)
			}
		}
		if !status {
			// Reset trusted state
			log.Debugf("NetworkID: %d, BatchNumber: %d, trusted state reorg detected. Reseting it...", s.networkID, batch.BatchNumber)
			previousBatchNumber := batch.BatchNumber - 1
			err := s.storage.ResetTrustedState(s.ctx, previousBatchNumber, dbTx) // This method has to reset the forced batches deleting the batchNumber for higher batchNumbers
			if err != nil {
				log.Errorf("networkID: %d, error resetting trusted state. BatchNumber: %d, BlockNumber: %d, error: %w",
					s.networkID, batch.BatchNumber, blockNumber, err)
				rollbackErr := s.storage.Rollback(s.ctx, dbTx)
				if rollbackErr != nil {
					log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %s, error: %w",
						s.networkID, batch.BatchNumber, blockNumber, rollbackErr.Error(), err)
				}
				log.Fatalf("networkID: %d, error resetting trusted state. BatchNumber: %d, BlockNumber: %d, error: %w",
					s.networkID, batch.BatchNumber, blockNumber, err)
			}
			err = s.storage.AddBatch(s.ctx, &batch, dbTx)
			if err != nil {
				log.Errorf("networkID: %d, error storing batch. BatchNumber: %d, BlockNumber: %d, error: %w",
					s.networkID, batch.BatchNumber, blockNumber, err)
				rollbackErr := s.storage.Rollback(s.ctx, dbTx)
				if rollbackErr != nil {
					log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %s, error: %w",
						s.networkID, batch.BatchNumber, blockNumber, rollbackErr.Error(), err)
				}
				log.Fatalf("networkID: %d, error storing batch. BatchNumber: %d, BlockNumber: %d, error: %w",
					s.networkID, batch.BatchNumber, blockNumber, err)
			}
		}
	}
}

func (s *ClientSynchronizer) processSequenceForceBatches(sequenceForceBatches []etherman.SequencedForceBatch, block etherman.Block, dbTx pgx.Tx) {
	if len(sequenceForceBatches) == 0 {
		log.Error("networkID: %d, error: empty sequenceForceBatches array", s.networkID)
		return
	}
	// First, reset trusted state
	lastVirtualizedBatchNumber := sequenceForceBatches[0].BatchNumber - 1
	err := s.storage.ResetTrustedState(s.ctx, lastVirtualizedBatchNumber, dbTx) // This method has to reset the forced batches deleting the batchNumber for higher batchNumbers
	if err != nil {
		log.Errorf("networkID: %d, error resetting trusted state. BatchNumber: %d, BlockNumber: %d, error: %w",
			s.networkID, lastVirtualizedBatchNumber, block.BlockNumber, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %s, error : %w",
				s.networkID, lastVirtualizedBatchNumber, block.BlockNumber, rollbackErr.Error(), err)
		}
		log.Fatalf("networkID: %d, error resetting trusted state. BatchNumber: %d, BlockNumber: %d, error: %s",
			s.networkID, lastVirtualizedBatchNumber, block.BlockNumber, err.Error())
	}
	// Read forcedBatches from db
	forcedBatches, err := s.storage.GetNextForcedBatches(s.ctx, len(sequenceForceBatches), dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error getting forcedBatches in processSequenceForceBatches. BlockNumber: %d",
			s.networkID, block.BlockNumber)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %s, error : %w",
				s.networkID, block.BlockNumber, rollbackErr.Error(), err)
		}
		log.Fatalf("networkID: %d, error getting forcedBatches in processSequenceForceBatches. BlockNumber: %d, error: %w",
			s.networkID, block.BlockNumber, err)
	}

	if len(sequenceForceBatches) != len(forcedBatches) {
		rollbackErr := dbTx.Rollback(s.ctx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %s, error : %w", s.networkID, block.BlockNumber, rollbackErr.Error(), err)
		}
		log.Fatal("networkID: %d, error number of forced batches doesn't match", s.networkID)
	}

	for i, fbatch := range sequenceForceBatches {
		if uint64(forcedBatches[i].ForcedAt.Unix()) != fbatch.MinForcedTimestamp ||
			forcedBatches[i].GlobalExitRoot != fbatch.GlobalExitRoot ||
			common.Bytes2Hex(forcedBatches[i].RawTxsData) != common.Bytes2Hex(fbatch.Transactions) {
			log.Errorf("networkID: %d, error: forcedBatch received doesn't match with the next expected forcedBatch stored in db. Expected: %+v, Synced: %+v", s.networkID, forcedBatches[i], fbatch)
			rollbackErr := dbTx.Rollback(s.ctx)
			if rollbackErr != nil {
				log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %w", s.networkID, fbatch.BatchNumber, block.BlockNumber, rollbackErr)
			}
			log.Fatalf("networkID: %d, error: forcedBatch received doesn't match with the next expected forcedBatch stored in db. Expected: %+v, Synced: %+v", s.networkID, forcedBatches[i], fbatch)
		}
		b := etherman.Batch{
			BatchNumber:    fbatch.BatchNumber,
			GlobalExitRoot: fbatch.GlobalExitRoot,
			Timestamp:      forcedBatches[i].ForcedAt,
			Coinbase:       fbatch.Sequencer,
			BatchL2Data:    forcedBatches[i].RawTxsData,
		}
		// Add batch, only store it. No need to process txs
		err := s.storage.AddBatch(s.ctx, &b, dbTx)
		if err != nil {
			log.Errorf("networkID: %d, error adding batch in processSequenceForceBatches. BatchNumber: %d, BlockNumber: %d, error: %w",
				s.networkID, b.BatchNumber, block.BlockNumber, err)
			rollbackErr := s.storage.Rollback(s.ctx, dbTx)
			if rollbackErr != nil {
				log.Fatalf("networkID: %d, error rolling back state. BatchNumber: %d, BlockNumber: %d, rollbackErr: %s, error: %w",
					s.networkID, b.BatchNumber, block.BlockNumber, rollbackErr.Error(), err)
			}
			log.Fatalf("networkID: %d, error processing batch in processSequenceForceBatches. BatchNumber: %d, BlockNumber: %d, error: %w",
				s.networkID, b.BatchNumber, block.BlockNumber, err)
		}
		// Store batchNumber in forced_batch table
		err = s.storage.AddBatchNumberInForcedBatch(s.ctx, forcedBatches[i].ForcedBatchNumber, b.BatchNumber, dbTx)
		if err != nil {
			log.Errorf("networkID: %d, error adding the batchNumber to forcedBatch in processSequenceForceBatches. BlockNumber: %d",
				s.networkID, block.BlockNumber)
			rollbackErr := s.storage.Rollback(s.ctx, dbTx)
			if rollbackErr != nil {
				log.Fatalf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %s, error: %w",
					s.networkID, block.BlockNumber, rollbackErr.Error(), err)
			}
			log.Fatalf("networkID: %d, error adding the batchNumber to forcedBatch in processSequenceForceBatches. BlockNumber: %d, error: %w",
				s.networkID, block.BlockNumber, err)
		}
	}
}

func (s *ClientSynchronizer) processForcedBatch(forcedBatch etherman.ForcedBatch, blockID uint64, dbTx pgx.Tx) {
	// Store forced batch into the db
	forcedBatch.BlockID = blockID
	err := s.storage.AddForcedBatch(s.ctx, &forcedBatch, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error storing the forcedBatch in processForcedBatch. BlockNumber: %d",
			s.networkID, forcedBatch.BlockNumber)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %s, error : %s",
				s.networkID, forcedBatch.BlockNumber, rollbackErr.Error(), err.Error())
		}
		log.Fatalf("networkID: %d, error storing the forcedBatch in processForcedBatch. BlockNumber: %d, error: %s",
			s.networkID, forcedBatch.BlockNumber, err.Error())
	}
}

func (s *ClientSynchronizer) processGlobalExitRoot(globalExitRoot etherman.GlobalExitRoot, blockID uint64, dbTx pgx.Tx) {
	// Store GlobalExitRoot
	globalExitRoot.BlockID = blockID
	err := s.storage.AddGlobalExitRoot(s.ctx, &globalExitRoot, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error storing the GlobalExitRoot in processGlobalExitRoot. BlockNumber: %d",
			s.networkID, globalExitRoot.BlockNumber)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %s, error : %s",
				s.networkID, globalExitRoot.BlockNumber, rollbackErr.Error(), err.Error())
		}
		log.Fatalf("networkID: %d, error storing the GlobalExitRoot in processGlobalExitRoot. BlockNumber: %d, error: %s",
			s.networkID, globalExitRoot.BlockNumber, err.Error())
	}
	latestExitRoot, err := s.storage.GetLatestL1SyncedExitRoot(s.ctx, dbTx)
	if err != nil && errors.Is(err, gerror.ErrStorageNotFound) {
		log.Errorf("networkID: %d, error getting the Latest L1 Synced GlobalExitRoot. BlockNumber: %d",
			s.networkID, globalExitRoot.BlockNumber)
	}
	if latestExitRoot.ExitRoots[1] != globalExitRoot.ExitRoots[1] {
		s.chExitRootEvent <- false
	}
}

func (s *ClientSynchronizer) processTrustedVerifyBatch(verifiedBatch etherman.VerifiedBatch, blockID, blockNumber uint64, dbTx pgx.Tx) {
	lastVBatch, err := s.storage.GetLastVerifiedBatch(s.ctx, dbTx)
	if errors.Is(err, gerror.ErrStorageNotFound) {
		lastVBatch = &etherman.VerifiedBatch{
			BatchNumber: 0,
		}
	} else if err != nil {
		log.Errorf("networkID: %d, error getting lastVerifiedBatch stored in db in processTrustedVerifyBatches. Processing synced blockNumber: %d", s.networkID, blockNumber)
		rollbackErr := dbTx.Rollback(s.ctx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state. Processing synced blockNumber: %d, rollbackErr: %s, error : %w", s.networkID, blockNumber, rollbackErr.Error(), err)
		}
		log.Fatalf("networkID: %d, error getting lastVerifiedBatch stored in db in processTrustedVerifyBatches. Processing synced blockNumber: %d, error: %w", s.networkID, blockNumber, err)
	}
	nbatches := verifiedBatch.BatchNumber - lastVBatch.BatchNumber
	var i uint64
	for i = 1; i <= nbatches; i++ {
		verifiedB := etherman.VerifiedBatch{
			BlockID:     blockID,
			BatchNumber: lastVBatch.BatchNumber + i,
			Aggregator:  verifiedBatch.Aggregator,
			StateRoot:   verifiedBatch.StateRoot,
			TxHash:      verifiedBatch.TxHash,
		}
		err = s.storage.AddVerifiedBatch(s.ctx, &verifiedB, dbTx)
		if err != nil {
			log.Errorf("networkID: %d, error storing the verifiedB in processTrustedVerifyBatches. verifiedBatch: %+v, verifiedBatch: %+v", s.networkID, verifiedB, verifiedBatch)
			rollbackErr := dbTx.Rollback(s.ctx)
			if rollbackErr != nil {
				log.Fatalf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %s, error : %w", s.networkID, blockNumber, rollbackErr.Error(), err)
			}
			log.Fatalf("networkID: %d, error storing the verifiedB in processTrustedVerifyBatches. BlockNumber: %d, error: %w", s.networkID, blockNumber, err)
		}
	}
}

func (s *ClientSynchronizer) processDeposit(deposit etherman.Deposit, blockID uint64, dbTx pgx.Tx) {
	deposit.BlockID = blockID
	deposit.NetworkID = s.networkID
	depositID, err := s.storage.AddDeposit(s.ctx, &deposit, dbTx)
	if err != nil {
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state to store block. BlockNumber: %v, rollbackErr: %s, err: %s",
				s.networkID, deposit.BlockNumber, rollbackErr.Error(), err.Error())
		}
		log.Fatalf("networkID: %d, failed to store new deposit locally, BlockNumber: %d, Deposit: %+v err: %s",
			s.networkID, deposit.BlockNumber, deposit, err.Error())
	}

	err = s.bridgeCtrl.AddDeposit(&deposit, depositID, dbTx)
	if err != nil {
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state to store block. BlockNumber: %v, rollbackErr: %s, err: %s",
				s.networkID, deposit.BlockNumber, rollbackErr.Error(), err.Error())
		}
		log.Fatalf("networkID: %d, failed to store new deposit in the bridge tree, BlockNumber: %d, Deposit: %+v err: %s",
			s.networkID, deposit.BlockNumber, deposit, err.Error())
	}
}

func (s *ClientSynchronizer) processClaim(claim etherman.Claim, blockID uint64, dbTx pgx.Tx) {
	claim.BlockID = blockID
	claim.NetworkID = s.networkID
	err := s.storage.AddClaim(s.ctx, &claim, dbTx)
	if err != nil {
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %s, err: %s",
				s.networkID, claim.BlockNumber, rollbackErr.Error(), err.Error())
		}
		log.Fatalf("networkID: %d, error storing new Claim in Block:  %d, Claim: %+v, err: %s",
			s.networkID, claim.BlockNumber, claim, err.Error())
	}
}

func (s *ClientSynchronizer) processTokenWrapped(tokenWrapped etherman.TokenWrapped, blockID uint64, dbTx pgx.Tx) {
	tokenWrapped.BlockID = blockID
	tokenWrapped.NetworkID = s.networkID
	err := s.storage.AddTokenWrapped(s.ctx, &tokenWrapped, dbTx)
	if err != nil {
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %s, err: %s",
				s.networkID, tokenWrapped.BlockNumber, rollbackErr.Error(), err.Error())
		}
		log.Fatalf("networkID: %d, error storing new L1 TokenWrapped in Block:  %d, ExitRoot: %+v, err: %s",
			s.networkID, tokenWrapped.BlockNumber, tokenWrapped, err.Error())
	}
}
