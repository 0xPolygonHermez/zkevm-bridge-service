package migrations_test

import (
	"database/sql"
	"testing"
)

type migrationTest0009 struct{}

func (m migrationTest0009) InsertData(db *sql.DB) error {
	return nil
}

func (m migrationTest0009) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {

}
func (m migrationTest0009) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
}
func TestMigration0009(t *testing.T) {
	runMigrationTest(t, 9, migrationTest0009{})
}
