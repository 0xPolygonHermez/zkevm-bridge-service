package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
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
	const getLastBlockSQL = "SELECT id, block_num, block_hash, parent_hash, network_id, received_at FROM sync.block where network_id = $1 ORDER BY block_num DESC LIMIT 1"

	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getLastBlockSQL, networkID).Scan(&block.ID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.NetworkID, &block.ReceivedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}

	return &block, err
}

// AddBlock adds a new block to the storage.
func (p *PostgresStorage) AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error) {
	var blockID uint64
	const addBlockSQL = `WITH block_id AS 
		(INSERT INTO sync.block (block_num, block_hash, parent_hash, network_id, received_at) 
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT (block_hash) DO NOTHING RETURNING id)
		SELECT * from block_id
		UNION ALL
		SELECT id FROM sync.block WHERE block_hash = $2;`
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, addBlockSQL, block.BlockNumber, block.BlockHash, block.ParentHash, block.NetworkID, block.ReceivedAt).Scan(&blockID)

	if err == pgx.ErrNoRows {
		err = nil
	}

	return blockID, err
}

// AddGlobalExitRoot adds a new ExitRoot to the db.
func (p *PostgresStorage) AddGlobalExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error {
	const addExitRootSQL = "INSERT INTO sync.exit_root (block_id, global_exit_root, exit_roots) VALUES ($1, $2, $3)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addExitRootSQL, exitRoot.BlockID, exitRoot.GlobalExitRoot, pq.Array([][]byte{exitRoot.ExitRoots[0][:], exitRoot.ExitRoots[1][:]}))
	return err
}

// AddDeposit adds new deposit to the storage.
func (p *PostgresStorage) AddDeposit(ctx context.Context, deposit *etherman.Deposit, dbTx pgx.Tx) (uint64, error) {
	const addDepositSQL = `
	INSERT INTO sync.deposit (
		leaf_type, network_id, orig_net, orig_addr, amount, dest_net, dest_addr, block_id, deposit_cnt, tx_hash, metadata, origin_rollup_id
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`
	e := p.getExecQuerier(dbTx)
	var depositID uint64
	err := e.QueryRow(
		ctx,
		addDepositSQL,
		deposit.LeafType,
		deposit.NetworkID,
		deposit.OriginalNetwork,
		deposit.OriginalAddress,
		deposit.Amount.String(),
		deposit.DestinationNetwork,
		deposit.DestinationAddress,
		deposit.BlockID,
		deposit.DepositCount,
		deposit.TxHash,
		deposit.Metadata,
		deposit.OriginRollupID,
	).Scan(&depositID)
	return depositID, err
}

// AddClaim adds new claim to the storage.
func (p *PostgresStorage) AddClaim(ctx context.Context, claim *etherman.Claim, dbTx pgx.Tx) error {
	const addClaimSQL = "INSERT INTO sync.claim (network_id, index, orig_net, orig_addr, amount, dest_addr, block_id, tx_hash, rollup_index, mainnet_flag) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, addClaimSQL, claim.NetworkID, claim.Index, claim.OriginalNetwork, claim.OriginalAddress, claim.Amount.String(), claim.DestinationAddress, claim.BlockID, claim.TxHash, claim.RollupIndex, claim.MainnetFlag)
	return err
}

// GetTokenMetadata gets the metadata of the dedicated token.
func (p *PostgresStorage) GetTokenMetadata(ctx context.Context, networkID, destNet uint, originalTokenAddr common.Address, dbTx pgx.Tx) ([]byte, error) {
	var metadata []byte
	const getMetadataSQL = "SELECT metadata from sync.deposit WHERE network_id = $1 AND orig_addr = $2 AND dest_net = $3 AND metadata IS NOT NULL LIMIT 1"
	e := p.getExecQuerier(dbTx)
	err := e.QueryRow(ctx, getMetadataSQL, networkID, originalTokenAddr, destNet).Scan(&metadata)
	return metadata, err
}

