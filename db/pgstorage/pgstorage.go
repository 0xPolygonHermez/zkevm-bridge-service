package pgstorage

import (
	"context"
	"errors"

	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	getLastBlockSQL = "SELECT * FROM state.block ORDER BY block_num DESC LIMIT 1"
)

// PostgresStorage implements the Storage interface
type PostgresStorage struct {
	db *pgxpool.Pool
}

// NewPostgresStorage creates a new StateDB
func NewPostgresStorage(db *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{db: db}
}

// GetLastBlock gets the latest block
func (s *PostgresStorage) GetLastBlock(ctx context.Context) (*etherman.Block, error) {
	var block etherman.Block
	err := s.db.QueryRow(ctx, getLastBlockSQL).Scan(&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.ReceivedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, etherman.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return &block, nil
}
