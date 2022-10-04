package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"
)

// PostgresStorage implements the Storage interface.
type PostgresStorage struct {
	*pgxpool.Pool
}

// getExecQuerier determines which execQuerier to use, dbTx or the main pgxpool
func (p *PostgresStorage) getExecQuerier(dbTx pgx.Tx) execQuerier {
	if dbTx != nil {
		return dbTx
	}
	return p
}

// NewPostgresStorage creates a new Storage DB
func NewPostgresStorage(cfg Config) (*PostgresStorage, error) {
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
	return &PostgresStorage{db}, nil
}

// Rollback rollbacks a db transaction.
func (p *PostgresStorage) Rollback(ctx context.Context, dbTx pgx.Tx) error {
	if dbTx != nil {
		return dbTx.Rollback(ctx)
	}

	return gerror.ErrNilDBTransaction
}

// Commit commits a db transaction.
func (p *PostgresStorage) Commit(ctx context.Context, dbTx pgx.Tx) error {
	if dbTx != nil {
		return dbTx.Commit(ctx)
	}
	return gerror.ErrNilDBTransaction
}

// BeginDBTransaction starts a transaction block.
func (p *PostgresStorage) BeginDBTransaction(ctx context.Context) (pgx.Tx, error) {
	return p.Begin(ctx)
}

// GetLastBlock gets the last block.
func (p *PostgresStorage) GetLastBlock(ctx context.Context, networkID uint, dbTx pgx.Tx) (*etherman.Block, error) {
	var block etherman.Block
	const getLastBlockSQL = "SELECT id, block_num, block_hash, parent_hash, network_id, received_at FROM syncv2.block where network_id = $1 ORDER BY block_num DESC LIMIT 1"

	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getLastBlockSQL, networkID).Scan(&block.ID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.NetworkID, &block.ReceivedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}

	return &block, err
}

// GetLastBatchNumber gets the last batch number.
func (p *PostgresStorage) GetLastBatchNumber(ctx context.Context, dbTx pgx.Tx) (uint64, error) {
	var batchNumber uint64
	const getLastBatchNumberSQL = "SELECT coalesce(max(batch_num),0) as batch FROM syncv2.batch"

	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getLastBatchNumberSQL).Scan(&batchNumber)

	return batchNumber, err
}

// GetBatchByNumber gets the specific batch by the batch number.
func (p *PostgresStorage) GetBatchByNumber(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) (*etherman.Batch, error) {
	var batch etherman.Batch
	const getBatchByNumberSQL = "SELECT batch_num, sequencer, raw_tx_data, timestamp, global_exit_root FROM syncv2.batch WHERE batch_num = $1"

	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getBatchByNumberSQL, batchNumber).Scan(
		&batch.BatchNumber, &batch.Coinbase, &batch.BatchL2Data, &batch.Timestamp, &batch.GlobalExitRoot)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}

	return &batch, err
}

// AddBlock adds a new block to the storage.
func (p *PostgresStorage) AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error) {
	var blockID uint64
	const addBlockSQL = "INSERT INTO syncv2.block (block_num, block_hash, parent_hash, network_id, received_at) VALUES ($1, $2, $3, $4, $5) RETURNING id;"
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, addBlockSQL, block.BlockNumber, block.BlockHash, block.ParentHash, block.NetworkID, block.ReceivedAt).Scan(&blockID)

	return blockID, err
}

// AddBatch adds a new batch to the storage.
func (p *PostgresStorage) AddBatch(ctx context.Context, batch *etherman.Batch, dbTx pgx.Tx) error {
	const addBatchSQL = "INSERT INTO syncv2.batch (batch_num, sequencer, raw_tx_data, timestamp, global_exit_root) VALUES ($1, $2, $3, $4, $5)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addBatchSQL, batch.BatchNumber, batch.Coinbase, batch.BatchL2Data, batch.Timestamp, batch.GlobalExitRoot)
	return err
}