// AddTokenWrapped adds new wrapped token to the storage.
func (p *PostgresStorage) AddTokenWrapped(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) error {
	metadata, err := p.GetTokenMetadata(ctx, tokenWrapped.OriginalNetwork, tokenWrapped.NetworkID, tokenWrapped.OriginalTokenAddress, dbTx)
	var tokenMetadata *etherman.TokenMetadata
	if err != nil {
		if err != pgx.ErrNoRows {
			return err
		}
		// if err == pgx.ErrNoRows, this is due to missing the related deposit in the opposite network in fast sync mode.
		// ref: https://github.com/0xPolygonHermez/zkevm-bridge-service/issues/230
		tokenMetadata = &etherman.TokenMetadata{}
	} else {
		tokenMetadata, err = getDecodedToken(metadata)
		if err != nil {
			return err
		}
	}

	const addTokenWrappedSQL = "INSERT INTO sync.token_wrapped (network_id, orig_net, orig_token_addr, wrapped_token_addr, block_id, name, symbol, decimals) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	e := p.getExecQuerier(dbTx)
	_, err = e.Exec(ctx, addTokenWrappedSQL, tokenWrapped.NetworkID, tokenWrapped.OriginalNetwork, tokenWrapped.OriginalTokenAddress, tokenWrapped.WrappedTokenAddress, tokenWrapped.BlockID, tokenMetadata.Name, tokenMetadata.Symbol, tokenMetadata.Decimals)
	return err
}

// Reset resets the state to a block for the given DB tx.
func (p *PostgresStorage) Reset(ctx context.Context, blockNumber uint64, networkID uint, dbTx pgx.Tx) error {
	const resetSQL = "DELETE FROM sync.block WHERE block_num > $1 AND network_id = $2"
	e := p.getExecQuerier(dbTx)
	_, err := e.Exec(ctx, resetSQL, blockNumber, networkID)
	return err
}

// GetPreviousBlock gets the offset previous L1 block respect to latest.
func (p *PostgresStorage) GetPreviousBlock(ctx context.Context, networkID uint, offset uint64, dbTx pgx.Tx) (*etherman.Block, error) {
	var block etherman.Block
	const getPreviousBlockSQL = "SELECT block_num, block_hash, parent_hash, network_id, received_at FROM sync.block WHERE network_id = $1 ORDER BY block_num DESC LIMIT 1 OFFSET $2"
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
	const getNumDepositsSQL = "SELECT coalesce(MAX(deposit_cnt), -1) FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id WHERE d.network_id = $1 AND b.block_num <= $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getNumDepositsSQL, networkID, blockNumber).Scan(&nDeposits)
	return uint64(nDeposits + 1), err
}

// AddTrustedGlobalExitRoot adds new global exit root which comes from the trusted sequencer.
func (p *PostgresStorage) AddTrustedGlobalExitRoot(ctx context.Context, trustedExitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) (bool, error) {
	const addTrustedGerSQL = `
		INSERT INTO sync.exit_root (block_id, global_exit_root, exit_roots) 
		VALUES (0, $1, $2)
		ON CONFLICT ON CONSTRAINT UC DO NOTHING;`
	res, err := p.getExecQuerier(dbTx).Exec(ctx, addTrustedGerSQL, trustedExitRoot.GlobalExitRoot, pq.Array([][]byte{trustedExitRoot.ExitRoots[0][:], trustedExitRoot.ExitRoots[1][:]}))
	return res.RowsAffected() > 0, err
}

