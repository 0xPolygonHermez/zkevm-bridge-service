package synchronizer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

// Synchronizer connects L1 and L2
type Synchronizer interface {
	Sync() error
	Stop()
}

// ClientSynchronizer connects L1 and L2
type ClientSynchronizer struct {
	etherMan         ethermanInterface
	bridgeCtrl       bridgectrlInterface
	storage          storageInterface
	ctx              context.Context
	cancelCtx        context.CancelFunc
	genBlockNumber   uint64
	cfg              Config
	networkID        uint
	chExitRootEvent  chan *etherman.GlobalExitRoot
	chSynced         chan uint
	zkEVMClient      zkEVMClientInterface
	synced           bool
	l1RollupExitRoot common.Hash
}

// NewSynchronizer creates and initializes an instance of Synchronizer
func NewSynchronizer(
	storage interface{},
	bridge bridgectrlInterface,
	ethMan ethermanInterface,
	zkEVMClient zkEVMClientInterface,
	genBlockNumber uint64,
	chExitRootEvent chan *etherman.GlobalExitRoot,
	chSynced chan uint,
	cfg Config) (Synchronizer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	networkID, err := ethMan.GetNetworkID(ctx)
	if err != nil {
		log.Fatal("error getting networkID. Error: ", err)
	}
	ger, err := storage.(storageInterface).GetLatestL1SyncedExitRoot(context.Background(), nil)
	if err != nil {
		if err == gerror.ErrStorageNotFound {
			ger.ExitRoots = []common.Hash{{}, {}}
		} else {
			log.Fatal("error getting last L1 synced exitroot. Error: ", err)
		}
	}

	if networkID == 0 {
		return &ClientSynchronizer{
			bridgeCtrl:       bridge,
			storage:          storage.(storageInterface),
			etherMan:         ethMan,
			ctx:              ctx,
			cancelCtx:        cancel,
			genBlockNumber:   genBlockNumber,
			cfg:              cfg,
			networkID:        networkID,
			chExitRootEvent:  chExitRootEvent,
			chSynced:         chSynced,
			zkEVMClient:      zkEVMClient,
			l1RollupExitRoot: ger.ExitRoots[1],
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
		chSynced:       chSynced,
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
	lastBlockSynced, err := s.storage.GetLastBlock(s.ctx, s.networkID, nil)
	if err != nil {
		if err == gerror.ErrStorageNotFound {
			log.Warnf("networkID: %d, error getting the latest ethereum block. No data stored. Setting genesis block. Error: %w", s.networkID, err)
			lastBlockSynced = &etherman.Block{
				BlockNumber: s.genBlockNumber,
				NetworkID:   s.networkID,
			}
			log.Warnf("networkID: %d, error getting the latest block. No data stored. Using initial block: %+v. Error: %s",
				s.networkID, lastBlockSynced, err.Error())
		} else {
			log.Fatalf("networkID: %d, unexpected error getting the latest block. Error: %s", s.networkID, err.Error())
		}
	}
	log.Debugf("NetworkID: %d, initial lastBlockSynced: %+v", s.networkID, lastBlockSynced)
	for {
		select {
		case <-s.ctx.Done():
			log.Debugf("NetworkID: %d, synchronizer ctx done", s.networkID)
			return nil
		case <-time.After(waitDuration):
			log.Debugf("NetworkID: %d, syncing...", s.networkID)
			//Sync L1Blocks
			if lastBlockSynced, err = s.syncBlocks(lastBlockSynced); err != nil {
				log.Warnf("networkID: %d, error syncing blocks: ", s.networkID, err)
				lastBlockSynced, err = s.storage.GetLastBlock(s.ctx, s.networkID, nil)
				if err != nil {
					log.Fatalf("networkID: %d, error getting lastBlockSynced to resume the synchronization... Error: ", s.networkID, err)
				}
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
				lastKnownBlock := header.Number.Uint64()
				if lastBlockSynced.BlockNumber == lastKnownBlock && !s.synced {
					log.Infof("NetworkID %d Synced!", s.networkID)
					waitDuration = s.cfg.SyncInterval.Duration
					s.synced = true
					s.chSynced <- s.networkID
				}
				if lastBlockSynced.BlockNumber > lastKnownBlock {
					if s.networkID == 0 {
						log.Fatalf("networkID: %d, error: latest Synced BlockNumber (%d) is higher than the latest Proposed block (%d) in the network", s.networkID, lastBlockSynced.BlockNumber, lastKnownBlock)
					} else {
						log.Errorf("networkID: %d, error: latest Synced BlockNumber (%d) is higher than the latest Proposed block (%d) in the network", s.networkID, lastBlockSynced.BlockNumber, lastKnownBlock)
						err = s.resetState(lastKnownBlock)
						if err != nil {
							log.Errorf("networkID: %d, error resetting the state to a previous block. Error: %v", s.networkID, err)
							continue
						}
					}
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
	lastBatchNumber, err := s.zkEVMClient.BatchNumber(s.ctx)
	if err != nil {
		log.Errorf("networkID: %d, error getting latest batch number from rpc. Error: %w", s.networkID, err)
		return err
	}
	lastBatch, err := s.zkEVMClient.BatchByNumber(s.ctx, big.NewInt(0).SetUint64(lastBatchNumber))
	if err != nil {
		log.Warnf("networkID: %d, failed to get batch %v from trusted state. Error: %v", s.networkID, lastBatchNumber, err)
		return err
	}
	ger := &etherman.GlobalExitRoot{
		GlobalExitRoot: lastBatch.GlobalExitRoot,
		ExitRoots: []common.Hash{
			lastBatch.MainnetExitRoot,
			lastBatch.RollupExitRoot,
		},
	}
	isUpdated, err := s.storage.AddTrustedGlobalExitRoot(s.ctx, ger, nil)
	if err != nil {
		log.Error("networkID: %d, error storing latest trusted globalExitRoot. Error: %w", s.networkID, err)
		return err
	}
	if isUpdated {
		s.chExitRootEvent <- ger
	}
	return nil
}

// This function syncs the node from a specific block to the latest
func (s *ClientSynchronizer) syncBlocks(lastBlockSynced *etherman.Block) (*etherman.Block, error) {
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
		// Order param is a map that contains the event order to allow the synchronizer store the info in the same order that is read.
		// Name can be different in the order struct. This name is an identifier to check if the next info that must be stored in the db.
		// The value pos (position) tells what is the array index where this value is.
		blocks, order, err := s.etherMan.GetRollupInfoByBlockRange(s.ctx, fromBlock, &toBlock)
		if err != nil {
			return lastBlockSynced, err
		}
		err = s.processBlockRange(blocks, order)
		if err != nil {
			return lastBlockSynced, err
		}
		if len(blocks) > 0 {
			lastBlockSynced = &blocks[len(blocks)-1]
			for i := range blocks {
				log.Debug("NetworkID: ", s.networkID, ", Position: ", i, ". BlockNumber: ", blocks[i].BlockNumber, ". BlockHash: ", blocks[i].BlockHash)
			}
		}
		fromBlock = toBlock + 1

		if lastKnownBlock.Cmp(new(big.Int).SetUint64(toBlock)) < 1 {
			if !s.synced {
				log.Infof("NetworkID %d Synced!", s.networkID)
				waitDuration = s.cfg.SyncInterval.Duration
				s.synced = true
				s.chSynced <- s.networkID
			}
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
			err = s.processBlockRange([]etherman.Block{b}, order)
			if err != nil {
				return lastBlockSynced, err
			}

			lastBlockSynced = &b
			log.Debugf("NetworkID: %d, Storing empty block. BlockNumber: %d. BlockHash: %s",
				s.networkID, b.BlockNumber, b.BlockHash.String())
		}
	}

	return lastBlockSynced, nil
}

func (s *ClientSynchronizer) processBlockRange(blocks []etherman.Block, order map[common.Hash][]etherman.Order) error {
	// New info has to be included into the db using the state
	for i := range blocks {
		// Begin db transaction
		dbTx, err := s.storage.BeginDBTransaction(s.ctx)
		if err != nil {
			log.Errorf("networkID: %d, error creating db transaction to store block. BlockNumber: %d. Error: %v",
				s.networkID, blocks[i].BlockNumber, err)
			return err
		}
		// Add block information
		blocks[i].NetworkID = s.networkID
		log.Infof("NetworkID: %d. Syncing block: %d", s.networkID, blocks[i].BlockNumber)
		blockID, err := s.storage.AddBlock(s.ctx, &blocks[i], dbTx)
		if err != nil {
			log.Errorf("networkID: %d, error storing block. BlockNumber: %d, error: %v", s.networkID, blocks[i].BlockNumber, err)
			rollbackErr := s.storage.Rollback(s.ctx, dbTx)
			if rollbackErr != nil {
				log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %v, err: %s",
					s.networkID, blocks[i].BlockNumber, rollbackErr, err.Error())
				return rollbackErr
			}
			return err
		}
		for _, element := range order[blocks[i].BlockHash] {
			switch element.Name {
			case etherman.GlobalExitRootsOrder:
				err = s.processGlobalExitRoot(blocks[i].GlobalExitRoots[element.Pos], blockID, dbTx)
				if err != nil {
					return err
				}
			case etherman.DepositsOrder:
				err = s.processDeposit(blocks[i].Deposits[element.Pos], blockID, dbTx)
				if err != nil {
					return err
				}
			case etherman.ClaimsOrder:
				err = s.processClaim(blocks[i].Claims[element.Pos], blockID, dbTx)
				if err != nil {
					return err
				}
			case etherman.TokensOrder:
				err = s.processTokenWrapped(blocks[i].Tokens[element.Pos], blockID, dbTx)
				if err != nil {
					return err
				}
			}
		}
		err = s.storage.Commit(s.ctx, dbTx)
		if err != nil {
			log.Errorf("networkID: %d, error committing state to store block. BlockNumber: %d, err: %v",
				s.networkID, blocks[i].BlockNumber, err)
			rollbackErr := s.storage.Rollback(s.ctx, dbTx)
			if rollbackErr != nil {
				log.Errorf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %v, err: %s",
					s.networkID, blocks[i].BlockNumber, rollbackErr, err.Error())
				return rollbackErr
			}
			return err
		}
	}
	return nil
}

// This function allows reset the state until an specific ethereum block
func (s *ClientSynchronizer) resetState(blockNumber uint64) error {
	log.Infof("NetworkID: %d. Reverting synchronization to block: %d", s.networkID, blockNumber)
	dbTx, err := s.storage.BeginDBTransaction(s.ctx)
	if err != nil {
		log.Errorf("networkID: %d, Error starting a db transaction to reset the state. Error: %v", s.networkID, err)
		return err
	}
	err = s.storage.Reset(s.ctx, blockNumber, s.networkID, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error resetting the state. Error: %v", s.networkID, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %v, error : %s",
				s.networkID, blockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}
	depositCnt, err := s.storage.GetNumberDeposits(s.ctx, s.networkID, blockNumber, dbTx)
	if err != nil {
		log.Error("networkID: %d, error getting GetNumberDeposits. Error: %v", s.networkID, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %v, error : %s",
				s.networkID, blockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}

	err = s.bridgeCtrl.ReorgMT(uint(depositCnt), s.networkID, dbTx)
	if err != nil {
		log.Error("networkID: %d, error resetting ReorgMT the state. Error: %v", s.networkID, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %v, error : %s",
				s.networkID, blockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}
	err = s.storage.Commit(s.ctx, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error committing the resetted state. Error: %v", s.networkID, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %v, error : %s",
				s.networkID, blockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
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
			log.Errorf("networkID: %d, error getting latest block synced from blockchain. Block: %d, error: %v",
				s.networkID, latestBlock.BlockNumber, err)
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
			log.Info("NetworkID: ", s.networkID, ", [checkReorg function] => latestBlockNumber: ", latestBlock.BlockNumber)
			log.Info("NetworkID: ", s.networkID, ", [checkReorg function] => latestBlockHash: ", latestBlock.BlockHash)
			log.Info("NetworkID: ", s.networkID, ", [checkReorg function] => latestBlockHashParent: ", latestBlock.ParentHash)
			log.Info("NetworkID: ", s.networkID, ", [checkReorg function] => BlockNumber: ", latestBlock.BlockNumber, block.NumberU64())
			log.Info("NetworkID: ", s.networkID, ", [checkReorg function] => BlockHash: ", block.Hash())
			log.Info("NetworkID: ", s.networkID, ", [checkReorg function] => BlockHashParent: ", block.ParentHash())
			depth++
			log.Info("NetworkID: ", s.networkID, ", REORG: Looking for the latest correct block. Depth: ", depth)
			// Reorg detected. Getting previous block
			dbTx, err := s.storage.BeginDBTransaction(s.ctx)
			if err != nil {
				log.Errorf("networkID: %d, error creating db transaction to get previous blocks. Error: %v", s.networkID, err)
				return nil, err
			}
			latestBlock, err = s.storage.GetPreviousBlock(s.ctx, s.networkID, depth, dbTx)
			errC := s.storage.Commit(s.ctx, dbTx)
			if errC != nil {
				log.Errorf("networkID: %d, error committing dbTx, err: %v", s.networkID, errC)
				rollbackErr := s.storage.Rollback(s.ctx, dbTx)
				if rollbackErr != nil {
					log.Errorf("networkID: %d, error rolling back state. RollbackErr: %v, err: %s",
						s.networkID, rollbackErr, errC.Error())
					return nil, rollbackErr
				}
				return nil, errC
			}
			if errors.Is(err, gerror.ErrStorageNotFound) {
				log.Warnf("networkID: %d, error checking reorg: previous block not found in db: %v", s.networkID, err)
				return &etherman.Block{}, nil
			} else if err != nil {
				log.Errorf("networkID: %d, error detected getting previous block: %v", s.networkID, err)
				return nil, err
			}
		} else {
			break
		}
	}
	if latestBlockSynced.BlockHash != latestBlock.BlockHash {
		log.Infof("NetworkID: %d, reorg detected in block: %d", s.networkID, latestBlockSynced.BlockNumber)
		return latestBlock, nil
	}
	log.Debugf("NetworkID: %d, no reorg detected", s.networkID)
	return nil, nil
}

func (s *ClientSynchronizer) processGlobalExitRoot(globalExitRoot etherman.GlobalExitRoot, blockID uint64, dbTx pgx.Tx) error {
	// Store GlobalExitRoot
	globalExitRoot.BlockID = blockID
	err := s.storage.AddGlobalExitRoot(s.ctx, &globalExitRoot, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error storing the GlobalExitRoot in processGlobalExitRoot. BlockNumber: %d. Error: %v", s.networkID, globalExitRoot.BlockNumber, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state. BlockNumber: %d, rollbackErr: %v, error : %s",
				s.networkID, globalExitRoot.BlockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}
	if s.l1RollupExitRoot != globalExitRoot.ExitRoots[1] {
		s.l1RollupExitRoot = globalExitRoot.ExitRoots[1]
		s.chExitRootEvent <- &globalExitRoot
	}
	return nil
}

func (s *ClientSynchronizer) processDeposit(deposit etherman.Deposit, blockID uint64, dbTx pgx.Tx) error {
	deposit.BlockID = blockID
	deposit.NetworkID = s.networkID
	depositID, err := s.storage.AddDeposit(s.ctx, &deposit, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, failed to store new deposit locally, BlockNumber: %d, Deposit: %+v err: %v", s.networkID, deposit.BlockNumber, deposit, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %v, rollbackErr: %v, err: %s",
				s.networkID, deposit.BlockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}

	err = s.bridgeCtrl.AddDeposit(&deposit, depositID, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, failed to store new deposit in the bridge tree, BlockNumber: %d, Deposit: %+v err: %v", s.networkID, deposit.BlockNumber, deposit, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %v, rollbackErr: %v, err: %s",
				s.networkID, deposit.BlockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}
	return nil
}

func (s *ClientSynchronizer) processClaim(claim etherman.Claim, blockID uint64, dbTx pgx.Tx) error {
	claim.BlockID = blockID
	claim.NetworkID = s.networkID
	err := s.storage.AddClaim(s.ctx, &claim, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error storing new Claim in Block:  %d, Claim: %+v, err: %v", s.networkID, claim.BlockNumber, claim, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %v, err: %s",
				s.networkID, claim.BlockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}
	return nil
}

func (s *ClientSynchronizer) processTokenWrapped(tokenWrapped etherman.TokenWrapped, blockID uint64, dbTx pgx.Tx) error {
	tokenWrapped.BlockID = blockID
	tokenWrapped.NetworkID = s.networkID
	err := s.storage.AddTokenWrapped(s.ctx, &tokenWrapped, dbTx)
	if err != nil {
		log.Errorf("networkID: %d, error storing new L1 TokenWrapped in Block:  %d, ExitRoot: %+v, err: %v", s.networkID, tokenWrapped.BlockNumber, tokenWrapped, err)
		rollbackErr := s.storage.Rollback(s.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %d, rollbackErr: %v, err: %s",
				s.networkID, tokenWrapped.BlockNumber, rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}
	return nil
}
