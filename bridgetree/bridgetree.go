package bridgetree

import (
	"context"
	"fmt"

	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
)

// KeyLen is the length of key and value in the Merkle Tree
const (
	MAIN_NETWORK_ID = 0
	KeyLen          = 32
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
	rollupKey  = "rollupu"
)

type favContextKey string

// NewBridgeTree creates new BridgeTree
func NewBridgeTree(cfg Config, dbConfig db.Config) (*BridgeTree, error) {
	if cfg.Store == "postgres" {
		storage, err := pgstorage.NewPostgresStorage(dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Name)
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

func (bt *BridgeTree) AddDeposit(deposit *etherman.Deposit) error {
	var key string
	leaf := hashDeposit(deposit)
	var ctx context.Context
	if deposit.OriginalNetwork == MAIN_NETWORK_ID {
		key = fmt.Sprintf("%s-%s", bt.database, mainnetKey)
		ctx = context.WithValue(context.TODO(), favContextKey(contextKeyTableName), contextValueMap[key])
		err := bt.mainnetTree.addLeaf(ctx, leaf)
		if err != nil {
			return err
		}
	} else {
		key = fmt.Sprintf("%s-%s", bt.database, rollupKey)
		ctx = context.WithValue(context.TODO(), favContextKey(contextKeyTableName), contextValueMap[key])
		err := bt.mainnetTree.addLeaf(ctx, leaf)
		if err != nil {
			return err
		}
	}
	bt.storage.AddDeposit(ctx, deposit)
	return nil
}
