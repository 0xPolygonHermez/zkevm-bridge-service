package bridgetree

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
)

const (
	// mainNetworkID  is the default is of the main net
	mainNetworkID = 0
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
)

// BridgeTree struct
type BridgeTree struct {
	mainnetTree       *MerkleTree
	rollupTree        *MerkleTree
	globalExitRoot    [KeyLen]byte
	globalExitRootNum *big.Int
	storage           bridgeTreeStorage
	// database is the kind of database
	database string
	lock     sync.RWMutex
}

var (
	contextKeyTableName                   = "merkle-tree-table-name"
	contextValueMap     map[string]string = map[string]string{
		"postgres-mainnet": "merkletree.mainnet",
		"postgres-rollup":  "merkletree.rollup",
	}
	mainnetKey = "mainnet"
	rollupKey  = "rollup"
)

// NewBridgeTree creates new BridgeTree.
func NewBridgeTree(cfg Config, dbConfig db.Config) (*BridgeTree, error) {
	if cfg.Store == "postgres" {
		storage, err := pgstorage.NewPostgresStorage(pgstorage.Config{
			User:     dbConfig.User,
			Password: dbConfig.Password,
			Host:     dbConfig.Host,
			Port:     dbConfig.Port,
			Name:     dbConfig.Name,
		})
		if err != nil {
			return nil, err
		}
		zeroHashes := generateZeroHashes(cfg.Height + 1)
		return &BridgeTree{
			mainnetTree:       NewMerkleTree(storage, cfg.Height),
			rollupTree:        NewMerkleTree(storage, cfg.Height),
			globalExitRoot:    zeroHashes[cfg.Height+1],
			globalExitRootNum: big.NewInt(0),
			storage:           storage,
			database:          cfg.Store,
		}, nil
	}
	return nil, gerror.ErrStorageNotRegister
}

// AddDeposit adds deposit information to the bridge tree.
func (bt *BridgeTree) AddDeposit(deposit *etherman.Deposit) error {
	var (
		key string
		ctx context.Context
	)

	leaf := hashDeposit(deposit)

	bt.lock.Lock()
	defer bt.lock.Unlock()

	if deposit.DestinationNetwork == mainNetworkID {
		key = fmt.Sprintf("%s-%s", bt.database, mainnetKey)
		ctx = context.WithValue(context.TODO(), contextKeyTableName, contextValueMap[key]) //nolint
		err := bt.mainnetTree.addLeaf(ctx, leaf)
		if err != nil {
			return err
		}
	} else {
		key = fmt.Sprintf("%s-%s", bt.database, rollupKey)
		ctx = context.WithValue(context.TODO(), contextKeyTableName, contextValueMap[key]) //nolint
		err := bt.rollupTree.addLeaf(ctx, leaf)
		if err != nil {
			return err
		}
	}

	bt.globalExitRootNum.Add(bt.globalExitRootNum, big.NewInt(1))
	bt.globalExitRoot = hash(bt.mainnetTree.root, bt.rollupTree.root)

	return bt.storage.AddDeposit(ctx, deposit)
}

// GetClaim returns claim information to the user.
func (bt *BridgeTree) GetClaim(networkID uint, index uint64, mtProoves [][KeyLen]byte) (*etherman.Deposit, *etherman.GlobalExitRoot, error) {
	deposit, err := bt.storage.GetDeposit(context.TODO(), networkID, index)
	if err != nil {
		return nil, nil, err
	}

	var (
		key     string
		ctx     context.Context
		prooves [][KeyLen]byte
	)

	bt.lock.RLock()
	defer bt.lock.RUnlock()

	if networkID == mainNetworkID {
		key = fmt.Sprintf("%s-%s", bt.database, mainnetKey)
		ctx = context.WithValue(context.TODO(), contextKeyTableName, contextValueMap[key]) //nolint
		prooves, err = bt.mainnetTree.getProofTreeByIndex(ctx, index)
		if err != nil {
			return deposit, nil, err
		}
	} else {
		key = fmt.Sprintf("%s-%s", bt.database, rollupKey)
		ctx = context.WithValue(context.TODO(), contextKeyTableName, contextValueMap[key]) //nolint
		prooves, err = bt.rollupTree.getProofTreeByIndex(ctx, index)
		if err != nil {
			return deposit, nil, err
		}
	}

	copy(mtProoves, prooves)

	return deposit, &etherman.GlobalExitRoot{MainnetExitRoot: bt.mainnetTree.root, RollupExitRoot: bt.rollupTree.root}, err
}
