package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/utils/gerror"
	"github.com/hermeznetwork/hermez-core/log"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"
)

const (
	getLastBlockSQL        = "SELECT * FROM sync.block where network_id = $1 ORDER BY block_num DESC LIMIT 1"
	addBlockSQL            = "INSERT INTO sync.block (block_num, block_hash, parent_hash, received_at, network_id) VALUES ($1, $2, $3, $4, $5) RETURNING id;"
	addDepositSQL          = "INSERT INTO sync.deposit (orig_net, token_addr, amount, dest_net, dest_addr, block_num, deposit_cnt, block_id, network_id, tx_hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
	getDepositSQL          = "SELECT orig_net, token_addr, amount, dest_net, dest_addr, block_num, deposit_cnt, block_id, network_id, tx_hash FROM sync.deposit WHERE network_id = $1 AND deposit_cnt = $2"
	getDepositsSQL         = "SELECT orig_net, token_addr, amount, dest_net, dest_addr, block_num, deposit_cnt, block_id, network_id, tx_hash FROM sync.deposit WHERE dest_addr = $1 ORDER BY block_id DESC LIMIT $2 OFFSET $3"
	getNodeByKeySQL        = "SELECT value, deposit_cnt FROM merkletree.rht WHERE key = $1 AND network = $2"
	getRootByDepositCntSQL = "SELECT key FROM merkletree.rht WHERE deposit_cnt = $1 AND depth = $2 AND network = $3"
	getLastDepositCntSQL   = "SELECT coalesce(MAX(deposit_cnt), 0) FROM merkletree.rht WHERE network = $1"
	setNodeByKeySQL        = "INSERT INTO merkletree.rht (key, value, network, deposit_cnt, depth) VALUES ($1, $2, $3, $4, $5)"
	resetNodeByKeySQL      = "DELETE FROM merkletree.rht WHERE deposit_cnt > $1 AND network = $2"
	getPreviousBlockSQL    = "SELECT id, block_num, block_hash, parent_hash, network_id, received_at FROM sync.block WHERE network_id = $1 ORDER BY block_num DESC LIMIT 1 OFFSET $2"
	resetSQL               = "DELETE FROM sync.block WHERE block_num > $1 AND network_id = $2"
	resetConsolidationSQL  = "UPDATE sync.batch SET aggregator = decode('0000000000000000000000000000000000000000', 'hex'), consolidated_tx_hash = decode('0000000000000000000000000000000000000000000000000000000000000000', 'hex'), consolidated_at = null WHERE consolidated_at > $1 AND network_id = $2"
	addGlobalExitRootSQL   = "INSERT INTO sync.exit_root (block_num, global_exit_root_num, mainnet_exit_root, rollup_exit_root, block_id, global_exit_root_l2_num) VALUES ($1, $2, $3, $4, $5, $6)"
	getLatestExitRootSQL   = "SELECT block_id, block_num, global_exit_root_num, mainnet_exit_root, rollup_exit_root FROM sync.exit_root ORDER BY global_exit_root_num DESC LIMIT 1"
	getLatestL1ExitRootSQL = "SELECT block_id, block_num, global_exit_root_num, mainnet_exit_root, rollup_exit_root FROM sync.exit_root WHERE global_exit_root_num = (SELECT MAX(global_exit_root_num)-1 FROM sync.exit_root WHERE global_exit_root_l2_num IS NOT NULL)"
	getLatestL2ExitRootSQL = "SELECT block_id, block_num, global_exit_root_num, mainnet_exit_root, rollup_exit_root, global_exit_root_l2_num FROM sync.exit_root WHERE global_exit_root_l2_num IS NOT NULL ORDER BY global_exit_root_num DESC LIMIT 1"
	getLatestMainnetRSQL   = "SELECT mainnet_exit_root FROM sync.exit_root ORDER BY global_exit_root_num DESC LIMIT 1"
	addClaimSQL            = "INSERT INTO sync.claim (index, orig_net, token_addr, amount, dest_addr, block_num, block_id, network_id, tx_hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	getClaimSQL            = "SELECT index, orig_net, token_addr, amount, dest_addr, block_num, block_id, network_id, tx_hash FROM sync.claim WHERE index = $1 AND network_id = $2"
	getClaimsSQL           = "SELECT index, orig_net, token_addr, amount, dest_addr, block_num, block_id, network_id, tx_hash FROM sync.claim WHERE dest_addr = $1 ORDER BY block_id DESC LIMIT $2 OFFSET $3"
	addTokenWrappedSQL     = "INSERT INTO sync.token_wrapped (orig_net, orig_token_addr, wrapped_token_addr, block_num, block_id, network_id) VALUES ($1, $2, $3, $4, $5, $6)"
	getTokenWrappedSQL     = "SELECT orig_net, orig_token_addr, wrapped_token_addr, block_num, block_id, network_id FROM sync.token_wrapped WHERE orig_net = $1 AND orig_token_addr = $2" // nolint
	consolidateBatchSQL    = "UPDATE sync.batch SET consolidated_tx_hash = $1, consolidated_at = $2, aggregator = $3 WHERE batch_num = $4 AND network_id = $5"
	addBatchSQL            = "INSERT INTO sync.batch (batch_num, batch_hash, block_num, sequencer, aggregator, consolidated_tx_hash, header, uncles, received_at, chain_id, global_exit_root, block_id, network_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)"
	getBatchByNumberSQL    = "SELECT block_num, sequencer, aggregator, consolidated_tx_hash, header, uncles, chain_id, global_exit_root, received_at, consolidated_at, block_id, network_id FROM sync.batch WHERE batch_num = $1 AND network_id = $2"
	getNumDepositsSQL      = "SELECT coalesce(MAX(deposit_cnt),-1) FROM sync.deposit WHERE network_id = $1 AND block_num <= $2"
	getLastBatchNumberSQL  = "SELECT coalesce(max(batch_num),0) as batch FROM sync.batch"
	getLastBatchStateSQL   = "SELECT batch_num, block_id, CASE WHEN consolidated_at IS NULL THEN false ELSE true END AS verified FROM sync.batch ORDER BY batch_num DESC LIMIT 1;"
)

