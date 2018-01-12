# Database

## SQL backend

Gohan generates SQL table definition based on a schema.

- string -> varchar(255)
- integer/number -> numeric
- boolean -> boolean

the other column will be "text".
We will encode data for JSON when we store complex data for RDBMS.


## YAML backend

YAML backend supports persistent data in YAML file format, and this backend is for only development purpose.

## Database Conversion tool

You can convert data source each other using convert tool.

```
  NAME:
     convert - Convert DB

  USAGE:
     command convert [command options] [arguments...]

  DESCRIPTION:
     Gohan convert can be used to migrate Gohan resources between different types of databases

  OPTIONS:
     --in-type, --it      Input db type (yaml, json, sqlite3)
     --in, -i             Input db connection spec (or filename)
     --out-type, --ot     Output db type (yaml, json, sqlite3)
     --out, -o            Output db connection spec (or filename)
     --schema, -s         Schema file
```

## Database Migration

Gohan supports generating database migration scripts. Current implementation is based
on a goose fork (https://github.org/cloudwan/goose) that allows easy integration
and has minimum number of external dependencies.
During startup gohan always checks if the database is in current version and if not, it rejects
to run to avoid data corruption. Using the scripts provided by a developer user can perform
migration across different versions. Since gohan migration support is mostly done with an external library,
the main subcommand documentation in located in the library:
https://github.com/cloudwan/goose/blob/master/README.md

```
  NAME:
     migrate - Manage migrations

  USAGE:
     gohan migrate [subcommand] [options]

  DESCRIPTION:
     Manage gohan migrations

  SUBCOMMAND:
     up         Migrate to the most recent version
     up-by-one  Migrate one version up
     create     Create a template for a new migration
     down       Migrate to the oldest version
     redo       Migrate one version back
     status     Display migration status
     version    Display migration version

  OPTIONS:
     --config-file "gohan.yaml"	Server config File
```

By default, all migration scripts are stored in 'db/migrations'. This path can be overridden
by a configuration setting 'db/migrations' field in yaml configuration.

## Fuzzying database connection (for testing purposes)

A developer can fuzz database connection so that each request to database will either
fail with a deadlock or return the original value as there was no fuzzing at all.
This is useful for simulating high load on database in which case it can occasionally
return deadlocks, which is a normal situation.

To enable fuzzing, set environment variable FUZZY_DB_TX to 'true' before gohan is run:
``` bash
FUZZY_DB_TX=true gohan [options]
```

## Metrics

The following metrics are exported:

- [prefix].db.connect (timer) - duration of establishing a database connection
- [prefix].db.close (timer) - duration of closing a database connection
- [prefix].db.active (counter) - number of active transactions
- [prefix].db.begin (timer) - duration of BEGIN TRANSACTION
- [prefix].db.begin_tx (timer) - duration of BEGIN TRANSACTION with custom options
- [prefix].db.begin.waiting (counter) - number of goroutines waiting for an available transaction
- [prefix].db.begin.failed (counter) - number of times a transaction could not be started
- [prefix].db.commit (timer) - duration of COMMIT
- [prefix].db.commit.failed (counter) - number of times the commit failed
- [prefix].db.rollback (timer) - duration of ROLLBACK
- [prefix].db.rollback.failed (counter) - number of times the ROLLBACK failed

For each schema:

- [prefix].tx.SCHEMA_ID.create (timer)
- [prefix].tx.SCHEMA_ID.update (timer)
- [prefix].tx.SCHEMA_ID.delete (timer)
- [prefix].tx.SCHEMA_ID.list (timer)
- [prefix].tx.SCHEMA_ID.lock_list (timer)
- [prefix].tx.SCHEMA_ID.fetch (timer)
- [prefix].tx.SCHEMA_ID.lock_fetch (timer)
- [prefix].tx.SCHEMA_ID.state_fetch (timer)
- [prefix].tx.SCHEMA_ID.state_update (timer)
- [prefix].tx.SCHEMA_ID.query (timer)
- [prefix].tx.SCHEMA_ID.count (timer)


- [prefix].tx.unknown_schema.exec (timer) - duration of Exec operations