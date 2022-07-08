package pgstorage

import (
	"context"
	"os"
	"strconv"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/gobuffalo/packr/v2"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	migrate "github.com/rubenv/sql-migrate"
)

// RunMigrations will execute pending migrations if needed to keep
// the database updated with the latest changes
func RunMigrations(cfg Config) error {
	c, err := pgx.ParseConfig("postgres://" + cfg.User + ":" + cfg.Password + "@" + cfg.Host + ":" + cfg.Port + "/" + cfg.Name)
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(*c)

	var migrations = &migrate.PackrMigrationSource{Box: packr.New("hermez-db-migrations", "./migrations")}
	nMigrations, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return err
	}

	log.Info("successfully ran ", nMigrations, " migrations Up")
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

	// reset db droping migrations table and schemas
	if _, err := pgStorage.db.Exec(context.Background(), "DROP TABLE IF EXISTS gorp_migrations CASCADE;"); err != nil {
		return err
	}
	if _, err := pgStorage.db.Exec(context.Background(), "DROP SCHEMA IF EXISTS sync CASCADE;"); err != nil {
		return err
	}
	if _, err := pgStorage.db.Exec(context.Background(), "DROP SCHEMA IF EXISTS merkletree CASCADE;"); err != nil {
		return err
	}

	// run migrations
	if err := RunMigrations(cfg); err != nil {
		return err
	}
	return nil
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
