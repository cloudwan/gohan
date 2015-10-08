==============
Database
==============

SQL backend
--------------

Gohan will geneate table based on schema.

- string -> varchar(255)
- integer/number -> numeric
- boolean -> boolean

the other column will be "text".
We will encode data for json when we store complex data for db.


YAML backend
--------------

This can be used to manipulate schema using gohan-ui.
output can be used schema for gohan-server.
However, YAML backend don't support constraints or concurrent
request, so this backend is only for development and testing purpose.


Database Conversion tool
------------------------

.. code-block:: shell

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


Database Migraion
-----------------

Gohan supports generating goose (https://bitbucket.org/liamstask/goose) migration script.
Currently, we don't support calcurating diff on schema, so that app developmer should manage
this migration script.

.. code-block:: shell

  NAME:
     migrate - Generate goose migration script

  USAGE:
     command migrate [command options] [arguments...]

  DESCRIPTION:
     Generates goose migraion script

  OPTIONS:
     --name, -n 'init_schema'		name of migrate
     --schema, -s 			Schema definition
     --path, -p 'etc/db/migrations'	Migrate path
     --cascade				If true, FOREIGN KEYS in database will be created with ON DELETE CASCADE

