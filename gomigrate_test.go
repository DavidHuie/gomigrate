package gomigrate

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var (
	db      *sql.DB
	adapter Migratable
)

func GetMigrator(test string) *Migrator {
	var suffix string
	if os.Getenv("DB") == "pg" {
		suffix = "pg"
	} else {
		suffix = "mysql"
	}
	path := fmt.Sprintf("test_migrations/%s_%s", test, suffix)
	m, err := NewMigrator(db, adapter, path)
	if err != nil {
		panic(err)
	}
	return m
}

func TestNewMigrator(t *testing.T) {
	m := GetMigrator("test1")
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

	cleanup()
}

func TestCreatingMigratorWhenTableExists(t *testing.T) {
	// Create the table and populate it with a row.
	_, err := db.Exec(adapter.CreateMigrationTableSql())
	if err != nil {
		t.Error(err)
	}
	_, err = db.Exec(adapter.MigrationLogInsertSql(), 123)
	if err != nil {
		t.Error(err)
	}

	GetMigrator("test1")

	// Check that our row is still present.
	row := db.QueryRow("select migration_id from gomigrate")
	var id uint64
	err = row.Scan(&id)
	if err != nil {
		t.Error(err)
	}
	if id != 123 {
		t.Error("Invalid id found in database")
	}
	cleanup()
}

func TestMigrationAndRollback(t *testing.T) {
	m := GetMigrator("test1")

	if err := m.Migrate(); err != nil {
		t.Error(err)
	}

	// Ensure that the migration ran.
	row := db.QueryRow(
		adapter.SelectMigrationTableSql(),
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
		adapter.GetMigrationSql(),
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
		adapter.SelectMigrationTableSql(),
		"test",
	)
	err := row.Scan(&tableName)
	if err != sql.ErrNoRows {
		t.Errorf("Migration table should be deleted")
	}

	// Ensure that the migration log is missing.
	row = db.QueryRow(
		adapter.GetMigrationSql(),
		1,
	)
	if err := row.Scan(&status); err != sql.ErrNoRows {
		t.Error(err)
	}
	if m.migrations[1].Status != Inactive {
		t.Errorf("Invalid status for migration, %v", m.migrations[1].Status)
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
	if os.Getenv("DB") == "pg" {
		log.Print("Using postgres")
		adapter = Postgres{}
		db, err = sql.Open("postgres", "host=localhost dbname=gomigrate sslmode=disable")
		if err != nil {
			panic(err)
		}
	} else {
		log.Print("Using mysql")
		adapter = Mysql{}
		db, err = sql.Open("mysql", "gomigrate:password@/gomigrate")
	}
}
