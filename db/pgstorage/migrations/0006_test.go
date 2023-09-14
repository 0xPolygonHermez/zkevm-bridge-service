package migrations_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This migration creates a different proof table dropping all the information.

type migrationTest0006 struct{}

func (m migrationTest0006) InsertData(db *sql.DB) error {
	return nil
}

func (m migrationTest0006) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// Check Primary Keys
	tokenPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'sync' and table_name = 'token_wrapped' and constraint_type = 'PRIMARY KEY';"
	row := db.QueryRow(tokenPK)
	var count int
	assert.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count)

	rhtPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'mt' and table_name = 'rht' and constraint_type = 'PRIMARY KEY';"
	row2 := db.QueryRow(rhtPK)
	var count2 int
	assert.NoError(t, row2.Scan(&count2))
	assert.Equal(t, 1, count2)

	rootPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'mt' and table_name = 'root' and constraint_type = 'PRIMARY KEY';"
	row3 := db.QueryRow(rootPK)
	var count3 int
	assert.NoError(t, row3.Scan(&count3))
	assert.Equal(t, 1, count3)

	rhtTemp := "select count(*) from mt.rht_temp;"
	row4 := db.QueryRow(rhtTemp)
	var count4 int
	assert.Error(t, row4.Scan(&count4))
}

func (m migrationTest0006) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	// Check Primary Keys
	tokenPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'sync' and table_name = 'token_wrapped' and constraint_type = 'PRIMARY KEY';"
	row := db.QueryRow(tokenPK)
	var count int
	assert.NoError(t, row.Scan(&count))
	assert.Equal(t, 0, count)

	rhtPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'mt' and table_name = 'rht' and constraint_type = 'PRIMARY KEY';"
	row2 := db.QueryRow(rhtPK)
	var count2 int
	assert.NoError(t, row2.Scan(&count2))
	assert.Equal(t, 0, count2)

	rootPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'mt' and table_name = 'root' and constraint_type = 'PRIMARY KEY';"
	row3 := db.QueryRow(rootPK)
	var count3 int
	assert.NoError(t, row3.Scan(&count3))
	assert.Equal(t, 0, count3)

	rhtTemp := "select count(*) from mt.rht_temp;"
	row4 := db.QueryRow(rhtTemp)
	var count4 int
	assert.NoError(t, row4.Scan(&count4))
	assert.Equal(t, 0, count4)
}

func TestMigration0006(t *testing.T) {
	runMigrationTest(t, 6, migrationTest0006{})
}
