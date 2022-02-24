package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"math/big"
	"time"

	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	getLastBlockSQL       = "SELECT * FROM sync.block ORDER BY block_num DESC LIMIT 1"
	getLastL2BlockSQL     = "SELECT * FROM sync.l2_block ORDER BY block_num DESC LIMIT 1"
	addBlockSQL           = "INSERT INTO sync.block (block_num, block_hash, parent_hash, received_at) VALUES ($1, $2, $3, $4)"
	addL2BlockSQL         = "INSERT INTO sync.l2_block (block_num, block_hash, parent_hash, received_at) VALUES ($1, $2, $3, $4)"
	addDepositSQL         = "INSERT INTO sync.deposit (orig_net, token_addr, amount, dest_net, dest_addr, block_num, deposit_cnt) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	addL2DepositSQL       = "INSERT INTO sync.l2_deposit (orig_net, token_addr, amount, dest_net, dest_addr, l2_block_num, deposit_cnt) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	getNodeByKeySQL       = "SELECT value FROM %s WHERE key = $1"
	setNodeByKeySQL       = "INSERT INTO %s (key, value) VALUES ($1, $2) ON CONFLICT(key) DO UPDATE SET value = $2"
	getPreviousBlockSQL   = "SELECT * FROM sync.block ORDER BY block_num DESC LIMIT 1 OFFSET $1"
	getPreviousL2BlockSQL = "SELECT * FROM sync.l2_block ORDER BY block_num DESC LIMIT 1 OFFSET $1"
	resetSQL              = "DELETE FROM sync.block WHERE block_num > $1"
	resetL2SQL            = "DELETE FROM sync.l2_block WHERE block_num > $1"
	addGlobalExitRootSQL  = "INSERT INTO sync.exit_root (block_num, global_exit_root_num, mainnet_exit_root, rollup_exit_root) VALUES ($1, $2, $3, $4)"
	getExitRootSQL        = "SELECT block_num, global_exit_root_num, mainnet_exit_root, rollup_exit_root FROM sync.exit_root ORDER BY global_exit_root_num DESC LIMIT 1"
	addClaimSQL           = "INSERT INTO sync.claim (index, orig_net, token_addr, amount, dest_addr, block_num) VALUES ($1, $2, $3, $4, $5, $6)"
	addL2ClaimSQL         = "INSERT INTO sync.l2_claim (index, orig_net, token_addr, amount, dest_addr, l2_block_num) VALUES ($1, $2, $3, $4, $5, $6)"
	addTokenWrappedSQL    = "INSERT INTO sync.token_wrapped (orig_net, orig_token_addr, wrapped_token_addr, block_num) ($1, $2, $3, $4)"
	addL2TokenWrappedSQL  = "INSERT INTO sync.l2_token_wrapped (orig_net, orig_token_addr, wrapped_token_addr, l2_block_num) ($1, $2, $3, $4)"
	consolidateBatchSQL   = "UPDATE sync.batch SET consolidated_tx_hash = $1, consolidated_at = $3, aggregator = $4 WHERE batch_num = $2"
	addBatchSQL           = "INSERT INTO sync.batch (batch_num, batch_hash, block_num, sequencer, aggregator, consolidated_tx_hash, header, uncles, received_at, chain_id, global_exit_root) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)"
)

var (
	contextKeyTableName = "merkle-tree-table-name"
)

// PostgresStorage implements the Storage interface
type PostgresStorage struct {
	db   *pgxpool.Pool
	dbTx pgx.Tx
}

// NewPostgresStorage creates a new Storage DB
func NewPostgresStorage(cfg Config) (*PostgresStorage, error) {
	db, err := pgxpool.Connect(context.Background(), "postgres://"+cfg.User+":"+cfg.Password+"@"+cfg.Host+":"+cfg.Port+"/"+cfg.Name)
	if err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db}, nil
}

// GetLastBlock gets the latest block
func (s *PostgresStorage) GetLastBlock(ctx context.Context) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getLastBlockSQL).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.ReceivedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}

// GetLastL2Block gets the latest block
func (s *PostgresStorage) GetLastL2Block(ctx context.Context) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getLastL2BlockSQL).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.ReceivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}

// AddBlock adds a new block to the db
func (s *PostgresStorage) AddBlock(ctx context.Context, block *etherman.Block) error {
	_, err := s.db.Exec(ctx, addBlockSQL, block.BlockNumber, block.BlockHash.Bytes(), block.ParentHash.Bytes(), block.ReceivedAt)
	return err
}

// AddL2Block adds a new l2 block to the db
func (s *PostgresStorage) AddL2Block(ctx context.Context, block *etherman.Block) error {
	_, err := s.db.Exec(ctx, addL2BlockSQL, block.BlockNumber, block.BlockHash.Bytes(), block.ParentHash.Bytes(), block.ReceivedAt)
	return err
}

// AddDeposit adds a new block to the db
func (s *PostgresStorage) AddDeposit(ctx context.Context, deposit *etherman.Deposit) error {
	_, err := s.db.Exec(ctx, addDepositSQL, deposit.OriginalNetwork, deposit.TokenAddress, deposit.Amount.String(), deposit.DestinationNetwork, deposit.DestinationAddress, deposit.BlockNumber, deposit.DepositCount)
	return err
}

// AddL2Deposit adds a new block to the db
func (s *PostgresStorage) AddL2Deposit(ctx context.Context, deposit *etherman.Deposit) error {
	_, err := s.db.Exec(ctx, addL2DepositSQL, deposit.OriginalNetwork, deposit.TokenAddress, deposit.Amount.String(), deposit.DestinationNetwork, deposit.DestinationAddress, deposit.BlockNumber, deposit.DepositCount)
	return err
}