// AddVerifiedBatch adds a new verified batch.
func (p *PostgresStorage) AddVerifiedBatch(ctx context.Context, verifiedBatch *etherman.VerifiedBatch, dbTx pgx.Tx) error {
	const addVerifiedBatchSQL = "INSERT INTO syncv2.verified_batch (batch_num, aggregator, tx_hash, block_id) VALUES ($1, $2, $3, $4)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addVerifiedBatchSQL, verifiedBatch.BatchNumber, verifiedBatch.Aggregator, verifiedBatch.TxHash, verifiedBatch.BlockID)

	return err
}

// AddGlobalExitRoot adds a new ExitRoot to the db.
func (p *PostgresStorage) AddGlobalExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error {
	const addExitRootSQL = "INSERT INTO syncv2.exit_root (block_id, global_exit_root_num, global_exit_root, exit_roots) VALUES ($1, $2, $3, $4)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addExitRootSQL, exitRoot.BlockID, exitRoot.GlobalExitRootNum.String(), exitRoot.GlobalExitRoot, pq.Array([][]byte{exitRoot.ExitRoots[0][:], exitRoot.ExitRoots[1][:]}))
	return err
}

// AddDeposit adds new deposit to the storage.
func (p *PostgresStorage) AddDeposit(ctx context.Context, deposit *etherman.Deposit, dbTx pgx.Tx) error {
	const addDepositSQL = "INSERT INTO syncv2.deposit (network_id, orig_net, token_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addDepositSQL, deposit.NetworkID, deposit.OriginalNetwork, deposit.TokenAddress, deposit.Amount.String(), deposit.DestinationNetwork, deposit.DestinationAddress, deposit.BlockID, deposit.DepositCount, deposit.TxHash, deposit.Metadata)
	return err
}

// AddClaim adds new claim to the storage.
func (p *PostgresStorage) AddClaim(ctx context.Context, claim *etherman.Claim, dbTx pgx.Tx) error {
	const addClaimSQL = "INSERT INTO syncv2.claim (network_id, index, orig_net, token_addr, amount, dest_addr, block_id, tx_hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addClaimSQL, claim.NetworkID, claim.Index, claim.OriginalNetwork, claim.Token, claim.Amount.String(), claim.DestinationAddress, claim.BlockID, claim.TxHash)
	return err
}

// GetTokenMetadata gets the metadata of the dedicated token.
func (p *PostgresStorage) GetTokenMetadata(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) ([]byte, error) {
	var metadata []byte
	const getMetadataSQL = "SELECT metadata from syncv2.deposit WHERE network_id = $1 AND token_addr = $2 AND dest_net = $3 AND metadata IS NOT NULL LIMIT 1"
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getMetadataSQL, tokenWrapped.OriginalNetwork, tokenWrapped.OriginalTokenAddress, tokenWrapped.NetworkID).Scan(&metadata)
	return metadata, err
}

// AddTokenWrapped adds new wrapped token to the storage.
func (p *PostgresStorage) AddTokenWrapped(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) error {
	metadata, err := p.GetTokenMetadata(ctx, tokenWrapped, dbTx)
	if err != nil {
		return err
	}
	tokenMetadata, err := getDecodedToken(metadata)
	if err != nil {
		return err
	}
	const addTokenWrappedSQL = "INSERT INTO syncv2.token_wrapped (network_id, orig_net, orig_token_addr, wrapped_token_addr, block_id, name, symbol, decimals) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	e := p.getExecQuerier(dbTx)
	_, err = e.Exec(ctx, addTokenWrappedSQL, tokenWrapped.NetworkID, tokenWrapped.OriginalNetwork, tokenWrapped.OriginalTokenAddress, tokenWrapped.WrappedTokenAddress, tokenWrapped.BlockID, tokenMetadata.Name, tokenMetadata.Symbol, tokenMetadata.Decimals)
	return err
}

// Reset resets the state to a block for the given DB tx.
func (p *PostgresStorage) Reset(ctx context.Context, blockNumber uint64, dbTx pgx.Tx) error {
	const resetSQL = "DELETE FROM syncv2.block WHERE block_num > $1"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, resetSQL, blockNumber)
	return err
}