// GetClaim gets a specific claim from the storage.
func (p *PostgresStorage) GetClaim(ctx context.Context, depositCount, originNetworkID, destNetworkID uint, dbTx pgx.Tx) (*etherman.Claim, error) {
	var (
		claim  etherman.Claim
		amount string
	)

	const getClaimSQLOriginRollupDestMainnet = `
	SELECT index, orig_net, orig_addr, amount, dest_addr, block_id, network_id, tx_hash, rollup_index 
	FROM sync.claim 
	WHERE index = $1 AND network_id = $2 AND mainnet_flag;
	`

	const getClaimSQLOriginMainnetDestMainnet = `
	SELECT index, orig_net, orig_addr, amount, dest_addr, block_id, network_id, tx_hash, rollup_index 
	FROM sync.claim 
	WHERE index = $1 AND network_id = 0 AND mainnet_flag AND rollup_index = 0;
	`

	const getClaimSQLDestRollup = `
	SELECT index, orig_net, orig_addr, amount, dest_addr, block_id, network_id, tx_hash, rollup_index 
	FROM sync.claim 
	WHERE index = $1 AND network_id = $2 AND NOT mainnet_flag AND rollup_index + 1 = $3;
	`

	var row pgx.Row
	if destNetworkID == 0 {
		claim.MainnetFlag = true

		if originNetworkID != 0 {
			row = p.getExecQuerier(dbTx).
				QueryRow(ctx, getClaimSQLOriginRollupDestMainnet, depositCount, originNetworkID)
		} else {
			row = p.getExecQuerier(dbTx).
				QueryRow(ctx, getClaimSQLOriginMainnetDestMainnet, depositCount)
		}
	} else {
		row = p.getExecQuerier(dbTx).
			QueryRow(ctx, getClaimSQLDestRollup, depositCount, originNetworkID, destNetworkID)
	}

	err := row.Scan(
		&claim.Index,
		&claim.OriginalNetwork,
		&claim.OriginalAddress,
		&amount,
		&claim.DestinationAddress,
		&claim.BlockID,
		&claim.NetworkID,
		&claim.TxHash,
		&claim.RollupIndex,
	)
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
	const getDepositSQL = `
	SELECT leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, origin_rollup_id 
	FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id 
	WHERE d.network_id = $1 AND deposit_cnt = $2`
	err := p.getExecQuerier(dbTx).
		QueryRow(ctx, getDepositSQL, networkID, depositCounterUser).
		Scan(
			&deposit.LeafType,
			&deposit.OriginalNetwork,
			&deposit.OriginalAddress,
			&amount,
			&deposit.DestinationNetwork,
			&deposit.DestinationAddress,
			&deposit.DepositCount,
			&deposit.BlockID,
			&deposit.BlockNumber,
			&deposit.NetworkID,
			&deposit.TxHash,
			&deposit.Metadata,
			&deposit.ReadyForClaim,
			&deposit.OriginRollupID,
		)
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
		ger       etherman.GlobalExitRoot
		exitRoots [][]byte
	)
	const getLatestL1SyncedExitRootSQL = "SELECT block_id, global_exit_root, exit_roots FROM sync.exit_root WHERE block_id > 0 ORDER BY id DESC LIMIT 1"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getLatestL1SyncedExitRootSQL).Scan(&ger.BlockID, &ger.GlobalExitRoot, pq.Array(&exitRoots))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &ger, gerror.ErrStorageNotFound
		}
		return nil, err
	}
	ger.ExitRoots = []common.Hash{common.BytesToHash(exitRoots[0]), common.BytesToHash(exitRoots[1])}
	return &ger, nil
}

// GetLatestTrustedExitRoot gets the latest trusted global exit root.
func (p *PostgresStorage) GetLatestTrustedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error) {
	var (
		ger       etherman.GlobalExitRoot
		exitRoots [][]byte
	)
	const getLatestTrustedExitRootSQL = "SELECT global_exit_root, exit_roots FROM sync.exit_root WHERE block_id = 0 ORDER BY id DESC LIMIT 1"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getLatestTrustedExitRootSQL).Scan(&ger.GlobalExitRoot, pq.Array(&exitRoots))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gerror.ErrStorageNotFound
		}
		return nil, err
	}
	ger.ExitRoots = []common.Hash{common.BytesToHash(exitRoots[0]), common.BytesToHash(exitRoots[1])}
	return &ger, nil
}