// Get gets value of key from the merkle tree
func (s *PostgresStorage) Get(ctx context.Context, key []byte) ([]byte, error) {
	var data []byte
	err := s.db.QueryRow(ctx, fmt.Sprintf(getNodeByKeySQL, ctx.Value(contextKeyTableName).(string)), key).Scan(&data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gerror.ErrStorageNotFound
		}
		return nil, err
	}
	return data, nil
}

// Set inserts a key-value pair into the db.
// If record with such a key already exists its assumed that the value is correct,
// because it's a reverse hash table, and the key is a hash of the value
func (s *PostgresStorage) Set(ctx context.Context, key []byte, value []byte) error {
	_, err := s.db.Exec(ctx, fmt.Sprintf(setNodeByKeySQL, ctx.Value(contextKeyTableName).(string)), key, value)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil
		}
		return err
	}
	return nil
}

// GetPreviousBlock gets the offset previous block respect to latest
func (s *PostgresStorage) GetPreviousBlock(ctx context.Context, offset uint64) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getPreviousBlockSQL, offset).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.ReceivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}

// GetPreviousL2Block gets the offset previous block respect to latest
func (s *PostgresStorage) GetPreviousL2Block(ctx context.Context, offset uint64) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getPreviousL2BlockSQL, offset).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.ReceivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}

// ResetL2 resets the state to a specific L2 block (batch)
func (s *PostgresStorage) ResetL2(ctx context.Context, blockNumber uint64) error {
	if _, err := s.db.Exec(ctx, resetL2SQL, blockNumber); err != nil {
		return err
	}
	return nil
}

// Reset resets the state to a specific block
func (s *PostgresStorage) Reset(ctx context.Context, blockNumber uint64) error {
	if _, err := s.db.Exec(ctx, resetSQL, blockNumber); err != nil {
		return err
	}
	return nil
}

// Rollback rollbacks a db transaction
func (s *PostgresStorage) Rollback(ctx context.Context) error {
	if s.dbTx != nil {
		err := s.dbTx.Rollback(ctx)
		s.dbTx = nil
		return err
	}

	return gerror.ErrNilDBTransaction
}

// BeginDBTransaction starts a transaction block
func (s *PostgresStorage) BeginDBTransaction(ctx context.Context) error {
	dbTx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	s.dbTx = dbTx
	return nil
}

// AddExitRoot adds a new ExitRoot to the db
func (s *PostgresStorage) AddExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot) error {
	_, err := s.db.Exec(ctx, addGlobalExitRootSQL, exitRoot.BlockNumber, exitRoot.GlobalExitRootNum.String(), exitRoot.MainnetExitRoot, exitRoot.RollupExitRoot)
	return err
}

// GetLatestExitRoot get the latest ExitRoot stored
func (s *PostgresStorage) GetLatestExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	var (
		exitRoot  etherman.GlobalExitRoot
		globalNum uint64
	)
	err := s.db.QueryRow(ctx, getExitRootSQL).Scan(&exitRoot.BlockNumber, &globalNum, &exitRoot.MainnetExitRoot, &exitRoot.RollupExitRoot)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	exitRoot.GlobalExitRootNum = new(big.Int).SetUint64(globalNum)
	return &exitRoot, nil
}

// AddClaim adds a new claim to the db
func (s *PostgresStorage) AddClaim(ctx context.Context, claim *etherman.Claim) error {
	_, err := s.db.Exec(ctx, addClaimSQL, claim.Index, claim.OriginalNetwork, claim.Token, claim.Amount, claim.DestinationAddress, claim.BlockNumber)
	return err
}

// AddL2Claim adds a new claim to the db
func (s *PostgresStorage) AddL2Claim(ctx context.Context, claim *etherman.Claim) error {
	_, err := s.db.Exec(ctx, addL2ClaimSQL, claim.Index, claim.OriginalNetwork, claim.Token, claim.Amount, claim.DestinationAddress, claim.BlockNumber)
	return err
}

// AddTokenWrapped adds a new claim to the db
func (s *PostgresStorage) AddTokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped) error {
	_, err := s.db.Exec(ctx, addTokenWrappedSQL, tokeWrapped.OriginalNetwork, tokeWrapped.OriginalTokenAddress, tokeWrapped.WrappedTokenAddress, tokeWrapped.BlockNum)
	return err
}

// AddL2TokenWrapped adds a new claim to the db
func (s *PostgresStorage) AddL2TokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped) error {
	_, err := s.db.Exec(ctx, addL2TokenWrappedSQL, tokeWrapped.OriginalNetwork, tokeWrapped.OriginalTokenAddress, tokeWrapped.WrappedTokenAddress, tokeWrapped.BlockNum)
	return err
}

// ConsolidateBatch changes the virtual status of a batch
func (s *PostgresStorage) ConsolidateBatch(ctx context.Context, batchNumber uint64, consolidatedTxHash common.Hash, consolidatedAt time.Time, aggregator common.Address) error {
	if _, err := s.db.Exec(ctx, consolidateBatchSQL, consolidatedTxHash, batchNumber, consolidatedAt, aggregator); err != nil {
		return err
	}
	return nil
}

// AddBatch adds a new batch to the db
func (s *PostgresStorage) AddBatch(ctx context.Context, batch *etherman.Batch) error {
	_, err := s.db.Exec(ctx, addBatchSQL, batch.Number().Uint64(), batch.Hash(), batch.BlockNumber, batch.Sequencer, batch.Aggregator,
		batch.ConsolidatedTxHash, batch.Header, batch.Uncles, batch.ReceivedAt, batch.ChainID.String(), batch.GlobalExitRoot)
	return err
}