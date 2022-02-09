package synchronizer

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
	"github.com/hermeznetwork/hermez-bridge/log"
)

// Synchronizer interface
type Synchronizer interface {
	Sync() error
	Stop()
}

// ClientSynchronizer connects L1 and L2
type ClientSynchronizer struct {
	etherMan       etherman.EtherMan
	storage        db.Storage
	ctx            context.Context
	cancelCtx      context.CancelFunc
	genBlockNumber uint64
	cfg            Config
}

// NewSynchronizer creates and initializes an instance of Synchronizer
func NewSynchronizer(ethMan etherman.EtherMan, genBlockNumber uint64, cfg Config) (Synchronizer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &ClientSynchronizer{
		etherMan:       ethMan,
		ctx:            ctx,
		cancelCtx:      cancel,
		genBlockNumber: genBlockNumber,
		cfg:            cfg,
	}, nil
}

// Sync function will read the last state synced and will continue from that point.
// Sync() will read blockchain events to detect bridge updates
func (s *ClientSynchronizer) Sync() error {
	go func() {
		// If there is no lastEthereumBlock means that sync from the beginning is necessary. If not, it continues from the retrieved ethereum block
		// Get the latest synced block. If there is no block on db, use genesis block
		log.Info("Sync started")
		lastEthBlockSynced, err := s.storage.GetLastBlock(s.ctx)
		if err != nil {
			if err == gerror.ErrStorageNotFound {
				log.Warn("error getting the latest ethereum block. No data stored. Setting genesis block. Error: ", err)
				lastEthBlockSynced = &etherman.Block{
					BlockNumber: s.genBlockNumber,
				}
			} else {
				log.Fatal("unexpected error getting the latest ethereum block. Setting genesis block. Error: ", err)
			}
		}
		waitDuration := time.Duration(0)
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(waitDuration):
				if lastEthBlockSynced, err = s.syncBlocks(lastEthBlockSynced); err != nil {
					if s.ctx.Err() != nil {
						continue
					}
				}
				if waitDuration != s.cfg.SyncInterval.Duration {
					// Check latest Ethereum Block
					header, err := s.etherMan.HeaderByNumber(s.ctx, nil)
					if err != nil {
						log.Warn("error getting latest block from Ethereum. Error: ", err)
						continue
					}
					lastKnownBlock := header.Number
					if lastEthBlockSynced.BlockNumber == lastKnownBlock.Uint64() {
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
func (s *ClientSynchronizer) syncBlocks(lastEthBlockSynced *etherman.Block) (*etherman.Block, error) {
	// Call the blockchain to retrieve data
	var fromBlock uint64
	if lastEthBlockSynced.BlockNumber > 0 {
		fromBlock = lastEthBlockSynced.BlockNumber + 1
	}

	header, err := s.etherMan.HeaderByNumber(s.ctx, nil)
	if err != nil {
		return nil, err
	}
	lastKnownBlock := header.Number

	for {
		toBlock := fromBlock + s.cfg.SyncChunkSize

		log.Debugf("Getting bridge info from block %d to block %d", fromBlock, toBlock)
		// This function returns the bridge information contained in the ethereum blocks and an extra param called order.
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
			lastEthBlockSynced = &blocks[len(blocks)-1]
		}
		fromBlock = toBlock + 1

		if lastKnownBlock.Cmp(new(big.Int).SetUint64(fromBlock)) < 1 {
			break
		}
	}

	return lastEthBlockSynced, nil
}

func (s *ClientSynchronizer) processBlockRange(blocks []etherman.Block, order map[common.Hash][]etherman.Order) {
	// New info has to be included into the db using the state
	for i := range blocks {
		// Add block information
		err := s.storage.AddBlock(context.Background(), &blocks[i])
		if err != nil {
			log.Fatal("error storing block. BlockNumber: ", blocks[i].BlockNumber)
		}
		for _, element := range order[blocks[i].BlockHash] {
			if element.Name == etherman.DepositsOrder {
				//TODO Store info into db
				log.Warn("Deposit functionality is not implemented in synchronizer yet")
			} else if element.Name == etherman.GlobalExitRootsOrder {
				//TODO Store info into db
				log.Warn("Consolidate globalExitRoot functionality is not implemented in synchronizer yet")
			} else if element.Name == etherman.ClaimsOrder {
				//TODO Store info into db
				log.Warn("Claim functionality is not implemented in synchronizer yet")
			} else {
				log.Fatal("error: invalid order element")
			}
		}
	}
}
