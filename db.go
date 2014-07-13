package gomigrate

import (
	"database/sql"
	"log"
)

type Migratable interface {
	CreateMigrationsTable(*sql.DB) error
}

type Postgres struct{}

const pgCreateMigrationTableSql = `
CREATE TABLE gomigrate (
  id           SERIAL       PRIMARY KEY,
  migration_id INT          UNIQUE NOT NULL,
  name         VARCHAR(100) UNIQUE NOT NULL,
  status       INT          NOT NULL
)`

// Creates the migrations table if it doesn't exist.
func (m Postgres) CreateMigrationsTable(db *sql.DB) error {
	log.Print("Creating migrations table")

	_, err := db.Query(pgCreateMigrationTableSql)
	if err != nil {
		log.Fatalf("Error creating migrations table: %v", err)
	}

	log.Printf("Created migrations table: %s", migrationTableName)

	return nil
}