var (
	contextKeyNetwork = "merkle-tree-network"
)

// PostgresStorage implements the Storage interface
type PostgresStorage struct {
	db   *pgxpool.Pool
	dbTx []pgx.Tx
}

// NewPostgresStorage creates a new Storage DB
func NewPostgresStorage(cfg Config, dbTxSize uint) (*PostgresStorage, error) {
	log.Debugf("Create PostgresStorage with Config: %v\n", cfg)
	config, err := pgxpool.ParseConfig(fmt.Sprintf("postgres://%s:%s@%s:%s/%s?pool_max_conns=%d", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.MaxConns))
	if err != nil {
		log.Errorf("Unable to parse DB config: %v\n", err)
		return nil, err
	}
	db, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Errorf("Unable to connect to database: %v\n", err)
		return nil, err
	}
	dbTx := make([]pgx.Tx, dbTxSize)
	return &PostgresStorage{db: db, dbTx: dbTx}, nil
}

// GetLastBlock gets the latest block
func (s *PostgresStorage) GetLastBlock(ctx context.Context, networkID uint) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getLastBlockSQL, networkID).Scan(&block.ID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.NetworkID, &block.ReceivedAt)

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
		deposit.DestinationAddress, deposit.BlockNumber, deposit.DepositCount, deposit.BlockID, deposit.NetworkID, deposit.TxHash)
	return err
}