// GetTokenWrapped gets a specific wrapped token.
func (p *PostgresStorage) GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address, dbTx pgx.Tx) (*etherman.TokenWrapped, error) {
	const getWrappedTokenSQL = "SELECT network_id, orig_net, orig_token_addr, wrapped_token_addr, block_id, name, symbol, decimals FROM sync.token_wrapped WHERE orig_net = $1 AND orig_token_addr = $2"

	var token etherman.TokenWrapped
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getWrappedTokenSQL, originalNetwork, originalTokenAddress).Scan(&token.NetworkID, &token.OriginalNetwork, &token.OriginalTokenAddress, &token.WrappedTokenAddress, &token.BlockID, &token.Name, &token.Symbol, &token.Decimals)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}

	// this is due to missing the related deposit in the opposite network in fast sync mode.
	// ref: https://github.com/0xPolygonHermez/zkevm-bridge-service/issues/230
	if token.Symbol == "" {
		metadata, err := p.GetTokenMetadata(ctx, token.OriginalNetwork, token.NetworkID, token.OriginalTokenAddress, dbTx)
		var tokenMetadata *etherman.TokenMetadata
		if err != nil {
			if err != pgx.ErrNoRows {
				return nil, err
			}
		} else {
			tokenMetadata, err = getDecodedToken(metadata)
			if err != nil {
				return nil, err
			}
			updateWrappedTokenSQL := "UPDATE sync.token_wrapped SET name = $3, symbol = $4, decimals = $5  WHERE orig_net = $1 AND orig_token_addr = $2" //nolint: gosec
			_, err = p.getExecQuerier(dbTx).Exec(ctx, updateWrappedTokenSQL, originalNetwork, originalTokenAddress, tokenMetadata.Name, tokenMetadata.Symbol, tokenMetadata.Decimals)
			if err != nil {
				return nil, err
			}
			token.Name, token.Symbol, token.Decimals = tokenMetadata.Name, tokenMetadata.Symbol, tokenMetadata.Decimals
		}
	}
	return &token, err
}

// GetDepositCountByRoot gets the deposit count by the root.
func (p *PostgresStorage) GetDepositCountByRoot(ctx context.Context, root []byte, network uint8, dbTx pgx.Tx) (uint, error) {
	var depositCount uint
	const getDepositCountByRootSQL = "SELECT sync.deposit.deposit_cnt FROM mt.root INNER JOIN sync.deposit ON sync.deposit.id = mt.root.deposit_id WHERE mt.root.root = $1 AND mt.root.network = $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getDepositCountByRootSQL, root, network).Scan(&depositCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, gerror.ErrStorageNotFound
	}
	return depositCount, nil
}

// CheckIfRootExists checks that the root exists on the db.
func (p *PostgresStorage) CheckIfRootExists(ctx context.Context, root []byte, network uint8, dbTx pgx.Tx) (bool, error) {
	var count uint
	const getDepositCountByRootSQL = "SELECT count(*) FROM mt.root WHERE root = $1 AND network = $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getDepositCountByRootSQL, root, network).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, gerror.ErrStorageNotFound
	}
	if count == 0 {
		return false, nil
	}
	return true, nil
}

// GetRoot gets root by the deposit count from the merkle tree.
func (p *PostgresStorage) GetRoot(ctx context.Context, depositCnt uint, network uint, dbTx pgx.Tx) ([]byte, error) {
	var root []byte
	const getRootByDepositCntSQL = "SELECT root FROM mt.root inner join sync.deposit on mt.root.deposit_id = sync.deposit.id WHERE sync.deposit.deposit_cnt = $1 AND network = $2"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getRootByDepositCntSQL, depositCnt, network).Scan(&root)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}
	return root, err
}

// SetRoot store the root with deposit count to the storage.
func (p *PostgresStorage) SetRoot(ctx context.Context, root []byte, depositID uint64, network uint, dbTx pgx.Tx) error {
	const setRootSQL = "INSERT INTO mt.root (root, deposit_id, network) VALUES ($1, $2, $3);"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, setRootSQL, root, depositID, network)
	return err
}

// Get gets value of key from the merkle tree.
func (p *PostgresStorage) Get(ctx context.Context, key []byte, dbTx pgx.Tx) ([][]byte, error) {
	const getValueByKeySQL = "SELECT value FROM mt.rht WHERE key = $1"
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
func (p *PostgresStorage) Set(ctx context.Context, key []byte, value [][]byte, depositID uint64, dbTx pgx.Tx) error {
	const setNodeSQL = "INSERT INTO mt.rht (deposit_id, key, value) VALUES ($1, $2, $3)"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, setNodeSQL, depositID, key, pq.Array(value))
	return err
}

// BulkSet is similar to Set, but it inserts multiple key-value pairs into the db.
func (p *PostgresStorage) BulkSet(ctx context.Context, rows [][]interface{}, dbTx pgx.Tx) error {
	_, err := p.getExecQuerier(dbTx).CopyFrom(ctx, pgx.Identifier{"mt", "rht"}, []string{"key", "value", "deposit_id"}, pgx.CopyFromRows(rows))
	return err
}

