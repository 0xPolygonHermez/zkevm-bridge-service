package pgstorage

import (
	"context"
	"errors"
	"fmt"

	etherman "github.com/0xPolygonHermez/zkevm-bridge-service/ethermanv2"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PostgresStorageV2 implements the Storage interface
type PostgresStorageV2 struct {
	*pgxpool.Pool
}

// getExecQuerier determines which execQuerier to use, dbTx or the main pgxpool
func (p *PostgresStorageV2) getExecQuerier(dbTx pgx.Tx) execQuerier {
	if dbTx != nil {
		return dbTx
	}
	return p
}

// NewPostgresStorageV2 creates a new Storage DB
func NewPostgresStorageV2(cfg Config, dbTxSize uint) (*PostgresStorageV2, error) {
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
	return &PostgresStorageV2{db}, nil
}

// Rollback rollbacks a db transaction.
func (p *PostgresStorageV2) Rollback(ctx context.Context, dbTx pgx.Tx) error {
	if dbTx != nil {
		return dbTx.Rollback(ctx)
	}

	return gerror.ErrNilDBTransaction
}

// Commit commits a db transaction.
func (p *PostgresStorageV2) Commit(ctx context.Context, dbTx pgx.Tx) error {
	if dbTx != nil {
		return dbTx.Commit(ctx)
	}
	return gerror.ErrNilDBTransaction
}

// BeginDBTransaction starts a transaction block.
func (p *PostgresStorageV2) BeginDBTransaction(ctx context.Context) (pgx.Tx, error) {
	return p.Begin(ctx)
}

// GetLastBlock gets the last block.
func (p *PostgresStorageV2) GetLastBlock(ctx context.Context, networkID int, dbTx pgx.Tx) (*etherman.Block, error) {
	var block etherman.Block
	const getLastBlockSQL = "SELECT * FROM syncv2.block where network_id = $1 ORDER BY block_num DESC LIMIT 1"

	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getLastBlockSQL, networkID).Scan(&block.ID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.NetworkID, &block.ReceivedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}

// GetLastBatchNumber gets the last batch number.
func (p *PostgresStorageV2) GetLastBatchNumber(ctx context.Context, dbTx pgx.Tx) (uint64, error) {
	var batchNumber uint64
	const getLastBatchNumberSQL = "SELECT coalesce(max(batch_num),0) as batch FROM syncv2.batch"

	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getLastBatchNumberSQL).Scan(&batchNumber)

	return batchNumber, err
}

// GetBatchByNumber gets the specific batch by the batch number.
func (p *PostgresStorageV2) GetBatchByNumber(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) (*etherman.Batch, error) {
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
func (p *PostgresStorageV2) AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error) {
	var blockID uint64
	const addBlockSQL = "INSERT INTO sync.block (block_num, block_hash, parent_hash, network_id, received_at) VALUES ($1, $2, $3, $4, $5) RETURNING id;"
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, addBlockSQL, block.BlockNumber, block.BlockHash, block.ParentHash, block.NetworkID, block.ReceivedAt).Scan(&blockID)

	return blockID, err
}

// AddBatch adds a new batch to the storage.
func (p *PostgresStorageV2) AddBatch(ctx context.Context, batch *etherman.Batch, dbTx pgx.Tx) error {
	const addBatchSQL = "INSERT INTO syncv2.batch (batch_num, sequencer, raw_tx_data, timestamp, global_exit_root) VALUES ($1, $2, $3, $4, $5)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addBatchSQL, batch.BatchNumber, batch.Coinbase, batch.BatchL2Data, batch.Timestamp, batch.GlobalExitRoot)

	return err
}

// AddVerifiedBatch adds a new verified batch.
func (p *PostgresStorageV2) AddVerifiedBatch(ctx context.Context, verifiedBatch *etherman.VerifiedBatch, dbTx pgx.Tx) error {
	const addVerifiedBatchSQL = "INSERT INTO syncv2.verified_batch (batch_num, aggregator, tx_hash, block_id) VALUES ($1, $2, $3, $4)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addVerifiedBatchSQL, verifiedBatch.BatchNumber, verifiedBatch.Aggregator, verifiedBatch.TxHash, verifiedBatch.BlockID)

	return err
}

