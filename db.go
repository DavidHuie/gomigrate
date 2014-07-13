package gomigrate

type Migratable interface {
	SelectMigrationTableSql() string
	CreateMigrationTableSql() string
	MigrationStatusSql() string
	MigrationLogInsertSql() string
	MigrationLogUpdateSql() string
}

// POSTGRESQL

type Postgres struct{}

func (p Postgres) SelectMigrationTableSql() string {
	return "SELECT tablename FROM pg_catalog.pg_tables WHERE tablename = $1"
}

func (p Postgres) CreateMigrationTableSql() string {
	return `CREATE TABLE gomigrate (
                  id           SERIAL       PRIMARY KEY,
                  migration_id INT          UNIQUE NOT NULL,
                  name         VARCHAR(100) UNIQUE NOT NULL,
                  status       INT          NOT NULL
                )`
}

func (p Postgres) MigrationStatusSql() string {
	return "SELECT status FROM gomigrate WHERE name = $1"
}

func (p Postgres) MigrationLogInsertSql() string {
	return "INSERT INTO gomigrate (migration_id, name, status) values ($1, $2, $3)"
}

func (p Postgres) MigrationLogUpdateSql() string {
	return "UPDATE gomigrate SET status = $1 WHERE migration_id = $2"
}

// MYSQL

type Mysql struct{}

func (m Mysql) SelectMigrationTableSql() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_name = ?"
}

func (m Mysql) CreateMigrationTableSql() string {
	return `CREATE TABLE gomigrate (
                  id           INT          NOT NULL AUTO_INCREMENT,
                  migration_id INT          NOT NULL UNIQUE,
                  name         VARCHAR(100) NOT NULL UNIQUE,
                  status       INT          NOT NULL,
                  PRIMARY KEY (id)
                ) ENGINE=MyISAM`
}

func (m Mysql) MigrationStatusSql() string {
	return "SELECT status FROM gomigrate WHERE name = ?"
}

func (m Mysql) MigrationLogInsertSql() string {
	return "INSERT INTO gomigrate (migration_id, name, status) values (?, ?, ?)"
}

func (m Mysql) MigrationLogUpdateSql() string {
	return "UPDATE gomigrate SET status = ? WHERE migration_id = ?"
}
