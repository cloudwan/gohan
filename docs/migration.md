## Migration

In this section, we will describe how to to configure and use Gohan
migrations.

Gohan migrations are based on goose library fork 'cloudwan/goose'
(see http://github.com/cloudwan/goose). The library is used internally
so user does not need to have an external working goose binary.
All migration tasks are exposed with a single 'gohan migrate' command.

Gohan migrate command has different subcommands which are similar
to goose subcommands. Each command takes "--config-file" parameter
to specify a configuration file. For example:

```
  gohan migrate status --config-file etc/gohan.yaml
```

will display current migration status. Remaining commands include:

##### up: Migrate to the most recent version

This subcommand will walk though all pending migrations and apply
each pending migration until there is an error or all migrations
were applied.

##### up-by-one: Migrate one version up

Migrate only one pending migration. This is useful to separately test
if each pending migration does not introduce any new problems to
the system.

##### create: Create a template for a new migration
Complete command line is:
```bash
gohan migrate create <name> [sql/go]
```
This subcommand will create an empty template for migrations either in GO or in SQL. By default
a new migration is created in GO if no migration type parameter is given.

##### init: Create an initial migration

This subcommand is used to create an initial migration from an empty
database to the current version of all schemas.

##### down: Migrate to the oldest version

This subcommand reverts all applied migrations.

##### redo: Migrate one version back

This subcommand reverts the last applied migration.

##### status: Display migration status

This command displays a complete summary of available migrations and their
status. A migration can be in one of the two states:
- applied: a migration has been applied and the corresponding status contains
the time/date when the migration was applied
- pending: a migration has not yet been applied

##### version: Display migration version

This subcommand will display the current version of database.

### Configuration

Migrations in SQL and in GO are read from a common 'migrations' directory
which is defined as key 'database/migrations' in Gohan configuration file.
By default it is set to "db/migrations":
```yaml
database:
    migrations: "db/migrations"
```

### SQL migrations

SQL migrations are the same as in goose. Please refer to the original
documentation for the details.

### Precompiled GO migrations

Gohan uses precompiled shared libraries to store migrations in go.
In order to create a migrations library, one can use the following command
in migrations directory:

```
  go build -buildmode=plugin -o gohan_go_migrations.so `find . -maxdepth 1 -type f -name "*.go"`
```

Note that each migration can be precompiled as separate libraries
or alternatively, all migrations can be stored in a single library.
Both solutions are handled by Gohan.