// AddExitRoot adds a new ExitRoot to the db.
func (p *PostgresStorageV2) AddExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error {
	const addExitRootSQL = "INSERT INTO syncv2.exit_root (block_id, global_exit_root_num, global_exit_root, mainnet_deposit_cnt, rollup_deposit_cnt) VALUES ($1, $2, $3, $4, $5)"
	mainnetDepositCount, err := p.GetDepositCountByExitRoot(ctx, exitRoot.MainnetExitRoot[:], 0, dbTx)
	if err != nil {
		return err
	}
	rollupDepositCount, err := p.GetDepositCountByExitRoot(ctx, exitRoot.RollupExitRoot[:], 1, dbTx)
	if err != nil {
		return err
	}

	e := p.getExecQuerier(dbTx)
	_, err = e.Exec(ctx, addExitRootSQL, exitRoot.BlockID, exitRoot.GlobalExitRootNum.String(), exitRoot.GlobalExitRoot, mainnetDepositCount, rollupDepositCount)
	return err
}

// GetDepositCountByExitRoot gets the related deposit id by the exit root.
func (p *PostgresStorageV2) GetDepositCountByExitRoot(ctx context.Context, exitRoot []byte, networkID uint, dbTx pgx.Tx) (uint64, error) {
	var depositID uint64
	const getDepositCountSQL = "SELECT deposit_cnt FROM mtv2.root WHERE root = $1 AND network = $2 ORDER BY deposit_cnt DESC LIMIT 1"
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getDepositCountSQL, exitRoot, networkID).Scan(&depositID)
	return depositID, err
}

// AddDeposit adds new deposit to the storage.
func (p *PostgresStorageV2) AddDeposit(ctx context.Context, deposit *etherman.Deposit, dbTx pgx.Tx) error {
	const addDepositSQL = "INSERT INTO syncv2.deposit (network_id, orig_net, token_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addDepositSQL, deposit.NetworkID, deposit.OriginalNetwork, deposit.TokenAddress, deposit.Amount.String(), deposit.DestinationNetwork, deposit.DestinationAddress, deposit.BlockID, deposit.DepositCount, deposit.TxHash, deposit.Metadata)
	return err
}

// AddClaim adds new claim to the storage.
func (p *PostgresStorageV2) AddClaim(ctx context.Context, claim *etherman.Claim, dbTx pgx.Tx) error {
	const addClaimSQL = "INSERT INTO syncv2.claim (network_id, index, orig_net, token_addr, amount, dest_addr, block_id, tx_hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addClaimSQL, claim.NetworkID, claim.OriginalNetwork, claim.Token, claim.Amount.String(), claim.DestinationAddress, claim.BlockID, claim.TxHash)
	return err
}

// GetTokenMetadata gets the metadata of the dedicated token.
func (p *PostgresStorageV2) GetTokenMetadata(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) ([]byte, error) {
	var metadata []byte
	const getMetadataSQL = "SELECT metadata from syncv2.deposit WHERE network_id = $1 AND token_addr = $2 AND dest_net = $3 AND metadata IS NOT NULL LIMIT 1"
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getMetadataSQL, tokenWrapped.OriginalNetwork, tokenWrapped.OriginalTokenAddress, tokenWrapped.NetworkID).Scan(&metadata)

	return metadata, err
}

// AddTokenWrapped adds new wrapped token to the storage.
func (p *PostgresStorageV2) AddTokenWrapped(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) error {
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
	_, err = e.Exec(ctx, addTokenWrappedSQL, tokenWrapped.NetworkID, tokenWrapped.OriginalNetwork, tokenWrapped.OriginalTokenAddress, tokenWrapped.WrappedTokenAddress, tokenWrapped.BlockID, tokenMetadata.name, tokenMetadata.symbol, tokenMetadata.decimals)
	return err
}

