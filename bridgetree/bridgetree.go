package bridgetree

import (
	"context"
	"fmt"

	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
)

const (
	// mainNetworkID  is the default is of the main net
	mainNetworkID = 0 //nolint
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
)

// BridgeTree struct
type BridgeTree struct {
	mainnetTree    *MerkleTree
	roullupTree    *MerkleTree
	globalExitRoot [KeyLen]byte
	storage        Storage
	// database is the kind of database
	database string
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
			mainnetTree:    NewMerkleTree(storage, cfg.Height),
			roullupTree:    NewMerkleTree(storage, cfg.Height),
			globalExitRoot: zeroHashes[cfg.Height+1],
			storage:        storage,
			database:       cfg.Store,
		}, nil
	}
	return nil, gerror.ErrStorageNotRegister
}

// AddDeposit adds deposit information to the bridge tree.
func (bt *BridgeTree) AddDeposit(deposit *etherman.Deposit) error {
	var key string
	leaf := hashDeposit(deposit)
	var ctx context.Context
	if deposit.OriginalNetwork == mainNetworkID {
		key = fmt.Sprintf("%s-%s", bt.database, mainnetKey)
		ctx = context.WithValue(context.TODO(), contextKeyTableName, contextValueMap[key]) //nolint
		err := bt.mainnetTree.addLeaf(ctx, leaf)
		if err != nil {
			return err
		}
	} else {
		key = fmt.Sprintf("%s-%s", bt.database, rollupKey)
		ctx = context.WithValue(context.TODO(), contextKeyTableName, contextValueMap[key]) //nolint
		err := bt.mainnetTree.addLeaf(ctx, leaf)
		if err != nil {
			return err
		}
	}
	err := bt.storage.AddDeposit(ctx, deposit)
	return err
}
