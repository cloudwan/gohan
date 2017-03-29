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