// GetPreviousBlock gets the offset previous L1 block respect to latest.
func (p *PostgresStorage) GetPreviousBlock(ctx context.Context, networkID uint, offset uint64, dbTx pgx.Tx) (*etherman.Block, error) {
	var block etherman.Block
	const getPreviousBlockSQL = "SELECT block_num, block_hash, parent_hash, network_id, received_at FROM syncv2.block WHERE network_id = $1 ORDER BY block_num DESC LIMIT 1 OFFSET $2"
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getPreviousBlockSQL, networkID, offset).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.NetworkID, &block.ReceivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}
	return &block, err
}

// GetNumberDeposits gets the number of  deposits.
func (p *PostgresStorage) GetNumberDeposits(ctx context.Context, networkID uint, blockNumber uint64, dbTx pgx.Tx) (uint64, error) {
	var nDeposits int64
	const getNumDepositsSQL = "SELECT coalesce(MAX(deposit_cnt), -1) FROM syncv2.deposit as d INNER JOIN syncv2.block as b ON d.network_id = b.network_id AND d.block_id = b.id WHERE d.network_id = $1 AND b.block_num <= $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getNumDepositsSQL, networkID, blockNumber).Scan(&nDeposits)
	return uint64(nDeposits + 1), err
}

// GetNextForcedBatches gets the next forced batches from the queue.
func (p *PostgresStorage) GetNextForcedBatches(ctx context.Context, nextForcedBatches int, dbTx pgx.Tx) ([]etherman.ForcedBatch, error) {
	const getNextForcedBatchesSQL = "SELECT forced_batch_num, global_exit_root, raw_tx_data, sequencer, batch_num, block_id FROM syncv2.forced_batch WHERE batch_num IS NULL LIMIT $1"
	e := p.getExecQuerier(dbTx)
	// Get the next forced batches
	rows, err := e.Query(ctx, getNextForcedBatchesSQL, nextForcedBatches)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	defer rows.Close()

	batches := make([]etherman.ForcedBatch, 0, len(rows.RawValues()))
	var forcedBatch etherman.ForcedBatch
	for rows.Next() {
		err := rows.Scan(&forcedBatch.ForcedBatchNumber, &forcedBatch.GlobalExitRoot, &forcedBatch.RawTxsData, &forcedBatch.Sequencer, &forcedBatch.BatchNumber, &forcedBatch.BlockID)
		if err != nil {
			return nil, err
		}
		batches = append(batches, forcedBatch)
	}

	return batches, nil
}

// AddBatchNumberInForcedBatch updates the forced_batch table with the batchNumber.
func (p *PostgresStorage) AddBatchNumberInForcedBatch(ctx context.Context, forceBatchNumber, batchNumber uint64, dbTx pgx.Tx) error {
	const addBatchNumberInForcedBatchSQL = "UPDATE syncv2.forced_batch SET batch_num = $2 WHERE forced_batch_num = $1"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addBatchNumberInForcedBatchSQL, forceBatchNumber, batchNumber)
	return err
}

// AddForcedBatch adds a new ForcedBatch to the db.
func (p *PostgresStorage) AddForcedBatch(ctx context.Context, forcedBatch *etherman.ForcedBatch, dbTx pgx.Tx) error {
	const addForcedBatchSQL = "INSERT INTO syncv2.forced_batch (forced_batch_num, global_exit_root, raw_tx_data, sequencer, batch_num, block_id) VALUES ($1, $2, $3, $4, $5, $6)"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, addForcedBatchSQL, forcedBatch.ForcedBatchNumber, forcedBatch.GlobalExitRoot, forcedBatch.RawTxsData, forcedBatch.Sequencer, forcedBatch.BatchNumber, forcedBatch.BlockID)
	return err
}

