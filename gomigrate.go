// A simple migrator for PostgreSQL.

package gomigrate

import (
	"database/sql"
	"errors"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

const (
	migrationTableName = "gomigrate"
)

var (
	// Regexps for migration file paths.
	upMigrationFile   = regexp.MustCompile(`(\d+)_(\w+)_down.sql`)
	downMigrationFile = regexp.MustCompile(`(\d+)_(\w+)_up.sql`)

	InvalidMigrationFile  = errors.New("Invalid migration file")
	InvalidMigrationPair  = errors.New("Invalid pair of migration files")
	InvalidMigrationsPath = errors.New("Invalid migrations path")
)

type Migrator struct {
	DB             *sql.DB
	MigrationsPath string
	migrations     map[uint64]*Migration
}

// Returns a new migrator.
func NewMigrator(db *sql.DB, migrationsPath string) (*Migrator, error) {
	// Normalize the migrations path.
	path := []byte(migrationsPath)
	pathLength := len(path)
	if path[pathLength-1] != '/' {
		path = append(path, '/')
	}

	log.Printf("Migrations path: %s", path)

	migrator := Migrator{
		db,
		string(path),
		make(map[uint64]*Migration),
	}

	if err := migrator.createMigrationsTable(); err != nil {
		return nil, err
	}

	// Get all metadata from the database.
	if err := migrator.fetchMigrations(); err != nil {
		return nil, err
	}
	if err := migrator.getMigrationStatuses(); err != nil {
		return nil, err
	}

	return &migrator, nil
}

const selectTablesSql = "SELECT tablename FROM pg_catalog.pg_tables"

// Returns true if the migration table already exists.
func (m *Migrator) migrationTableExists() (bool, error) {
	rows, err := m.DB.Query(selectTablesSql)
	if err != nil {
		log.Printf("Error checking for migration table: %v", err)
		return false, err
	}
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			return false, err
		}
		if tableName == migrationTableName {
			log.Printf("Found migrations table: %v", tableName)
			return true, nil
		}
	}

	log.Print("Migrations table not found")

	return false, nil
}

const createMigrationTableSql = `
CREATE TABLE gomigrate (
  id     SERIAL       PRIMARY KEY,
  name   VARCHAR(100) UNIQUE NOT NULL,
  status INT                 NOT NULL
);`

// Creates the migrations table if it doesn't exist.
func (m *Migrator) createMigrationsTable() error {
	status, err := m.migrationTableExists()
	if err != nil {
		return err
	}
	if status {
		return nil
	}

	log.Print("Creating migrations table")

	_, err = m.DB.Query(createMigrationTableSql)
	if err != nil {
		log.Fatalf("Error creating migrations table: %v", err)
	}

	log.Printf("Created migrations table: %s", migrationTableName)

	return nil
}

// Returns the migration number, type and base name, so 1, "up", "migration" from "01_migration_up.sql"
func parseMigrationPath(path string) (uint64, string, string, error) {
	filebase := filepath.Base(path)

	// Check to see if this is a up migration.
	matches := upMigrationFile.FindAllSubmatch([]byte(filebase), -1)
	if matches != nil {
		num := matches[0][1]
		name := matches[0][2]
		parsedNum, err := strconv.ParseUint(string(num), 10, 64)
		if err != nil {
			return 0, "", "", err
		}
		return parsedNum, "up", string(name), nil
	}

	// Down migration.
	matches = downMigrationFile.FindAllSubmatch([]byte(filebase), -1)
	if matches != nil {
		num := matches[0][1]
		name := matches[0][2]
		parsedNum, err := strconv.ParseUint(string(num), 10, 64)
		if err != nil {
			return 0, "", "", err
		}
		return parsedNum, "down", string(name), nil
	}

	return 0, "", "", InvalidMigrationFile
}

// Populates a migrator with a sorted list of migrations from the file system.
func (m *Migrator) fetchMigrations() error {
	pathGlob := append([]byte(m.MigrationsPath), []byte("*")...)

	log.Printf("Migrations path glob: %s", pathGlob)

	matches, err := filepath.Glob(string(pathGlob))
	if err != nil {
		log.Fatalf("Error while globbing migrations: %v", err)
	}

	for _, match := range matches {
		num, migrationType, name, err := parseMigrationPath(match)
		if err != nil {
			log.Printf("Invalid migration file found: %s", match)
			continue
		}

		log.Printf("Migration file found: %s", match)

		migration, ok := m.migrations[num]
		if !ok {
			migration = &Migration{Num: num, Name: name, Status: inactive}
			m.migrations[num] = migration
		}
		if migrationType == "up" {
			migration.UpPath = match
		} else {
			migration.DownPath = match
		}
	}

	// Validate each migration.
	for _, migration := range m.migrations {
		if !migration.valid() {
			path := migration.UpPath
			if path == "" {
				path = migration.DownPath
			}

			log.Printf("Invalid migration pair for path: %s", path)
			return InvalidMigrationPair
		}
	}

	log.Printf("Migrations file pairs found: %v", len(m.migrations))

	return nil
}

const migrationStatusSql = "SELECT status FROM gomigrate WHERE name = $1"

// Queries the migration table to determine the status of each
// migration.
func (m *Migrator) getMigrationStatuses() error {
	for _, migration := range m.migrations {
		rows, err := m.DB.Query(migrationStatusSql, migration.Name)
		if err != nil {
			log.Printf(
				"Error getting migration status for %s: %v",
				migration.Name,
				err,
			)
			return err
		}
		for rows.Next() {
			var status int
			err := rows.Scan(&status)
			if err != nil {
				log.Printf("Error getting migration status: %v", err)
				return err
			}
			migration.Status = status
			log.Printf(
				"Migration %s found with status: %v",
				migration.Name,
				status,
			)
		}
	}
	return nil
}

// Returns a sorted list of migration ids for a given status.
func (m *Migrator) Migrations(status int) []*Migration {
	// Sort all migration ids.
	ids := make([]uint64, 0)
	for id, _ := range m.migrations {
		ids = append(ids, id)
	}
	sort.Sort(uint64slice(ids))

	// Find inactive ids.
	migrations := make([]*Migration, 0)
	for _, id := range ids {
		migration := m.migrations[id]
		if migration.Status == status {
			migrations = append(migrations, migration)
		}
	}
	return migrations
}

// Applies all inactive migrations.
func (m *Migrator) Migrate() error {
	for _, migration := range m.Migrations(inactive) {
		sql, err := ioutil.ReadFile(migration.UpPath)
		if err != nil {
			return err
		}
		transaction, err := m.DB.Begin()
		if err != nil {
			return err
		}
		_, err = transaction.Exec(string(sql))
		if err != nil {
			transaction.Rollback()
			return err
		}
		err = transaction.Commit()
		if err != nil {
			return err
		}
		migration.Status = active
	}
	return nil
}

// Rolls back the last migration
func (m *Migrator) Rollback() error {
	migrations := m.Migrations(active)
	lastMigration := migrations[len(migrations)-1]
	sql, err := ioutil.ReadFile(lastMigration.DownPath)
	if err != nil {
		return err
	}
	transaction, err := m.DB.Begin()
	if err != nil {
		return err
	}
	_, err = transaction.Exec(string(sql))
	if err != nil {
		transaction.Rollback()
		return err
	}
	err = transaction.Commit()
	if err != nil {
		return err
	}
	lastMigration.Status = inactive
	return nil
}
