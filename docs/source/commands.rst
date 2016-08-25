===================
Gohan Commands
===================

Run ``gohan -h`` to list available commands

.. code-block:: shell

  NAME:
     gohan - Gohan

  USAGE:
     gohan [global options] command [command options] [arguments...]

  VERSION:
     0.0.0

  COMMANDS:
     validate, v          Validate Json Schema file
     init-db, id          Init DB
     convert, conv        Convert DB
     server, srv          Run API Server
     help, h              Shows a list of commands or help for one command

  GLOBAL OPTIONS:
     --version, -v        print the version
     --help, -h           show help


-----------------
Validate
-----------------

.. code-block:: shell

  NAME:
     validate - Validate Json Schema file

  USAGE:
     command validate [command options] [arguments...]

  DESCRIPTION:


  OPTIONS:
     --schema, -s '../schema/core.json'   Json schema path
     --json, -i '../example/example.json' json path


-----------------
Template
-----------------

.. code-block:: shell

    NAME:
        template - Convert gohan schema using pongo2 template

    USAGE:
        command template [command options] [arguments...]

    DESCRIPTION:
        Convert gohan schema using pongo2 template

    OPTIONS:
        --config-file "gohan.yaml"	Server config File
        --template, -t 		Template File

-----------------
OpenAPI
-----------------

.. code-block:: shell

    NAME:
        openapi - Convert gohan schema to OpenAPI

    USAGE:
        command openapi [command options] [arguments...]

    DESCRIPTION:
        Convert gohan schema to OpenAPI

    OPTIONS:
        --config-file "gohan.yaml"				Server config File
        --template, -t "embed://etc/templates/openapi.tmpl"	Template File

-----------------
MarkDown
-----------------

.. code-block:: shell

    NAME:
        markdown - Convert gohan schema using pongo2 template

    USAGE:
        command markdown [command options] [arguments...]

    DESCRIPTION:
        Convert gohan schema using pongo2 template

    OPTIONS:
        --config-file "gohan.yaml"				Server config File
        --template, -t "embed://etc/templates/markdown.tmpl"	Template File

-----------------
CLI Client
-----------------

You can use gohan client command to connect to gohan.

Gohan client will show quick help for sub command for current
server schema.

Configuration
=============

Gohan CLI Client is configured though environment variables. Configuration can be divided into few sections.

You can use default configuraion for client with:
:code:`source ./etc/gohan_client.rc`

Keystone
--------

To configure keystone access for Gohan Client CLI, only few options should be set (they are compatible with those
from rackspace/gophercloud:

* :code:`OS_AUTH_URL` - keystone url
* :code:`OS_USERNAME` - keystone username
* :code:`OS_PASSWORD` - keystone password
* :code:`OS_TENANT_NAME` or :code:`OS_TENANT_ID` - keystone tenant name or id
* :code:`OS_DOMAIN_NAME` or :code:`OS_DOMAIN_ID` - keystone domain name or id (for keystone v3 only)
* :code:`OS_TOKEN_ID` - token ID used to authenticate requests

Endpoint
--------

Gohan enpoint url can be specified directly or Gohan Client can get it from configured keystone.

To specify endpoint directly, just set :code:`GOHAN_ENDPOINT_URL` to proper url.

When using keystone:

* :code:`GOHAN_SERVICE_NAME` - Gohan service name in keystone
* :code:`GOHAN_REGION` - Gohan region name in keystone

**Note:** setting all above options will result in using :code:`GOHAN_ENDPOINT_URL`!

Schemas
-------

Gohan CLI Client is fetching available schemas from Gohan endpoint and can cache them in temp file for performance:

* :code:`GOHAN_SCHEMA_URL` - Gohan schema url
* :code:`GOHAN_CACHE_SCHEMAS` - should cache schemas (default - true)
* :code:`GOHAN_CACHE_TIMEOUT` - how long cache is valid, uses 1h20m10s format (default - 5m)
* :code:`GOHAN_CACHE_PATH` - where to store cache schemas (default - :code:`/tmp/.cached_gohan_schemas`)

Usage
=====

Usage can be briefly described with:

:code:`gohan client schema_id command [arguments...] [resource_identifier]`

Each part of above command is described below.

Schema_id
---------

:code:`schema_id` determines which resource type we want to manage.

If used :code:`schema_id` is not defined in Gohan, Gohan CLI Client will return :code:`Command not found` error.

Command
-------

Commands are identical for each resources:

* :code:`list` - List all resources
* :code:`show` - Show resource details
* :code:`create` - Create resource
* :code:`set` - Update existing resource
* :code:`delete` - Delete resource

Arguments
---------

Arguments should be specified in :code:`--name value` format.

Passing JSON null value: :code:`--name "<null>"`

Passing JSON not null value: :code:`--name '{"name": [{"name": "value"}, {"name": "value"}]}'`

**Common arguments**

In addition to resource related commands, some formatting commands are available:

* :code:`--output-format [json/table]` - specifies in which format results should be shown. Can also be specified with :code:`GOHAN_OUTPUT_FORMAT` environment variable.

  - :code:`json`, e.g:

    .. code-block:: javascript

      {
        "name": "Resource name",
        "description": "Resource very long and meaningful description",
        "some_list": [
          "list_element_1",
          "list_element_2"
        ]
      }
    ..

  - :code:`table`, e.g:

    .. code-block:: shell

      +-------------+--------------------------------------+
      |    FIELD    |               TYPE                   |
      +-------------+--------------------------------------+
      | name        | Resource name                        |
      | description | Resource description                 |
      | some_list   | ["list_element_1", "list_element_2"] |
      +-------------+--------------------------------------+

* :code:`--verbosity [0-3]` - specifies how much information Gohan Client should show - handy for dubugging. Can also be specified with :code:`GOHAN_VERBOSITY` environment variable.

  - :code:`0` - no additional debug information is shown
  - :code:`1` - Sent request url and method is shown
  - :code:`2` - same as level :code:`1` + request & response body
  - :code:`3` - same as level :code:`2` + used auth token

Resource identifier
-------------------

Some commands (:code:`show, set, delete`) are executed on one resource only. To identify whis reource,
it's name or id could be used:

.. code-block:: shell

  gohan client network show network-id
  gohan client network show "Network Name"
  gohan client subnet create --network "Network Name"
  gohan client subnet create --network network-id
  gohan client subnet create --network_id network-id

Custom commands
---------------

Any custom actions specified in schema are also supported as commands by gohan client and should be invoked as follows:

:code:`gohan client schema_id command [common arguments] [command input] [resource identifier]`

Where :code:`common arguments` and :code:`resource identifier` are described aboved and :code:`command input` is passed as JSON value.