// AddTrustedGlobalExitRoot adds new global exit root which comes from the trusted sequencer.
func (p *PostgresStorage) AddTrustedGlobalExitRoot(ctx context.Context, trustedExitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error {
	const addTrustedGerSQL = `
		INSERT INTO syncv2.exit_root (block_id, global_exit_root_num, global_exit_root, exit_roots) 
		VALUES (0, $1, $2, $3)
		ON CONFLICT (block_id, global_exit_root_num) DO UPDATE
			SET global_exit_root = $2,
				exit_roots = $3;`
	_, err := p.getExecQuerier(dbTx).Exec(ctx, addTrustedGerSQL, trustedExitRoot.GlobalExitRootNum.String(), trustedExitRoot.GlobalExitRoot, pq.Array([][]byte{trustedExitRoot.ExitRoots[0][:], trustedExitRoot.ExitRoots[1][:]}))
	return err
}

// GetClaim gets a specific claim from the storage.
func (p *PostgresStorage) GetClaim(ctx context.Context, depositCounterUser uint, networkID uint, dbTx pgx.Tx) (*etherman.Claim, error) {
	var (
		claim  etherman.Claim
		amount string
	)
	const getClaimSQL = "SELECT index, orig_net, token_addr, amount, dest_addr, block_id, network_id, tx_hash FROM syncv2.claim WHERE index = $1 AND network_id = $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getClaimSQL, depositCounterUser, networkID).Scan(&claim.Index, &claim.OriginalNetwork, &claim.Token, &amount, &claim.DestinationAddress, &claim.BlockID, &claim.NetworkID, &claim.TxHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}
	claim.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
	return &claim, err
}

// GetDeposit gets a specific deposit from the storage.
func (p *PostgresStorage) GetDeposit(ctx context.Context, depositCounterUser uint, networkID uint, dbTx pgx.Tx) (*etherman.Deposit, error) {
	var (
		deposit etherman.Deposit
		amount  string
	)
	const getDepositSQL = "SELECT orig_net, token_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata FROM syncv2.deposit as d INNER JOIN syncv2.block as b ON d.network_id = b.network_id AND d.block_id = b.id WHERE d.network_id = $1 AND deposit_cnt = $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getDepositSQL, networkID, depositCounterUser).Scan(&deposit.OriginalNetwork, &deposit.TokenAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress, &deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}
	deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd

	return &deposit, err
}

// GetLatestExitRoot gets the latest global exit root.
func (p *PostgresStorage) GetLatestExitRoot(ctx context.Context, isRollup bool, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error) {
	if !isRollup {
		return p.GetLatestTrustedExitRoot(ctx, dbTx)
	}

	return p.GetLatestL1SyncedExitRoot(ctx, dbTx)
}

// GetLatestL1SyncedExitRoot gets the latest L1 synced global exit root.
func (p *PostgresStorage) GetLatestL1SyncedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error) {
	var (
		ger         etherman.GlobalExitRoot
		exitRootNum int64
		exitRoots   [][]byte
	)
	const getLatestL1SyncedExitRootSQL = "SELECT block_id, global_exit_root_num, global_exit_root, exit_roots FROM syncv2.exit_root WHERE block_id > 0 ORDER BY global_exit_root_num DESC LIMIT 1"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getLatestL1SyncedExitRootSQL).Scan(&ger.BlockID, &exitRootNum, &ger.GlobalExitRoot, pq.Array(&exitRoots))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gerror.ErrStorageNotFound
		}
		return nil, err
	}
	ger.GlobalExitRootNum = big.NewInt(exitRootNum)
	ger.ExitRoots = []common.Hash{common.BytesToHash(exitRoots[0]), common.BytesToHash(exitRoots[1])}
	return &ger, nil
}

