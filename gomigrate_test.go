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