// GetDeposit gets a specific deposit
func (s *PostgresStorage) GetDeposit(ctx context.Context, depositCounterUser uint, networkID uint) (*etherman.Deposit, error) {
	var (
		deposit etherman.Deposit
		amount  string
	)
	err := s.db.QueryRow(ctx, getDepositSQL, networkID, depositCounterUser).Scan(&deposit.OriginalNetwork, &deposit.TokenAddress,
		&amount, &deposit.DestinationNetwork, &deposit.DestinationAddress, &deposit.BlockNumber, &deposit.DepositCount, &deposit.BlockID,
		&deposit.NetworkID, &deposit.TxHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	deposit.Amount, _ = new(big.Int).SetString(amount, 10) // nolint
	return &deposit, nil
}

// GetDeposits gets the deposit list which be smaller than depositCount
func (s *PostgresStorage) GetDeposits(ctx context.Context, destAddr string, limit uint, offset uint) ([]*etherman.Deposit, error) {
	rows, err := s.db.Query(ctx, getDepositsSQL, common.FromHex(destAddr), limit, offset)
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
		err := rows.Scan(&deposit.OriginalNetwork, &deposit.TokenAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
			&deposit.BlockNumber, &deposit.DepositCount, &deposit.BlockID, &deposit.NetworkID, &deposit.TxHash)
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
func (s *PostgresStorage) Get(ctx context.Context, key []byte) ([][]byte, uint, error) {
	var (
		data       [][]byte
		depositCnt uint
	)
	err := s.db.QueryRow(ctx, getNodeByKeySQL, key, string(ctx.Value(contextKeyNetwork).(uint8))).Scan(pq.Array(&data), &depositCnt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, gerror.ErrStorageNotFound
		}
		return nil, 0, err
	}
	return data, depositCnt, nil
}

// GetRoot gets root by the deposit count from the merkle tree
func (s *PostgresStorage) GetRoot(ctx context.Context, depositCnt uint, depth uint8) ([]byte, error) {
	var root []byte

	err := s.db.QueryRow(ctx, getRootByDepositCntSQL, depositCnt, string(depth+1), string(ctx.Value(contextKeyNetwork).(uint8))).Scan(&root)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gerror.ErrStorageNotFound
		}
		return nil, err
	}
	return root, nil
}

// GetLastDepositCount gets the last deposit count from the merkle tree
func (s *PostgresStorage) GetLastDepositCount(ctx context.Context) (uint, error) {
	var depositCnt uint
	err := s.db.QueryRow(ctx, getLastDepositCntSQL, string(ctx.Value(contextKeyNetwork).(uint8))).Scan(&depositCnt)
	if err != nil {
		return 0, err
	}

	return depositCnt, nil
}

// Set inserts a key-value pair into the db.
// If record with such a key already exists its assumed that the value is correct,
// because it's a reverse hash table, and the key is a hash of the value
func (s *PostgresStorage) Set(ctx context.Context, key []byte, value [][]byte, depositCnt uint, depth uint8) error {
	_, err := s.db.Exec(ctx, setNodeByKeySQL, key, pq.Array(value), string(ctx.Value(contextKeyNetwork).(uint8)), depositCnt, string(depth+1))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil
		}
		return err
	}
	return nil
}

// ResetMT resets nodes of the Merkle Tree.
func (s *PostgresStorage) ResetMT(ctx context.Context, depositCnt uint) error {
	_, err := s.db.Exec(ctx, resetNodeByKeySQL, depositCnt, string(ctx.Value(contextKeyNetwork).(uint8)))
	return err
}

// GetPreviousBlock gets the offset previous block respect to latest
func (s *PostgresStorage) GetPreviousBlock(ctx context.Context, networkID uint, offset uint64) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getPreviousBlockSQL, networkID, offset).Scan(&block.ID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.NetworkID, &block.ReceivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}

// Reset resets the state to a specific block
func (s *PostgresStorage) Reset(ctx context.Context, block *etherman.Block, networkID uint) error {
	if _, err := s.db.Exec(ctx, resetSQL, block.BlockNumber, networkID); err != nil {
		return err
	}

	if block.BlockNumber != 0 { // if its zero, everything is removed so there is no consolidations
		//Remove consolidations
		_, err := s.db.Exec(ctx, resetConsolidationSQL, block.ReceivedAt, networkID)
		return err
	}
	return nil
}

