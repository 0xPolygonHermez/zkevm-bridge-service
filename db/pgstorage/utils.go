package pgstorage

import (
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/gobuffalo/packr/v2"
	"github.com/hermeznetwork/hermez-core/log"
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
	pgStorage, err := NewPostgresStorage(cfg, 0)
	if err != nil {
		return err
	}
	defer pgStorage.db.Close()

	// run migrations
	if err := RunMigrationsDown(cfg); err != nil {
		return err
	}
	return RunMigrationsUp(cfg)
}

// NewConfigFromEnv creates config from standard postgres environment variables,
func NewConfigFromEnv() Config {
	maxConns, _ := strconv.Atoi(getEnv("HERMEZBRIDGE_DATABASE_MAXCONNS", "20"))
	return Config{
		User:     getEnv("HERMEZBRIDGE_DATABASE_USER", "test_user"),
		Password: getEnv("HERMEZBRIDGE_DATABASE_PASSWORD", "test_password"),
		Name:     getEnv("HERMEZBRIDGE_DATABASE_NAME", "test_db"),
		Host:     getEnv("HERMEZBRIDGE_DATABASE_HOST", "localhost"),
		Port:     getEnv("HERMEZBRIDGE_DATABASE_PORT", "5433"),
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

// TokenMetadata is a metadata of ERC20 token.
type TokenMetadata struct {
	name     string
	symbol   string
	decimals uint8
}

func getDecodedToken(metadata []byte) (*TokenMetadata, error) {
	//nolint
	args := abi.Arguments{
		{"name", String, false},
		{"symbol", String, false},
		{"decimals", Uint8, false},
	}
	var token map[string]interface{}
	err := args.UnpackIntoMap(token, metadata)
	if err != nil {
		return nil, err
	}

	return &TokenMetadata{
		name:     token["name"].(string),
		symbol:   token["symbol"].(string),
		decimals: token["decimals"].(uint8),
	}, nil
}