// AddRollupExitLeaves iinserts multiple entries into the db.
func (p *PostgresStorage) AddRollupExitLeaves(ctx context.Context, rows [][]interface{}, dbTx pgx.Tx) error {
	_, err := p.getExecQuerier(dbTx).CopyFrom(ctx, pgx.Identifier{"mt", "rollup_exit"}, []string{"leaf", "rollup_id", "root", "block_id"}, pgx.CopyFromRows(rows))
	return err
}

// GetRollupExitLeavesByRoot gets the leaves of the rollupExitTree given a root
func (p *PostgresStorage) GetRollupExitLeavesByRoot(ctx context.Context, root common.Hash, dbTx pgx.Tx) ([]etherman.RollupExitLeaf, error) {
	const getLeavesSQL = "SELECT id, leaf, rollup_id, root, block_id FROM mt.rollup_exit WHERE root = $1 ORDER BY rollup_id ASC"
	rows, err := p.getExecQuerier(dbTx).Query(ctx, getLeavesSQL, root)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	} else if err != nil {
		return nil, err
	}
	leaves := make([]etherman.RollupExitLeaf, 0, len(rows.RawValues()))

	for rows.Next() {
		var leaf etherman.RollupExitLeaf
		err = rows.Scan(&leaf.ID, &leaf.Leaf, &leaf.RollupId, &leaf.Root, &leaf.BlockID)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, leaf)
	}
	return leaves, nil
}

// IsRollupExitRoot checks if db contains the root
func (p *PostgresStorage) IsRollupExitRoot(ctx context.Context, root common.Hash, dbTx pgx.Tx) (bool, error) {
	const getLeavesSQL = "SELECT count(*) FROM mt.rollup_exit WHERE root = $1"
	var count int
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getLeavesSQL, root).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, gerror.ErrStorageNotFound
	} else if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

// IsLxLyActivated checks in db if LxLy is activated
func (p *PostgresStorage) IsLxLyActivated(ctx context.Context, dbTx pgx.Tx) (bool, error) {
	const getLeavesSQL = "SELECT count(*) FROM mt.rollup_exit"
	var count int
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getLeavesSQL).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, gerror.ErrStorageNotFound
	} else if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

// GetLatestRollupExitLeaves gets the latest leaves of the rollupExitTree
func (p *PostgresStorage) GetLatestRollupExitLeaves(ctx context.Context, dbTx pgx.Tx) ([]etherman.RollupExitLeaf, error) {
	const getLeavesSQL = `SELECT distinct re.id, re.leaf, re.rollup_id, re.root, re.block_id
		FROM mt.rollup_exit re
		INNER JOIN
			(SELECT distinct rollup_id, MAX(id) AS maxid
			FROM mt.rollup_exit
			GROUP BY rollup_id) groupedre
		ON re.id = groupedre.maxid
		ORDER BY rollup_id asc;
	`
	rows, err := p.getExecQuerier(dbTx).Query(ctx, getLeavesSQL)
	if err != nil {
		return nil, err
	}
	leaves := make([]etherman.RollupExitLeaf, 0, len(rows.RawValues()))

	for rows.Next() {
		var leaf etherman.RollupExitLeaf
		err = rows.Scan(&leaf.ID, &leaf.Leaf, &leaf.RollupId, &leaf.Root, &leaf.BlockID)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, leaf)
	}
	return leaves, nil
}

// GetLastDepositCount gets the last deposit count from the merkle tree.
func (p *PostgresStorage) GetLastDepositCount(ctx context.Context, network uint, dbTx pgx.Tx) (uint, error) {
	var depositCnt int64
	const getLastDepositCountSQL = "SELECT coalesce(MAX(deposit_cnt), -1) FROM sync.deposit WHERE id = (SELECT coalesce(MAX(deposit_id), -1) FROM mt.root WHERE network = $1)"
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getLastDepositCountSQL, network).Scan(&depositCnt)
	if err != nil {
		return 0, nil
	}
	if depositCnt < 0 {
		return 0, gerror.ErrStorageNotFound
	}
	return uint(depositCnt), nil
}

