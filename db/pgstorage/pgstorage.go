package pgstorage

import (
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"
)

const (
	getLastBlockSQL      = "SELECT * FROM sync.block where network_id = $1 ORDER BY block_num DESC LIMIT 1"
	addBlockSQL          = "INSERT INTO sync.block (block_num, block_hash, parent_hash, received_at, network_id) VALUES ($1, $2, $3, $4, $5) RETURNING id;"
	addDepositSQL        = "INSERT INTO sync.deposit (orig_net, token_addr, amount, dest_net, dest_addr, block_num, deposit_cnt, block_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	getDepositSQL        = "SELECT orig_net, token_addr, amount, dest_net, dest_addr, block_num, deposit_cnt, block_id FROM sync.deposit WHERE orig_net = $1 AND deposit_cnt = $2"
	getDepositsSQL       = "SELECT orig_net, token_addr, amount, dest_net, dest_addr, block_num, deposit_cnt, block_id FROM sync.deposit WHERE orig_net = $1 AND deposit_cnt < $2 ORDER BY deposit_cnt DESC LIMIT $3"
	getNodeByKeySQL      = "SELECT value FROM merkletree.rht WHERE key = $1 AND network = $2"
	setNodeByKeySQL      = "INSERT INTO merkletree.rht (key, value, network) VALUES ($1, $2, $3)"
	getMTRootSQL         = "SELECT index FROM merkletree.root_track WHERE root = $1 AND network = $2"
	setMTRootSQL         = "INSERT INTO merkletree.root_track (index, root, network) VALUES($1, $2, $3)"
	getPreviousBlockSQL  = "SELECT id, block_num, block_hash, parent_hash, network_id, received_at FROM sync.block WHERE network_id = $1 ORDER BY block_num DESC LIMIT 1 OFFSET $2"
	resetSQL             = "DELETE FROM sync.block WHERE block_num > $1 AND network_id = $2"
	addGlobalExitRootSQL = "INSERT INTO sync.exit_root (block_num, global_exit_root_num, mainnet_exit_root, rollup_exit_root, block_id) VALUES ($1, $2, $3, $4, $5)"
	getExitRootSQL       = "SELECT block_id, block_num, global_exit_root_num, mainnet_exit_root, rollup_exit_root FROM sync.exit_root ORDER BY global_exit_root_num DESC LIMIT 1"
	addClaimSQL          = "INSERT INTO sync.claim (index, orig_net, token_addr, amount, dest_addr, block_num, dest_net, block_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	getClaimSQL          = "SELECT index, orig_net, token_addr, amount, dest_addr, block_num, dest_net, block_id FROM sync.claim WHERE index = $1 AND orig_net = $2"
	getClaimsSQL         = "SELECT index, orig_net, token_addr, amount, dest_addr, block_num, dest_net, block_id FROM sync.claim WHERE dest_net = $1 LIMIT $2 OFFSET $3"
	addTokenWrappedSQL   = "INSERT INTO sync.token_wrapped (orig_net, orig_token_addr, wrapped_token_addr, block_num, dest_net, block_id) VALUES ($1, $2, $3, $4, $5, $6)"
	getTokenWrappedSQL   = "SELECT orig_net, orig_token_addr, wrapped_token_addr, block_num, dest_net, block_id FROM sync.token_wrapped WHERE orig_net = $1 AND orig_token_addr = $2" // nolint
	consolidateBatchSQL  = "UPDATE sync.batch SET consolidated_tx_hash = $1, consolidated_at = $2, aggregator = $3 WHERE batch_num = $4 AND network_id = $5"
	addBatchSQL          = "INSERT INTO sync.batch (batch_num, batch_hash, block_num, sequencer, aggregator, consolidated_tx_hash, header, uncles, received_at, chain_id, global_exit_root, block_id, network_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)"
	getBatchByNumberSQL  = "SELECT block_num, sequencer, aggregator, consolidated_tx_hash, header, uncles, chain_id, global_exit_root, received_at, consolidated_at, block_id, network_id FROM sync.batch WHERE batch_num = $1 AND network_id = $2"
	getNumDepositsSQL    = "SELECT MAX(deposit_cnt) FROM sync.deposit WHERE orig_net = $1"
)

var (
	contextKeyNetwork = "merkle-tree-network"
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
func (s *PostgresStorage) GetLastBlock(ctx context.Context, networkID uint) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getLastBlockSQL, networkID).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.ReceivedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}

// AddBlock adds a new block to the db
func (s *PostgresStorage) AddBlock(ctx context.Context, block *etherman.Block) (uint64, error) {
	var id uint64
	err := s.db.QueryRow(ctx, addBlockSQL, block.BlockNumber, block.BlockHash.Bytes(), block.ParentHash.Bytes(), block.ReceivedAt, block.NetworkID).Scan(&id)
	return id, err
}

