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
	_, err := NewMigrator(db, "test_migrations")
	if err != nil {
		t.Error(err)
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