// GetClaimCount gets the claim count for the destination address.
func (p *PostgresStorage) GetClaimCount(ctx context.Context, destAddr string, dbTx pgx.Tx) (uint64, error) {
	const getClaimCountSQL = "SELECT COUNT(*) FROM sync.claim WHERE dest_addr = $1"
	var claimCount uint64
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getClaimCountSQL, common.FromHex(destAddr)).Scan(&claimCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, gerror.ErrStorageNotFound
	}
	return claimCount, err
}

// GetClaims gets the claim list which be smaller than index.
func (p *PostgresStorage) GetClaims(ctx context.Context, destAddr string, limit uint, offset uint, dbTx pgx.Tx) ([]*etherman.Claim, error) {
	const getClaimsSQL = "SELECT index, orig_net, orig_addr, amount, dest_addr, block_id, network_id, tx_hash, rollup_index, mainnet_flag FROM sync.claim WHERE dest_addr = $1 ORDER BY block_id DESC LIMIT $2 OFFSET $3"
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
		err = rows.Scan(&claim.Index, &claim.OriginalNetwork, &claim.OriginalAddress, &amount, &claim.DestinationAddress, &claim.BlockID, &claim.NetworkID, &claim.TxHash, &claim.RollupIndex, &claim.MainnetFlag)
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
	const getDepositsSQL = `
	SELECT leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, origin_rollup_id
	FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id 
	WHERE dest_addr = $1 ORDER BY d.block_id DESC, d.deposit_cnt DESC LIMIT $2 OFFSET $3
	`
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
		err = rows.Scan(
			&deposit.LeafType,
			&deposit.OriginalNetwork,
			&deposit.OriginalAddress,
			&amount,
			&deposit.DestinationNetwork,
			&deposit.DestinationAddress,
			&deposit.DepositCount,
			&deposit.BlockID,
			&deposit.BlockNumber,
			&deposit.NetworkID,
			&deposit.TxHash,
			&deposit.Metadata,
			&deposit.ReadyForClaim,
			&deposit.OriginRollupID,
		)
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
	const getDepositCountSQL = "SELECT COUNT(*) FROM sync.deposit WHERE dest_addr = $1"
	var depositCount uint64
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getDepositCountSQL, common.FromHex(destAddr)).Scan(&depositCount)
	return depositCount, err
}

// UpdateBlocksForTesting updates the hash of blocks.
func (p *PostgresStorage) UpdateBlocksForTesting(ctx context.Context, networkID uint, blockNum uint64, dbTx pgx.Tx) error {
	const updateBlocksSQL = "UPDATE sync.block SET block_hash = SUBSTRING(block_hash FROM 1 FOR LENGTH(block_hash)-1) || '\x61' WHERE network_id = $1 AND block_num >= $2"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, updateBlocksSQL, networkID, blockNum)
	return err
}

// UpdateL1DepositsStatus updates the ready_for_claim status of L1 deposits.
func (p *PostgresStorage) UpdateL1DepositsStatus(ctx context.Context, exitRoot []byte, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const updateDepositsStatusSQL = `UPDATE sync.deposit SET ready_for_claim = true 
		WHERE deposit_cnt <=
			(SELECT sync.deposit.deposit_cnt FROM mt.root INNER JOIN sync.deposit ON sync.deposit.id = mt.root.deposit_id WHERE mt.root.root = $1 AND mt.root.network = 0) 
			AND network_id = 0 AND ready_for_claim = false
			RETURNING leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, network_id, tx_hash, metadata, ready_for_claim;`
	rows, err := p.getExecQuerier(dbTx).Query(ctx, updateDepositsStatusSQL, exitRoot)
	if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))
	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress, &deposit.DepositCount, &deposit.BlockID, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}
	return deposits, nil
}

// UpdateL2DepositsStatus updates the ready_for_claim status of L2 deposits.
func (p *PostgresStorage) UpdateL2DepositsStatus(ctx context.Context, exitRoot []byte, rollupID, networkID uint, dbTx pgx.Tx) error {
	const updateDepositsStatusSQL = `UPDATE sync.deposit SET ready_for_claim = true
		WHERE deposit_cnt <=
		(SELECT sync.deposit.deposit_cnt FROM mt.root INNER JOIN sync.deposit ON sync.deposit.id = mt.root.deposit_id WHERE mt.root.root = (select leaf from mt.rollup_exit where root = $1 and rollup_id = $2) AND mt.root.network = $3)
			AND network_id = $3 AND ready_for_claim = false;`
	_, err := p.getExecQuerier(dbTx).Exec(ctx, updateDepositsStatusSQL, exitRoot, rollupID, networkID)
	return err
}

