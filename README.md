# gomigrate

A database migration tool for Postgres in Golang.

## Usage

First import the package:

```go
import "github.com/DavidHuie/gomigrate"
```

Given a `database/sql` database connection to a PostgreSQL database, `db`,
and a directory to migration files, create a migrator:

```go
migrator := gomigrate.NewMigrator(db, "./migrations")
```

To migrate the database, run:

```go
err := migrator.Migrate()
```

To rollback the last migration, run:

```go
err := migrator.Rollback()
```

## Migration files

Migration files need to follow a standard format and must be present
in the same directory. Given "up" and "down" steps for a migration,
create two files that follow this template:

```
{{ id }}_{{ name }}_{{ "up" or "down" }}.sql
```

For a given migration, the `id` and `name` fields must be the same.
The id field is an integer that corresponds to the order in which
the migration should run relative to the other migrations.

### Example

If I'm trying to add a "users" table to the database, I would create
the following two files:

#### 1_add_users_table_up.sql

```
CREATE TABLE users();
```

#### 1_add_users_table_down.sql
```
DROP TABLE users;
```

## Copyright

Copyright (c) 2014 David Huie. See LICENSE.txt for further details.