// GetLatestTrustedExitRoot gets the latest trusted global exit root.
func (p *PostgresStorage) GetLatestTrustedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error) {
	var (
		ger         etherman.GlobalExitRoot
		exitRootNum int64
		exitRoots   [][]byte
	)
	const getLatestTrustedExitRootSQL = "SELECT global_exit_root_num, global_exit_root, exit_roots FROM syncv2.exit_root WHERE block_id = 0 ORDER BY global_exit_root_num DESC LIMIT 1"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getLatestTrustedExitRootSQL).Scan(&exitRootNum, &ger.GlobalExitRoot, pq.Array(&exitRoots))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gerror.ErrStorageNotFound
		}
		return nil, err
	}
	ger.GlobalExitRootNum = big.NewInt(exitRootNum)
	ger.ExitRoots = []common.Hash{common.BytesToHash(exitRoots[0]), common.BytesToHash(exitRoots[1])}
	return &ger, nil
}

// GetTokenWrapped gets a specific wrapped token.
func (p *PostgresStorage) GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address, dbTx pgx.Tx) (*etherman.TokenWrapped, error) {
	const getWrappedTokenSQL = "SELECT network_id, orig_net, orig_token_addr, wrapped_token_addr, block_id, name, symbol, decimals FROM syncv2.token_wrapped WHERE orig_net = $1 AND orig_token_addr = $2"
	var token etherman.TokenWrapped
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getWrappedTokenSQL, originalNetwork, originalTokenAddress).Scan(&token.NetworkID, &token.OriginalNetwork, &token.OriginalTokenAddress, &token.WrappedTokenAddress, &token.BlockID, &token.Name, &token.Symbol, &token.Decimals)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}
	return &token, err
}

// GetDepositCountByRoot gets the deposit count by the root.
func (p *PostgresStorage) GetDepositCountByRoot(ctx context.Context, root []byte, network uint8, dbTx pgx.Tx) (uint, error) {
	var depositCount uint
	const getDepositCountByRootSQL = "SELECT deposit_cnt FROM mtv2.root WHERE root = $1 AND network = $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getDepositCountByRootSQL, root, network).Scan(&depositCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, gerror.ErrStorageNotFound
	}
	return depositCount, nil
}

// GetRoot gets root by the deposit count from the merkle tree.
func (p *PostgresStorage) GetRoot(ctx context.Context, depositCnt uint, network uint8, dbTx pgx.Tx) ([]byte, error) {
	var root []byte
	const getRootByDepositCntSQL = "SELECT root FROM mtv2.root WHERE deposit_cnt = $1 AND network = $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getRootByDepositCntSQL, depositCnt, network).Scan(&root)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}
	return root, err
}

// SetRoot store the root with deposit count to the storage.
func (p *PostgresStorage) SetRoot(ctx context.Context, root []byte, depositCnt uint, network uint8, dbTx pgx.Tx) error {
	const setRootSQL = "INSERT INTO mtv2.root (root, deposit_cnt, network) VALUES ($1, $2, $3)"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, setRootSQL, root, depositCnt, network)
	return err
}

// Get gets value of key from the merkle tree.
func (p *PostgresStorage) Get(ctx context.Context, key []byte, dbTx pgx.Tx) ([][]byte, error) {
	const getValueByKeySQL = "SELECT value FROM mtv2.rht WHERE key = $1"
	var data [][]byte
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getValueByKeySQL, key).Scan(pq.Array(&data))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}
	return data, err
}

// Set inserts a key-value pair into the db.
// If record with such a key already exists its assumed that the value is correct,
// because it's a reverse hash table, and the key is a hash of the value
func (p *PostgresStorage) Set(ctx context.Context, key []byte, value [][]byte, dbTx pgx.Tx) error {
	const setNodeSQL = "INSERT INTO mtv2.rht (key, value) VALUES ($1, $2)"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, setNodeSQL, key, pq.Array(value))
	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil
	}
	return err
}

// GetLastDepositCount gets the last deposit count from the merkle tree.
func (p *PostgresStorage) GetLastDepositCount(ctx context.Context, network uint8, dbTx pgx.Tx) (uint, error) {
	var depositCnt int64
	const getLastDepositCountSQL = "SELECT coalesce(MAX(deposit_cnt), -1) FROM mtv2.root WHERE network = $1"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getLastDepositCountSQL, network).Scan(&depositCnt)
	if err != nil {
		return 0, nil
	}
	if depositCnt < 0 {
		return 0, gerror.ErrStorageNotFound
	}
	return uint(depositCnt), nil
}

