package gomigrate

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

var (
	db *sql.DB
)

func TestNewMigrator(t *testing.T) {
	m, err := NewMigrator(db, "test_migrations/test1")
	if err != nil {
		t.Error(err)
	}
	if len(m.migrations) != 1 {
		t.Errorf("Invalid number of migrations detected")
	}

	migration := m.migrations[1]

	if migration.Name != "test" {
		t.Errorf("Invalid migration name detected: %s", migration.Name)
	}
	if migration.Id != 1 {
		t.Errorf("Invalid migration num detected: %s", migration.Id)
	}
	if migration.Status != Inactive {
		t.Errorf("Invalid migration num detected: %s", migration.Status)
	}
	if migration.UpPath != "test_migrations/test1/1_test_up.sql" {
		t.Errorf("Invalid migration up path detected: %s", migration.UpPath)
	}
	if migration.DownPath != "test_migrations/test1/1_test_down.sql" {
		t.Errorf("Invalid migration down path detected: %s", migration.DownPath)
	}

	cleanup()
}

func TestCreatingMigratorWhenTableExists(t *testing.T) {
	// Create the table and populate it with a row.
	_, err := db.Exec(createMigrationTableSql)
	if err != nil {
		t.Error(err)
	}
	_, err = db.Exec(migrationLogInsertSql, 123, "my_test", Active)
	if err != nil {
		t.Error(err)
	}
	// Create a migrator.
	_, err = NewMigrator(db, "test_migrations/test1")
	if err != nil {
		t.Error(err)
	}
	// Check that our row is still present.
	row := db.QueryRow("select name, status from gomigrate")
	var name string
	var status int
	err = row.Scan(&name, &status)
	if err != nil {
		t.Error(err)
	}
	if name != "my_test" {
		t.Error("Invalid name found in database")
	}
	if status != Active {
		t.Error("Invalid status found in database")
	}
	cleanup()
}

func TestMigrationAndRollback(t *testing.T) {
	m, err := NewMigrator(db, "test_migrations/test1")
	if err != nil {
		t.Error(err)
	}

	if err := m.Migrate(); err != nil {
		t.Error(err)
	}

	// Ensure that the migration ran.
	row := db.QueryRow(
		"SELECT tablename FROM pg_catalog.pg_tables WHERE tablename = $1",
		"test",
	)
	var tableName string
	if err := row.Scan(&tableName); err != nil {
		t.Error(err)
	}
	if tableName != "test" {
		t.Errorf("Migration table not created")
	}
	// Ensure that the migrate status is correct.
	row = db.QueryRow(
		"SELECT status FROM gomigrate where migration_id = $1",
		1,
	)
	var status int
	if err := row.Scan(&status); err != nil {
		t.Error(err)
	}
	if status != Active || m.migrations[1].Status != Active {
		t.Error("Invalid status for migration")
	}

	if err := m.Rollback(); err != nil {
		t.Error(err)
	}

	// Ensure that the down migration ran.
	row = db.QueryRow(
		"SELECT tablename FROM pg_catalog.pg_tables WHERE tablename = $1",
		"test",
	)
	err = row.Scan(&tableName)
	if err != sql.ErrNoRows {
		t.Errorf("Migration table should be deleted")
	}

	// Ensure that the migrate status is correct.
	row = db.QueryRow(
		"SELECT status FROM gomigrate where migration_id = $1",
		1,
	)
	if err := row.Scan(&status); err != nil {
		t.Error(err)
	}
	if status != Inactive || m.migrations[1].Status != Inactive {
		t.Error("Invalid status for migration")
	}

	cleanup()
}

func cleanup() {
	_, err := db.Exec("drop table gomigrate")
	if err != nil {
		panic(err)
	}
}

func init() {
	var err error
	db, err = sql.Open("postgres", "host=localhost dbname=gomigrate sslmode=disable")
	if err != nil {
		panic(err)
	}
}