// Rollback rollbacks a db transaction
func (s *PostgresStorage) Rollback(ctx context.Context, index uint) error {
	if s.dbTx[index] != nil {
		err := s.dbTx[index].Rollback(ctx)
		s.dbTx[index] = nil
		return err
	}

	return gerror.ErrNilDBTransaction
}

// Commit commits a db transaction
func (s *PostgresStorage) Commit(ctx context.Context, index uint) error {
	if s.dbTx[index] != nil {
		err := s.dbTx[index].Commit(ctx)
		s.dbTx[index] = nil
		return err
	}
	return gerror.ErrNilDBTransaction
}

// BeginDBTransaction starts a transaction block
func (s *PostgresStorage) BeginDBTransaction(ctx context.Context, index uint) error {
	if s.dbTx[index] != nil {
		return fmt.Errorf("db tx already ongoing!. networkID: %d", index)
	}
	dbTx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	s.dbTx[index] = dbTx
	return nil
}

// AddExitRoot adds a new ExitRoot to the db
func (s *PostgresStorage) AddExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot) error {
	var batch uint64
	err := s.db.QueryRow(ctx, getLastBatchNumberSQL).Scan(&batch)
	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return err
	}
	var mainnetExitRoot common.Hash
	err = s.db.QueryRow(ctx, getLatestMainnetRSQL).Scan(&mainnetExitRoot)
	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return err
	}
	var storeBatch *uint64
	if mainnetExitRoot != exitRoot.ExitRoots[0] {
		storeBatch = nil
	} else {
		storeBatch = &batch
	}
	_, err = s.db.Exec(ctx, addGlobalExitRootSQL, exitRoot.BlockNumber, exitRoot.GlobalExitRootNum.String(), exitRoot.ExitRoots[0], exitRoot.ExitRoots[1], exitRoot.BlockID, storeBatch)
	return err
}

// GetLatestL1SyncedExitRoot get the latest ExitRoot stored that fully synced the rollup exit root in L1.
func (s *PostgresStorage) GetLatestL1SyncedExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	var (
		batch           uint64
		exitRoot        etherman.GlobalExitRoot
		globalNum       uint64
		mainnetExitRoot common.Hash
		rollupExitRoot  common.Hash
	)
	err := s.db.QueryRow(ctx, getLastBatchNumberSQL).Scan(&batch)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	} else if batch == uint64(0) {
		return nil, gerror.ErrStorageNotFound
	}
	err = s.db.QueryRow(ctx, getLatestL1ExitRootSQL).Scan(&exitRoot.BlockID, &exitRoot.BlockNumber, &globalNum, &mainnetExitRoot, &rollupExitRoot)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	exitRoot.GlobalExitRootNum = new(big.Int).SetUint64(globalNum)
	exitRoot.GlobalExitRootL2Num = new(big.Int).SetUint64(batch)
	exitRoot.ExitRoots = []common.Hash{mainnetExitRoot, rollupExitRoot}
	return &exitRoot, nil
}

// GetLatestL2SyncedExitRoot get the latest ExitRoot stored in L2.
func (s *PostgresStorage) GetLatestL2SyncedExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	var (
		exitRoot        etherman.GlobalExitRoot
		globalNum       uint64
		globalL2Num     uint64
		mainnetExitRoot common.Hash
		rollupExitRoot  common.Hash
	)
	err := s.db.QueryRow(ctx, getLatestL2ExitRootSQL).Scan(&exitRoot.BlockID, &exitRoot.BlockNumber, &globalNum, &mainnetExitRoot, &rollupExitRoot, &globalL2Num)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	exitRoot.GlobalExitRootNum = new(big.Int).SetUint64(globalNum)
	exitRoot.GlobalExitRootL2Num = new(big.Int).SetUint64(globalL2Num)
	exitRoot.ExitRoots = []common.Hash{mainnetExitRoot, rollupExitRoot}
	return &exitRoot, nil
}

