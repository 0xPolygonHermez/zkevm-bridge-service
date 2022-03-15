package synchronizer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/bridgetree"
	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
	"github.com/hermeznetwork/hermez-core/log"
)

// Synchronizer interface
type Synchronizer interface {
	Sync() error
	Stop()
}

// ClientSynchronizer
type ClientSynchronizer struct {
	etherMan       localEtherMan
	storage        db.Storage
	ctx            context.Context
	cancelCtx      context.CancelFunc
	bridgeTree     *bridgetree.BridgeTree
	genBlockNumber uint64
	cfg            Config
	networkID      uint
}

// NewSynchronizer creates and initializes an instance of Synchronizer
func NewSynchronizer(storage db.Storage, bridge *bridgetree.BridgeTree, ethMan localEtherMan, genBlockNumber uint64, cfg Config) (Synchronizer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	networkID, err := ethMan.GetNetworkID(ctx)
	if err != nil {
		log.Fatal("error getting networkID. Error: ", err)
	}
	return &ClientSynchronizer{
		etherMan:       ethMan,
		storage:        storage,
		ctx:            ctx,
		cancelCtx:      cancel,
		bridgeTree:     bridge,
		genBlockNumber: genBlockNumber,
		cfg:            cfg,
		networkID:      networkID,
	}, nil
}

// Sync function will read the last state synced and will continue from that point.
// Sync() will read blockchain events to detect bridge updates
func (s *ClientSynchronizer) Sync() error {
	go func() {
		// If there is no lastBlock means that sync from the beginning is necessary. If not, it continues from the retrieved block
		// Get the latest synced block. If there is no block on db, use genesis block
		var (
			err             error
			lastBlockSynced *etherman.Block
		)
		log.Info("Synchronization started")
		lastBlockSynced, err = s.storage.GetLastBlock(s.ctx, s.networkID)
		if err != nil {
			if err == gerror.ErrStorageNotFound {
				log.Warn("error getting the latest block. No data stored. Setting genesis block. Error: ", err)
				lastBlockSynced = &etherman.Block {
					BlockNumber: s.genBlockNumber,
					NetworkID: s.networkID,
				}
			} else {
				log.Fatal("unexpected error getting the latest block. Error: ", err)
			}
		}
		waitDuration := time.Duration(0)
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(waitDuration):
				if lastBlockSynced, err = s.syncBlocks(lastBlockSynced); err != nil {
					if s.ctx.Err() != nil {
						continue
					}
				}
				if waitDuration != s.cfg.SyncInterval.Duration {
					// Check latest Block
					header, err := s.etherMan.HeaderByNumber(s.ctx, nil)
					if err != nil {
						log.Warn("error getting latest block from. Error: ", err)
						continue
					}
					lastKnownBlock := header.Number
					if lastBlockSynced.BlockNumber == lastKnownBlock.Uint64() {
						waitDuration = s.cfg.SyncInterval.Duration
					}
				}
			}
		}
	}()
	return nil
}

// Stop function stops the synchronizer
func (s *ClientSynchronizer) Stop() {
	s.cancelCtx()
}

// This function syncs the node from a specific block to the latest
func (s *ClientSynchronizer) syncBlocks(lastBlockSynced *etherman.Block) (*etherman.Block, error) {
	// This function will read events fromBlockNum to latestBlock. Check reorg to be sure that everything is ok.
	block, err := s.checkReorg(lastBlockSynced)
	if err != nil {
		log.Errorf("error checking reorgs. Retrying... Err: %s", err.Error())
		return lastBlockSynced, fmt.Errorf("error checking reorgs")
	} else if block != nil {
		err = s.resetState(block)
		if err != nil {
			log.Error("error resetting the state to a previous block. Retrying...")
			return lastBlockSynced, fmt.Errorf("error resetting the state to a previous block")
		}
		return block, nil
	}
	// Call the blockchain to retrieve data
	var fromBlock uint64
	if lastBlockSynced.BlockNumber > 0 {
		fromBlock = lastBlockSynced.BlockNumber + 1
	}

	header, err := s.etherMan.HeaderByNumber(s.ctx, nil)
	if err != nil {
		return nil, err
	}
	lastKnownBlock := header.Number

	for {
		toBlock := fromBlock + s.cfg.SyncChunkSize

		log.Debugf("Getting bridge info from block %d to block %d", fromBlock, toBlock)
		// This function returns the bridge information contained in the blocks and an extra param called order.
		// Order param is a map that contains the event order to allow the synchronizer store the info in the same order that is readed.
		// Name can be defferent in the order struct. For instance: Batches or Name:NewSequencers. This name is an identifier to check
		// if the next info that must be stored in the db is a new sequencer or a batch. The value pos (position) tells what is the
		// array index where this value is.
		blocks, order, err := s.etherMan.GetBridgeInfoByBlockRange(s.ctx, fromBlock, &toBlock)
		if err != nil {
			return nil, err
		}
		s.processBlockRange(blocks, order)
		if len(blocks) > 0 {
			lastBlockSynced = &blocks[len(blocks)-1]
		}
		fromBlock = toBlock + 1

		if lastKnownBlock.Cmp(new(big.Int).SetUint64(fromBlock)) < 1 {
			break
		}
	}

	return lastBlockSynced, nil
}

