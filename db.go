package gomigrate

type Migratable interface {
	SelectMigrationTableSql() string
	CreateMigrationTableSql() string
	GetMigrationSql() string
	MigrationLogInsertSql() string
	MigrationLogDeleteSql() string
}

// POSTGRES

type Postgres struct{}

func (p Postgres) SelectMigrationTableSql() string {
	return "SELECT tablename FROM pg_catalog.pg_tables WHERE tablename = $1"
}

func (p Postgres) CreateMigrationTableSql() string {
	return `CREATE TABLE gomigrate (
                  id           SERIAL       PRIMARY KEY,
                  migration_id BIGINT       UNIQUE NOT NULL
                )`
}

func (p Postgres) GetMigrationSql() string {
	return `SELECT migration_id FROM gomigrate WHERE migration_id = $1`
}

func (p Postgres) MigrationLogInsertSql() string {
	return "INSERT INTO gomigrate (migration_id) values ($1)"
}

func (p Postgres) MigrationLogDeleteSql() string {
	return "DELETE FROM gomigrate WHERE migration_id = $1"
}

// MYSQL

type Mysql struct{}

func (m Mysql) SelectMigrationTableSql() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_name = ?"
}

func (m Mysql) CreateMigrationTableSql() string {
	return `CREATE TABLE gomigrate (
                  id           INT          NOT NULL AUTO_INCREMENT,
                  migration_id BIGINT       NOT NULL UNIQUE,
                  PRIMARY KEY (id)
                ) ENGINE=MyISAM`
}

func (m Mysql) GetMigrationSql() string {
	return `SELECT migration_id FROM gomigrate WHERE migration_id = ?`
}

func (m Mysql) MigrationLogInsertSql() string {
	return "INSERT INTO gomigrate (migration_id) values (?)"
}

func (m Mysql) MigrationLogDeleteSql() string {
	return "DELETE FROM gomigrate WHERE migration_id = ?"
}

// MARIADB

type Mariadb Mysql