// GetLatestExitRoot get the latest ExitRoot from L1.
func (s *PostgresStorage) GetLatestExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	var (
		exitRoot        etherman.GlobalExitRoot
		globalNum       uint64
		mainnetExitRoot common.Hash
		rollupExitRoot  common.Hash
	)
	err := s.db.QueryRow(ctx, getLatestExitRootSQL).Scan(&exitRoot.BlockID, &exitRoot.BlockNumber, &globalNum, &mainnetExitRoot, &rollupExitRoot)
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
	_, err := s.db.Exec(ctx, addClaimSQL, claim.Index, claim.OriginalNetwork, claim.Token, claim.Amount.String(),
		claim.DestinationAddress, claim.BlockNumber, claim.BlockID, claim.NetworkID, claim.TxHash)
	return err
}

// GetClaim gets a specific L1 claim
func (s *PostgresStorage) GetClaim(ctx context.Context, depositCounterUser uint, networkID uint) (*etherman.Claim, error) {
	var (
		claim  etherman.Claim
		amount string
	)
	err := s.db.QueryRow(ctx, getClaimSQL, depositCounterUser, networkID).Scan(&claim.Index, &claim.OriginalNetwork, &claim.Token,
		&amount, &claim.DestinationAddress, &claim.BlockNumber, &claim.BlockID, &claim.NetworkID, &claim.TxHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	claim.Amount, _ = new(big.Int).SetString(amount, 10) // nolint
	return &claim, nil
}

// GetClaims gets the claim list which be smaller than index
func (s *PostgresStorage) GetClaims(ctx context.Context, destAddr string, limit uint, offset uint) ([]*etherman.Claim, error) {
	rows, err := s.db.Query(ctx, getClaimsSQL, common.FromHex(destAddr), limit, offset)
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
		err := rows.Scan(&claim.Index, &claim.OriginalNetwork, &claim.Token, &amount, &claim.DestinationAddress, &claim.BlockNumber,
			&claim.BlockID, &claim.NetworkID, &claim.TxHash)
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
		tokeWrapped.WrappedTokenAddress, tokeWrapped.BlockNumber, tokeWrapped.BlockID, tokeWrapped.NetworkID)
	return err
}

// GetTokenWrapped gets a specific tokenWrapped
func (s *PostgresStorage) GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address) (*etherman.TokenWrapped, error) {
	var token etherman.TokenWrapped
	err := s.db.QueryRow(ctx, getTokenWrappedSQL, originalNetwork, originalTokenAddress).Scan(&token.OriginalNetwork, &token.OriginalTokenAddress,
		&token.WrappedTokenAddress, &token.BlockNumber, &token.BlockID, &token.NetworkID)
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
func (s *PostgresStorage) GetNumberDeposits(ctx context.Context, networkID uint, blockNumber uint64) (uint64, error) {
	var nDeposits int64
	err := s.db.QueryRow(ctx, getNumDepositsSQL, networkID, blockNumber).Scan(&nDeposits)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	// 0-index in deposit table, 1-index in MT table
	return uint64(nDeposits + 1), nil
}

// GetLastBatchState returns the lates verified batch number.
func (s *PostgresStorage) GetLastBatchState(ctx context.Context) (uint64, uint64, bool, error) {
	var (
		batchNumber   uint64
		blockID       uint64
		batchVerified bool
	)
	err := s.db.QueryRow(ctx, getLastBatchStateSQL).Scan(&batchNumber, &blockID, &batchVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, false, nil
	} else if err != nil {
		return 0, 0, false, err
	}
	return batchNumber, blockID, batchVerified, err
}