func (s *ClientSynchronizer) processBlockRange(blocks []etherman.Block, order map[common.Hash][]etherman.Order) {
	// New info has to be included into the db using the state
	for i := range blocks {
		ctx := context.Background()
		blocks[i].NetworkID = s.networkID
		// Begin db transaction
		err := s.storage.BeginDBTransaction(ctx)
		if err != nil {
			log.Fatal("error createing db transaction to store block. BlockNumber: ", blocks[i].BlockNumber)
		}
		// Add block information
		blockID, err := s.storage.AddBlock(ctx, &blocks[i])
		if err != nil {
			log.Fatal("error storing block. BlockNumber: ", blocks[i].BlockNumber)
		}
		for _, element := range order[blocks[i].BlockHash] {
			if element.Name == etherman.BatchesOrder {
				batch := &blocks[i].Batches[element.Pos]
				batch.BlockID = blockID
				batch.NetworkID = s.networkID
				emptyHash := common.Hash{}
				log.Debug("consolidatedTxHash received: ", batch.ConsolidatedTxHash)
				if batch.ConsolidatedTxHash.String() != emptyHash.String() {
					err = s.storage.ConsolidateBatch(ctx, batch)
					if err != nil {
						err = s.storage.Rollback(ctx)
						if err != nil {
							log.Fatal("error rolling back state to store block. BlockNumber: ", blocks[i].BlockNumber)
						}
						log.Fatal("failed to consolidate batch locally, batch number: %d, err: %v", batch.Number().Uint64(), err)
					}
				} else {
					err = s.storage.AddBatch(ctx, batch)
					if err != nil {
						err = s.storage.Rollback(ctx)
						if err != nil {
							log.Fatal("error rolling back state to store block. BlockNumber: ", blocks[i].BlockNumber)
						}
						log.Fatal("failed to add batch locally, batch number: %d, err: %v", batch.Number().Uint64(), err)
					}
				}
			} else if element.Name == etherman.DepositsOrder {
				deposit := &blocks[i].Deposits[element.Pos]
				deposit.BlockID = blockID
				err := s.storage.AddDeposit(ctx, deposit)
				if err != nil {
					err = s.storage.Rollback(ctx)
					if err != nil {
						log.Fatal("error rolling back state to store block. BlockNumber: ", blocks[i].BlockNumber)
					}
					log.Fatal("failed to store new deposit locally, block: %d, Deposit: %+v err: %v", &blocks[i].BlockNumber, deposit, err)
				}

				err = s.bridgeTree.AddDeposit(deposit)
				if err != nil {
					log.Fatal("failed to store new deposit in the bridge tree, block: %d, Deposit: %+v err: %v", &blocks[i].BlockNumber, deposit, err)
				}
			} else if element.Name == etherman.GlobalExitRootsOrder {
				exitRoot := blocks[i].GlobalExitRoots[element.Pos]
				exitRoot.BlockID = blockID
				exitRoot.NetworkID = s.networkID
				err := s.storage.AddExitRoot(ctx, &exitRoot)
				if err != nil {
					err = s.storage.Rollback(ctx)
					if err != nil {
						log.Fatal("error rolling back state to store block. BlockNumber: ", blocks[i].BlockNumber)
					}
					log.Fatal("error storing new globalExitRoot in Block: %d, ExitRoot: %+v, err: %v", blocks[i].BlockNumber, exitRoot, err)
				}
			} else if element.Name == etherman.ClaimsOrder {
				claim := blocks[i].Claims[element.Pos]
				claim.BlockID = blockID
				claim.DestinationNetwork = s.networkID
				err := s.storage.AddClaim(ctx, &claim)
				if err != nil {
					err = s.storage.Rollback(ctx)
					if err != nil {
						log.Fatal("error rolling back state to store block. BlockNumber: ", blocks[i].BlockNumber)
					}
					log.Fatal("error storing new L1 Claim in Block:  %d, Claim: %+v, err: %v", blocks[i].BlockNumber, claim, err)
				}
			} else if element.Name == etherman.TokensOrder {
				tokenWrapped := blocks[i].Tokens[element.Pos]
				tokenWrapped.BlockID = blockID
				tokenWrapped.DestinationNetwork = s.networkID
				err := s.storage.AddTokenWrapped(ctx, &tokenWrapped)
				if err != nil {
					err = s.storage.Rollback(ctx)
					if err != nil {
						log.Fatal("error rolling back state to store block. BlockNumber: ", blocks[i].BlockNumber)
					}
					log.Fatal("error storing new L1 TokenWrapped in Block:  %d, ExitRoot: %+v, err: %v", blocks[i].BlockNumber, tokenWrapped, err)
				}
			} else {
				log.Fatal("error: invalid order element")
			}
		}
	}
}

