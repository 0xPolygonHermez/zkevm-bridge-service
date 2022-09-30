package pgstorage

import (
	"os"
	"strconv"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/gobuffalo/packr/v2"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	migrate "github.com/rubenv/sql-migrate"
)

// RunMigrationsUp migrate up.
func RunMigrationsUp(cfg Config) error {
	return runMigrations(cfg, migrate.Up)
}

// RunMigrationsDown migrate down.
func RunMigrationsDown(cfg Config) error {
	return runMigrations(cfg, migrate.Down)
}

// runMigrations will execute pending migrations if needed to keep
// the database updated with the latest changes in either direction of up or down.
func runMigrations(cfg Config, direction migrate.MigrationDirection) error {
	c, err := pgx.ParseConfig("postgres://" + cfg.User + ":" + cfg.Password + "@" + cfg.Host + ":" + cfg.Port + "/" + cfg.Name)
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(*c)

	var migrations = &migrate.PackrMigrationSource{Box: packr.New("hermez-db-migrations", "./migrations")}
	nMigrations, err := migrate.Exec(db, "postgres", migrations, direction)
	if err != nil {
		return err
	}

	log.Info("successfully ran ", nMigrations, " migrations")
	return nil
}

// InitOrReset will initializes the db running the migrations or
// will reset all the known data and rerun the migrations
func InitOrReset(cfg Config) error {
	// connect to database
	_, err := NewPostgresStorage(cfg, 0)
	if err != nil {
		return err
	}

	// run migrations
	if err := RunMigrationsDown(cfg); err != nil {
		return err
	}
	return RunMigrationsUp(cfg)
}

// NewConfigFromEnv creates config from standard postgres environment variables,
func NewConfigFromEnv() Config {
	maxConns, _ := strconv.Atoi(getEnv("ZKEVM_BRIDGE_DATABASE_MAXCONNS", "20"))
	return Config{
		User:     getEnv("ZKEVM_BRIDGE_DATABASE_USER", "test_user"),
		Password: getEnv("ZKEVM_BRIDGE_DATABASE_PASSWORD", "test_password"),
		Name:     getEnv("ZKEVM_BRIDGE_DATABASE_NAME", "test_db"),
		Host:     getEnv("ZKEVM_BRIDGE_DATABASE_HOST", "localhost"),
		Port:     getEnv("ZKEVM_BRIDGE_DATABASE_PORT", "5435"),
		MaxConns: maxConns,
	}
}

func getEnv(key string, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if exists {
		return value
	}
	return defaultValue
}

//nolint
var (
	String, _ = abi.NewType("string", "", nil)
	Uint8, _  = abi.NewType("uint8", "", nil)
)

func getDecodedToken(metadata []byte) (*etherman.TokenMetadata, error) {
	//nolint
	args := abi.Arguments{
		{"name", String, false},
		{"symbol", String, false},
		{"decimals", Uint8, false},
	}
	token := make(map[string]interface{})
	err := args.UnpackIntoMap(token, metadata)
	if err != nil {
		return nil, err
	}

	return &etherman.TokenMetadata{
		Name:     token["name"].(string),
		Symbol:   token["symbol"].(string),
		Decimals: token["decimals"].(uint8),
	}, nil
}