// AddDeposit adds a new block to the db
func (s *PostgresStorage) AddDeposit(ctx context.Context, deposit *etherman.Deposit) error {
	_, err := s.db.Exec(ctx, addDepositSQL, deposit.OriginalNetwork, deposit.TokenAddress, deposit.Amount.String(), deposit.DestinationNetwork,
		deposit.DestinationAddress, deposit.BlockNumber, deposit.DepositCount, deposit.BlockID)
	return err
}

// GetDeposit gets a specific L1 deposit
func (s *PostgresStorage) GetDeposit(ctx context.Context, depositCounterUser uint, origNetwork uint) (*etherman.Deposit, error) {
	var (
		deposit etherman.Deposit
		amount  string
	)
	err := s.db.QueryRow(ctx, getDepositSQL, origNetwork, depositCounterUser).Scan(&deposit.OriginalNetwork, &deposit.TokenAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress, &deposit.BlockNumber, &deposit.DepositCount, &deposit.BlockID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	deposit.Amount, _ = new(big.Int).SetString(amount, 10) // nolint
	return &deposit, nil
}

// GetDeposits gets the deposit list which be smaller than depositCount
func (s *PostgresStorage) GetDeposits(ctx context.Context, depositCount uint, origNetwork uint, limit uint) ([]*etherman.Deposit, error) {
	rows, err := s.db.Query(ctx, getDepositsSQL, origNetwork, depositCount, limit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))

	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err := rows.Scan(&deposit.OriginalNetwork, &deposit.TokenAddress, &amount, &deposit.DestinationAddress, &deposit.DestinationAddress, &deposit.BlockNumber, &deposit.DepositCount, &deposit.BlockID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gerror.ErrStorageNotFound
		} else if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) // nolint
		deposits = append(deposits, &deposit)
	}

	return deposits, nil
}

// Get gets value of key from the merkle tree
func (s *PostgresStorage) Get(ctx context.Context, key []byte) ([][]byte, error) {
	var data [][]byte
	err := s.db.QueryRow(ctx, getNodeByKeySQL, key, string(ctx.Value(contextKeyNetwork).(uint8))).Scan(pq.Array(&data))
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
func (s *PostgresStorage) Set(ctx context.Context, key []byte, value [][]byte) error {
	_, err := s.db.Exec(ctx, setNodeByKeySQL, key, pq.Array(value), string(ctx.Value(contextKeyNetwork).(uint8)))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil
		}
		return err
	}
	return nil
}

// GetMTRoot returns deposit count for the specific root in the merkle tree
func (s *PostgresStorage) GetMTRoot(ctx context.Context, root []byte) (uint, error) {
	var index uint
	err := s.db.QueryRow(ctx, getMTRootSQL, root, string(ctx.Value(contextKeyNetwork).(uint8))).Scan(&index)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, gerror.ErrStorageNotFound
		}
		return 0, err
	}
	return index, nil
}

// SetMTRoot inserts a track of root to the merkle tree
func (s *PostgresStorage) SetMTRoot(ctx context.Context, index uint, root []byte) error {
	_, err := s.db.Exec(ctx, setMTRootSQL, index, root, string(ctx.Value(contextKeyNetwork).(uint8)))
	return err
}

// GetPreviousBlock gets the offset previous block respect to latest
func (s *PostgresStorage) GetPreviousBlock(ctx context.Context, networkID uint, offset uint64) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getPreviousBlockSQL, networkID, offset).Scan(&block.BlockID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.NetworkID, &block.ReceivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}

// Reset resets the state to a specific block
func (s *PostgresStorage) Reset(ctx context.Context, blockNumber uint64, networkID uint) error {
	_, err := s.db.Exec(ctx, resetSQL, blockNumber, networkID)
	return err
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
	_, err := s.db.Exec(ctx, addGlobalExitRootSQL, exitRoot.BlockNumber, exitRoot.GlobalExitRootNum.String(), exitRoot.ExitRoots[0], exitRoot.ExitRoots[1], exitRoot.BlockID)
	return err
}

// GetLatestExitRoot get the latest ExitRoot stored
func (s *PostgresStorage) GetLatestExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	var (
		exitRoot        etherman.GlobalExitRoot
		globalNum       uint64
		mainnetExitRoot common.Hash
		rollupExitRoot  common.Hash
	)
	err := s.db.QueryRow(ctx, getExitRootSQL).Scan(&exitRoot.BlockID, &exitRoot.BlockNumber, &globalNum, &mainnetExitRoot, &rollupExitRoot)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	exitRoot.GlobalExitRootNum = new(big.Int).SetUint64(globalNum)
	exitRoot.ExitRoots = []common.Hash{mainnetExitRoot, rollupExitRoot}
	return &exitRoot, nil
}

// AddClaim adds a new claim to the db
func (s *PostgresStorage) AddClaim(ctx context.Context, claim *etherman.Claim) error {
	_, err := s.db.Exec(ctx, addClaimSQL, claim.Index, claim.OriginalNetwork, claim.Token, claim.Amount.String(), claim.DestinationAddress, claim.BlockNumber, claim.DestinationNetwork, claim.BlockID)
	return err
}

