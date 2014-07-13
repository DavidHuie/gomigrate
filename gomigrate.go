// A simple database migrator for PostgreSQL.

package gomigrate

import (
	"database/sql"
	"errors"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"sort"
)

const (
	migrationTableName = "gomigrate"
)

var (
	upMigrationFile       = regexp.MustCompile(`(\d+)_(\w+)_up\.(\w+)`)
	downMigrationFile     = regexp.MustCompile(`(\d+)_(\w+)_down\.(\w+)`)
	InvalidMigrationFile  = errors.New("Invalid migration file")
	InvalidMigrationPair  = errors.New("Invalid pair of migration files")
	InvalidMigrationsPath = errors.New("Invalid migrations path")
)

type Migrator struct {
	DB             *sql.DB
	MigrationsPath string
	dbAdapter      Migratable
	migrations     map[uint64]*Migration
}

// Returns a new migrator.
func NewMigrator(db *sql.DB, adapter Migratable, migrationsPath string) (*Migrator, error) {
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
		adapter,
		make(map[uint64]*Migration),
	}

	// Create the migrations table if it doesn't exist.
	tableExists, err := migrator.migrationTableExists()
	if err != nil {
		return nil, err
	}
	if !tableExists {
		if err := migrator.dbAdapter.CreateMigrationsTable(migrator.DB); err != nil {
			return nil, err
		}
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
			migration = &Migration{Id: num, Name: name, Status: Inactive}
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

	// Find ids for the given status.
	migrations := make([]*Migration, 0)
	for _, id := range ids {
		migration := m.migrations[id]
		if migration.Status == status {
			migrations = append(migrations, migration)
		}
	}
	return migrations
}

const migrationLogInsertSql = `
INSERT INTO gomigrate (migration_id, name, status) values ($1, $2, $3)
`

// Applies all inactive migrations.
func (m *Migrator) Migrate() error {
	for _, migration := range m.Migrations(Inactive) {
		log.Printf("Applying migration: %s", migration.Name)

		sql, err := ioutil.ReadFile(migration.UpPath)
		if err != nil {
			log.Printf("Error reading up migration: %s", migration.Name)
			return err
		}
		transaction, err := m.DB.Begin()
		if err != nil {
			log.Printf("Error opening transaction: %v", err)
			return err
		}
		// Perform the migration.
		if _, err = transaction.Exec(string(sql)); err != nil {
			log.Printf("Error executing migration: %v", err)
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				log.Printf("Error rolling back transaction: %v", rollbackErr)
				return rollbackErr
			}
			return err
		}
		// Log the exception in the migrations table.
		_, err = transaction.Exec(
			migrationLogInsertSql,
			migration.Id,
			migration.Name,
			Active,
		)
		if err != nil {
			log.Printf("Error logging migration: %v", err)
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				log.Printf("Error rolling back transaction: %v", rollbackErr)
				return rollbackErr
			}
			return err
		}
		if err := transaction.Commit(); err != nil {
			log.Printf("Error commiting transaction: %v", err)
			return err
		}

		// Do this as the last step to ensure that the database has
		// been updated.
		migration.Status = Active
	}
	return nil
}

var (
	NoActiveMigrations = errors.New("No active migrations to rollback")
)

const (
	migrationLogUpdateSql = "UPDATE gomigrate SET status = $1 WHERE migration_id = $2"
)

// Rolls back the last migration
func (m *Migrator) Rollback() error {
	migrations := m.Migrations(Active)

	if len(migrations) == 0 {
		return NoActiveMigrations
	}

	lastMigration := migrations[len(migrations)-1]

	log.Printf("Rolling back migration: %v", lastMigration.Name)

	sql, err := ioutil.ReadFile(lastMigration.DownPath)
	if err != nil {
		log.Printf("Error reading migration: %s", lastMigration.DownPath)
		return err
	}
	transaction, err := m.DB.Begin()
	if err != nil {
		log.Printf("Error creating transaction: %v", err)
		return err
	}
	_, err = transaction.Exec(string(sql))
	if err != nil {
		transaction.Rollback()
		log.Printf("Error rolling back transaction: %v", err)
		return err
	}

	// Change the status in the migrations table.
	_, err = transaction.Exec(
		migrationLogUpdateSql,
		Inactive,
		lastMigration.Id,
	)
	if err != nil {
		log.Printf("Error logging rollback: %v", err)
		if rollbackErr := transaction.Rollback(); rollbackErr != nil {
			log.Printf("Error rolling back transaction: %v", rollbackErr)
			return rollbackErr
		}
		return err
	}

	err = transaction.Commit()
	if err != nil {
		log.Printf("Error commiting transaction: %v", err)
		return err
	}
	lastMigration.Status = Inactive
	return nil
}