// Reset resets the state to a block for the given DB tx.
func (p *PostgresStorageV2) Reset(ctx context.Context, blockNumber uint64, dbTx pgx.Tx) error {
	const resetSQL = "DELETE FROM syncv2.block WHERE block_num > $1"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, resetSQL, blockNumber)
	return err
}

// GetPreviousBlock gets the offset previous L1 block respect to latest.
func (p *PostgresStorageV2) GetPreviousBlock(ctx context.Context, networkID uint, offset uint64, dbTx pgx.Tx) (*etherman.Block, error) {
	var block etherman.Block
	const getPreviousBlockSQL = "SELECT block_num, block_hash, parent_hash, network_id, received_at FROM syncv2.block WHERE network_id = $1 ORDER BY block_num DESC LIMIT 1 OFFSET $2"
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getPreviousBlockSQL, networkID, offset).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.NetworkID, &block.ReceivedAt)
	return &block, err
}

// GetNumberDeposits gets the number of  deposits.
func (p *PostgresStorageV2) GetNumberDeposits(ctx context.Context, networkID uint, blockNumber uint64, dbTx pgx.Tx) (uint64, error) {
	var nDeposits int64
	const getNumDepositsSQL = "SELECT coalesce(MAX(deposit_cnt), -1) FROM syncv2.deposit WHERE network_id = $1 AND block_num <= $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getNumDepositsSQL, networkID, blockNumber).Scan(&nDeposits)
	return uint64(nDeposits + 1), err
}

// GetNextForcedBatches gets the next forced batches from the queue.
func (p *PostgresStorageV2) GetNextForcedBatches(ctx context.Context, nextForcedBatches int, dbTx pgx.Tx) ([]etherman.ForcedBatch, error) {
	const getNextForcedBatchesSQL = "SELECT forced_batch_num, global_exit_root, timestamp, raw_txs_data, coinbase, batch_num, block_num FROM syncv2.forced_batch WHERE batch_num IS NULL LIMIT $1"
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
		err := rows.Scan(&forcedBatch.ForcedBatchNumber, &forcedBatch.GlobalExitRoot, &forcedBatch.ForcedAt, &forcedBatch.RawTxsData, &forcedBatch.Sequencer, &forcedBatch.BatchNumber, &forcedBatch.BlockNumber)
		if err != nil {
			return nil, err
		}
		batches = append(batches, forcedBatch)
	}

	return batches, nil
}

// AddBatchNumberInForcedBatch updates the forced_batch table with the batchNumber.
func (p *PostgresStorageV2) AddBatchNumberInForcedBatch(ctx context.Context, forceBatchNumber, batchNumber uint64, dbTx pgx.Tx) error {
	const addBatchNumberInForcedBatchSQL = "UPDATE syncv2.forced_batch SET batch_num = $2 WHERE forced_batch_num = $1"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addBatchNumberInForcedBatchSQL, forceBatchNumber, batchNumber)
	return err
}

// AddForcedBatch adds a new ForcedBatch to the db.
func (p *PostgresStorageV2) AddForcedBatch(ctx context.Context, forcedBatch *etherman.ForcedBatch, dbTx pgx.Tx) error {
	const addForcedBatchSQL = "INSERT INTO syncv2.forced_batch (forced_batch_num, global_exit_root, timestamp, raw_txs_data, coinbase, batch_num, block_num) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, addForcedBatchSQL, forcedBatch.ForcedBatchNumber, forcedBatch.GlobalExitRoot.String(), forcedBatch.ForcedAt, forcedBatch.RawTxsData, forcedBatch.Sequencer.String(), forcedBatch.BatchNumber, forcedBatch.BlockNumber)
	return err
}

// AddTrustedGlobalExitRoot adds new global exit root which comes from the trusted sequencer.
func (p *PostgresStorageV2) AddTrustedGlobalExitRoot(ctx context.Context, ger common.Hash) error {
	return nil
}
