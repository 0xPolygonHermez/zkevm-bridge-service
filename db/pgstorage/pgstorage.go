package pgstorage

import (
	"context"
	"errors"

	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	// ErrNotFound is used when the object is not found
	ErrNotFound = errors.New("Not found")
)

const (
	getLastBlockSQL = "SELECT * FROM state.block ORDER BY block_num DESC LIMIT 1"
	addBlockSQL     = "INSERT INTO state.block (block_num, block_hash, parent_hash, received_at) VALUES ($1, $2, $3, $4)"
)

// PostgresStorage implements the Storage interface
type PostgresStorage struct {
	db *pgxpool.Pool
}

// NewPostgresStorage creates a new StateDB
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
		return nil, ErrNotFound
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