// ResetMT resets nodes of the Merkle Tree.
func (p *PostgresStorage) ResetMT(ctx context.Context, depositCnt uint, network uint8, dbTx pgx.Tx) error {
	const resetRootSQL = "DELETE FROM mtv2.root WHERE network = $1 AND deposit_cnt > $2"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, resetRootSQL, network, depositCnt)
	return err
}

// GetClaimCount gets the claim count for the destination address.
func (p *PostgresStorage) GetClaimCount(ctx context.Context, destAddr string, dbTx pgx.Tx) (uint64, error) {
	const getClaimCountSQL = "SELECT COUNT(*) FROM syncv2.claim WHERE dest_addr = $1"
	var claimCount uint64
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getClaimCountSQL, common.FromHex(destAddr)).Scan(&claimCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, gerror.ErrStorageNotFound
	}
	return claimCount, err
}

// GetClaims gets the claim list which be smaller than index.
func (p *PostgresStorage) GetClaims(ctx context.Context, destAddr string, limit uint, offset uint, dbTx pgx.Tx) ([]*etherman.Claim, error) {
	const getClaimsSQL = "SELECT index, orig_net, token_addr, amount, dest_addr, block_id, network_id, tx_hash FROM syncv2.claim WHERE dest_addr = $1 ORDER BY block_id DESC LIMIT $2 OFFSET $3"
	rows, err := p.getExecQuerier(dbTx).Query(ctx, getClaimsSQL, common.FromHex(destAddr), limit, offset)
	if err != nil {
		return nil, err
	}
	claims := make([]*etherman.Claim, 0, len(rows.RawValues()))

	for rows.Next() {
		var (
			claim  etherman.Claim
			amount string
		)
		err = rows.Scan(&claim.Index, &claim.OriginalNetwork, &claim.Token, &amount, &claim.DestinationAddress, &claim.BlockID, &claim.NetworkID, &claim.TxHash)
		if err != nil {
			return nil, err
		}
		claim.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		claims = append(claims, &claim)
	}
	return claims, nil
}

// GetDeposits gets the deposit list which be smaller than depositCount.
func (p *PostgresStorage) GetDeposits(ctx context.Context, destAddr string, limit uint, offset uint, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const getDepositsSQL = "SELECT orig_net, token_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata FROM syncv2.deposit as d INNER JOIN syncv2.block as b ON d.network_id = b.network_id AND d.block_id = b.id WHERE dest_addr = $1 ORDER BY d.block_id DESC LIMIT $2 OFFSET $3"
	rows, err := p.getExecQuerier(dbTx).Query(ctx, getDepositsSQL, common.FromHex(destAddr), limit, offset)
	if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))

	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.OriginalNetwork, &deposit.TokenAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress, &deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}

	return deposits, nil
}

// GetDepositCount gets the deposit count for the destination address.
func (p *PostgresStorage) GetDepositCount(ctx context.Context, destAddr string, dbTx pgx.Tx) (uint64, error) {
	const getDepositCountSQL = "SELECT COUNT(*) FROM syncv2.deposit WHERE dest_addr = $1"
	var depositCount uint64
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getDepositCountSQL, common.FromHex(destAddr)).Scan(&depositCount)
	return depositCount, err
}

// ResetTrustedState resets trusted batches from the storage.
func (p *PostgresStorage) ResetTrustedState(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) error {
	const resetTrustedStateSQL = "DELETE FROM syncv2.batch WHERE batch_num > $1"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, resetTrustedStateSQL, batchNumber)
	return err
}

// Update the hash of blocks.
func (p *PostgresStorage) UpdateBlocks(ctx context.Context, networkID uint, blockNum uint64, dbTx pgx.Tx) error {
	const updateBlocksSQL = "UPDATE syncv2.block SET block_hash = $1 WHERE network_id = $2 AND block_num >= $3"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, updateBlocksSQL, common.Hash{}, networkID, blockNum)
	return err
}