// GetClaim gets a specific L1 claim
func (s *PostgresStorage) GetClaim(ctx context.Context, depositCounterUser uint, origNetwork uint) (*etherman.Claim, error) {
	var (
		claim  etherman.Claim
		amount string
	)
	err := s.db.QueryRow(ctx, getClaimSQL, depositCounterUser, origNetwork).Scan(&claim.Index, &claim.OriginalNetwork, &claim.Token, &amount, &claim.DestinationAddress, &claim.BlockNumber, &claim.DestinationNetwork, &claim.BlockID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	claim.Amount, _ = new(big.Int).SetString(amount, 10) // nolint
	return &claim, nil
}

// GetClaims gets the claim list which be smaller than index
func (s *PostgresStorage) GetClaims(ctx context.Context, destNetwork uint, limit uint, offset uint) ([]*etherman.Claim, error) {
	rows, err := s.db.Query(ctx, getClaimsSQL, destNetwork, limit, offset)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	claims := make([]*etherman.Claim, 0, len(rows.RawValues()))

	for rows.Next() {
		var (
			claim  etherman.Claim
			amount string
		)
		err := rows.Scan(&claim.Index, &claim.OriginalNetwork, &claim.Token, &amount, &claim.DestinationAddress, &claim.BlockNumber, &claim.DestinationNetwork, &claim.BlockID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gerror.ErrStorageNotFound
		} else if err != nil {
			return nil, err
		}
		claim.Amount, _ = new(big.Int).SetString(amount, 10) // nolint
		claims = append(claims, &claim)
	}

	return claims, nil
}

// AddTokenWrapped adds a new claim to the db
func (s *PostgresStorage) AddTokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped) error {
	_, err := s.db.Exec(ctx, addTokenWrappedSQL, tokeWrapped.OriginalNetwork, tokeWrapped.OriginalTokenAddress,
		tokeWrapped.WrappedTokenAddress, tokeWrapped.BlockNumber, tokeWrapped.DestinationNetwork, tokeWrapped.BlockID)
	return err
}

// GetTokenWrapped gets a specific L1 tokenWrapped
func (s *PostgresStorage) GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address) (*etherman.TokenWrapped, error) {
	var token etherman.TokenWrapped
	err := s.db.QueryRow(ctx, getTokenWrappedSQL, originalNetwork, originalTokenAddress).Scan(&token.OriginalNetwork, &token.OriginalTokenAddress,
		&token.WrappedTokenAddress, &token.BlockNumber, &token.DestinationNetwork, &token.BlockID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	return &token, nil
}

// ConsolidateBatch changes the virtual status of a batch
func (s *PostgresStorage) ConsolidateBatch(ctx context.Context, batch *etherman.Batch) error {
	_, err := s.db.Exec(ctx, consolidateBatchSQL, batch.ConsolidatedTxHash, batch.ConsolidatedAt, batch.Aggregator, batch.Number().Uint64(), batch.NetworkID)
	return err
}

// AddBatch adds a new batch to the db
func (s *PostgresStorage) AddBatch(ctx context.Context, batch *etherman.Batch) error {
	_, err := s.db.Exec(ctx, addBatchSQL, batch.Number().Uint64(), batch.Hash(), batch.BlockNumber, batch.Sequencer, batch.Aggregator,
		batch.ConsolidatedTxHash, batch.Header, batch.Uncles, batch.ReceivedAt, batch.ChainID.String(), batch.GlobalExitRoot, batch.BlockID, batch.NetworkID)
	return err
}

// GetBatchByNumber gets the batch with the required number
func (s *PostgresStorage) GetBatchByNumber(ctx context.Context, batchNumber uint64, networkID uint) (*etherman.Batch, error) {
	var (
		batch etherman.Batch
		chain uint64
	)
	err := s.db.QueryRow(ctx, getBatchByNumberSQL, batchNumber, networkID).Scan(
		&batch.BlockNumber, &batch.Sequencer, &batch.Aggregator, &batch.ConsolidatedTxHash,
		&batch.Header, &batch.Uncles, &chain, &batch.GlobalExitRoot, &batch.ReceivedAt,
		&batch.ConsolidatedAt, &batch.BlockID, &batch.NetworkID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	batch.ChainID = new(big.Int).SetUint64(chain)

	return &batch, nil
}

// GetNumberDeposits gets the number of  deposits
func (s *PostgresStorage) GetNumberDeposits(ctx context.Context, origNetworkID uint) (uint64, error) {
	var nDeposits uint64
	err := s.db.QueryRow(ctx, getNumDepositsSQL, origNetworkID).Scan(&nDeposits)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, gerror.ErrStorageNotFound
	} else if err != nil {
		return 0, err
	}
	return nDeposits, nil
}