// This function allows reset the state until an specific block
func (s *ClientSynchronizer) resetState(block *etherman.Block) error {
	log.Debug("NetworkID: ", s.networkID, ". Reverting synchronization to block: ", block.BlockNumber)
	err := s.storage.Reset(s.ctx, block.BlockNumber, s.networkID)
	if err != nil {
		return err
	}
	return nil
}

/*
This function will check if there is a reorg.
As input param needs the last block synced. Retrieve the block info from the blockchain
to compare it with the stored info. If hash and hash parent matches, then no reorg is detected and return a nil.
If hash or hash parent don't match, reorg detected and the function will return the block until the sync process
must be reverted. Then, check the previous block synced, get block info from the blockchain and check
hash and has parent. This operation has to be done until a match is found.
*/
func (s *ClientSynchronizer) checkReorg(latestBlock *etherman.Block) (*etherman.Block, error) {
	// This function only needs to worry about reorgs if some of the reorganized blocks contained rollup info.
	latestBlockSynced := *latestBlock
	var depth uint64
	for {
		block, err := s.etherMan.BlockByNumber(s.ctx, latestBlock.BlockNumber)
		if err != nil {
			if errors.Is(err, etherman.ErrNotFound) {
				return nil, nil
			} else if strings.Contains(err.Error(), "connection refused") {
				log.Fatal("Connection refused: ", err)
			}
			return nil, err
		}
		if block.NumberU64() != latestBlock.BlockNumber {
			log.Error("Wrong block retrieved from blockchain. Block numbers don't match. BlockNumber stored: ",
				latestBlock.BlockNumber, ". BlockNumber retrieved: ", block.NumberU64())
			return nil, fmt.Errorf("Wrong block retrieved from blockchain. Block numbers don't match. BlockNumber stored: %d. BlockNumber retrieved: %d",
				latestBlock.BlockNumber, block.NumberU64())
		}
		// Compare hashes
		if (block.Hash() != latestBlock.BlockHash || block.ParentHash() != latestBlock.ParentHash) && latestBlock.BlockNumber > s.genBlockNumber {
			log.Debug("[checkReorg function] => latestBlockNumber: ", latestBlock.BlockNumber)
			log.Debug("[checkReorg function] => latestBlockHash: ", latestBlock.BlockHash)
			log.Debug("[checkReorg function] => latestBlockHashParent: ", latestBlock.ParentHash)
			log.Debug("[checkReorg function] => BlockNumber: ", latestBlock.BlockNumber, block.NumberU64())
			log.Debug("[checkReorg function] => BlockHash: ", block.Hash())
			log.Debug("[checkReorg function] => BlockHashParent: ", block.ParentHash())
			depth++
			log.Debug("REORG: Looking for the latest correct block. Depth: ", depth)
			// Reorg detected. Getting previous block
			latestBlock, err = s.storage.GetPreviousBlock(s.ctx, s.networkID, depth)
			if errors.Is(err, gerror.ErrStorageNotFound) {
				log.Warn("error checking reorg: previous block not found in db: ", err)
				return nil, nil
			} else if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	if latestBlockSynced.BlockHash != latestBlock.BlockHash {
		log.Debug("Reorg detected in block: ", latestBlockSynced.BlockNumber)
		return latestBlock, nil
	}
	return nil, nil
}
