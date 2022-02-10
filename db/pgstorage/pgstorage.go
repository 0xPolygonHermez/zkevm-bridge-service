package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	getLastBlockSQL = "SELECT * FROM state.block ORDER BY block_num DESC LIMIT 1"
	addBlockSQL     = "INSERT INTO state.block (block_num, block_hash, parent_hash, received_at) VALUES ($1, $2, $3, $4)"
	getNodeByKeySQL = "SELECT data FROM %s WHERE hash = $1"
	setNodeByKeySQL = "INSERT INTO %s (hash, data) VALUES ($1, $2)"
)

var (
	contextKeyTableName = "postgres-table-name"
)

// PostgresStorage implements the Storage interface
type PostgresStorage struct {
	db *pgxpool.Pool
}

// NewPostgresStorage creates a new Storage DB
func NewPostgresStorage(user string, password string, host string, port string, name string) (*PostgresStorage, error) {
	db, err := pgxpool.Connect(context.Background(), "postgres://"+user+":"+password+"@"+host+":"+port+"/"+name)
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

// AddBlock adds a new block to the State Store
func (s *PostgresStorage) AddBlock(ctx context.Context, block *etherman.Block) error {
	_, err := s.db.Exec(ctx, addBlockSQL, block.BlockNumber, block.BlockHash.Bytes(), block.ParentHash.Bytes(), block.ReceivedAt)
	return err
}

// Get gets value of key from the merkle tree
func (p *PostgresStorage) Get(ctx context.Context, key []byte) ([]byte, error) {
	var data []byte
	err := p.db.QueryRow(ctx, fmt.Sprintf(getNodeByKeySQL, ctx.Value(contextKeyTableName).(string)), key).Scan(&data)
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
func (p *PostgresStorage) Set(ctx context.Context, key []byte, value []byte) error {
	_, err := p.db.Exec(ctx, fmt.Sprintf(setNodeByKeySQL, ctx.Value(contextKeyTableName).(string)), key, value)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil
		}
		return err
	}
	return nil
}