// AddClaimTx adds a claim monitored transaction to the storage.
func (p *PostgresStorage) AddClaimTx(ctx context.Context, mTx ctmtypes.MonitoredTx, dbTx pgx.Tx) error {
	const addMonitoredTxSQL = `INSERT INTO sync.monitored_txs 
		(deposit_id, from_addr, to_addr, nonce, value, data, gas, status, history, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := p.getExecQuerier(dbTx).Exec(ctx, addMonitoredTxSQL, mTx.DepositID, mTx.From, mTx.To, mTx.Nonce, mTx.Value.String(), mTx.Data, mTx.Gas, mTx.Status, pq.Array(mTx.HistoryHashSlice()), time.Now().UTC(), time.Now().UTC())
	return err
}

// UpdateClaimTx updates a claim monitored transaction in the storage.
func (p *PostgresStorage) UpdateClaimTx(ctx context.Context, mTx ctmtypes.MonitoredTx, dbTx pgx.Tx) error {
	const updateMonitoredTxSQL = `UPDATE sync.monitored_txs 
		SET from_addr = $2
		, to_addr = $3
		, nonce = $4
		, value = $5
		, data = $6
		, gas = $7
		, status = $8
		, history = $9
		, updated_at = $10
		WHERE deposit_id = $1`
	_, err := p.getExecQuerier(dbTx).Exec(ctx, updateMonitoredTxSQL, mTx.DepositID, mTx.From, mTx.To, mTx.Nonce, mTx.Value.String(), mTx.Data, mTx.Gas, mTx.Status, pq.Array(mTx.HistoryHashSlice()), time.Now().UTC())
	return err
}

// GetClaimTxsByStatus gets the monitored transactions by status.
func (p *PostgresStorage) GetClaimTxsByStatus(ctx context.Context, statuses []ctmtypes.MonitoredTxStatus, dbTx pgx.Tx) ([]ctmtypes.MonitoredTx, error) {
	const getMonitoredTxsSQL = "SELECT deposit_id, from_addr, to_addr, nonce, value, data, gas, status, history, created_at, updated_at FROM sync.monitored_txs WHERE status = ANY($1) ORDER BY created_at ASC"
	rows, err := p.getExecQuerier(dbTx).Query(ctx, getMonitoredTxsSQL, pq.Array(statuses))
	if errors.Is(err, pgx.ErrNoRows) {
		return []ctmtypes.MonitoredTx{}, nil
	} else if err != nil {
		return nil, err
	}

	mTxs := make([]ctmtypes.MonitoredTx, 0, len(rows.RawValues()))
	for rows.Next() {
		var (
			value   string
			history [][]byte
		)
		mTx := ctmtypes.MonitoredTx{}
		err = rows.Scan(&mTx.DepositID, &mTx.From, &mTx.To, &mTx.Nonce, &value, &mTx.Data, &mTx.Gas, &mTx.Status, pq.Array(&history), &mTx.CreatedAt, &mTx.UpdatedAt)
		if err != nil {
			return mTxs, err
		}
		mTx.Value, _ = new(big.Int).SetString(value, 10) //nolint:gomnd
		mTx.History = make(map[common.Hash]bool)
		for _, h := range history {
			mTx.History[common.BytesToHash(h)] = true
		}
		mTxs = append(mTxs, mTx)
	}

	return mTxs, nil
}

// UpdateDepositsStatusForTesting updates the ready_for_claim status of all deposits for testing.
func (p *PostgresStorage) UpdateDepositsStatusForTesting(ctx context.Context, dbTx pgx.Tx) error {
	const updateDepositsStatusSQL = "UPDATE sync.deposit SET ready_for_claim = true;"
	_, err := p.getExecQuerier(dbTx).Exec(ctx, updateDepositsStatusSQL)
	return err
}
