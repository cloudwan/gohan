# Utitliy Commands

Gohan provides various utitliy commands.

Run ``gohan -h`` to list available commands

```
NAME:
   gohan - Gohan

USAGE:
   gohan [global options] command [command options] [arguments...]

COMMANDS:
   client			Manage Gohan resources
   validate, v			Validate document
   init-db, idb			Initialize DB backend with given schema file
   convert, conv		Convert DB
   server, srv			Run API Server
   test_extensions, test_ex	Run extension tests
   migrate, mig			Generate goose migration script
   template, template		Convert gohan schema using pongo2 template
   run, run			Run Gohan script Code
   test, test			Run Gohan script Test
   openapi, openapi		Convert gohan schema to OpenAPI
   markdown, markdown		Convert gohan schema to markdown doc
   dot, dot			Convert gohan schema to dot file for graphviz
   glace-server, gsrv		Run API Server with graceful restart support
   help, h			Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -d		Show debug messages
   --help, -h		show help
   --version, -v	print the version
```

## Validate

```
  NAME:
     validate - Validate Json Schema file

  USAGE:
     command validate [command options] [arguments...]

  DESCRIPTION:


  OPTIONS:
     --schema, -s '../schema/core.json'   Json schema path
     --json, -i '../example/example.json' json path
```

## Template

```
    NAME:
        template - Convert gohan schema using pongo2 template

    USAGE:
        command template [command options] [arguments...]

    DESCRIPTION:
        Convert gohan schema using pongo2 template

    OPTIONS:
        --config-file "gohan.yaml"	Server config File
        --template, -t 		Template File
```

## OpenAPI

```
    NAME:
        openapi - Convert gohan schema to OpenAPI

    USAGE:
        command openapi [command options] [arguments...]

    DESCRIPTION:
        Convert gohan schema to OpenAPI

    OPTIONS:
        --config-file "gohan.yaml"				Server config File
        --template, -t "embed://etc/templates/openapi.tmpl"	Template File
```

## MarkDown

```
    NAME:
        markdown - Convert gohan schema using pongo2 template

    USAGE:
        command markdown [command options] [arguments...]

    DESCRIPTION:
        Convert gohan schema using pongo2 template

    OPTIONS:
        --config-file "gohan.yaml"				Server config File
        --template, -t "embed://etc/templates/markdown.tmpl"	Template File
```

## CLI Client

You can use gohan client command to connect to gohan.

Gohan client will show quick help for sub command for current
server schema.

### Configuration

Gohan CLI Client is configured though environment variables. Configuration can be divided into few sections.
You can use default configuraion for client with: source ./etc/gohan_client.rc

### Keystone

To configure keystone access for Gohan Client CLI, only few options should be set (they are compatible with those
from rackspace/gophercloud:

* `OS_AUTH_URL` - keystone url
* `OS_USERNAME` - keystone username
* `OS_PASSWORD` - keystone password
* `OS_TENANT_NAME` or `OS_TENANT_ID` - keystone tenant name or id
* `OS_DOMAIN_NAME` or `OS_DOMAIN_ID` - keystone domain name or id (for keystone v3 only)
* `OS_TOKEN_ID` - token ID used to authenticate requests

### Endpoint

Gohan enpoint url can be specified directly or Gohan Client can get it from configured keystone.

To specify endpoint directly, set `GOHAN_ENDPOINT_URL` to proper url.

When using keystone:

* `GOHAN_SERVICE_NAME` - Gohan service name in keystone
* `GOHAN_REGION` - Gohan region name in keystone

**Note:** setting all above options will result in using `GOHAN_ENDPOINT_URL`!

### Schemas

Gohan CLI Client is fetching available schemas from Gohan endpoint and can cache them in temp file for performance:

* `GOHAN_SCHEMA_URL` - Gohan schema url
* `GOHAN_CACHE_SCHEMAS` - should cache schemas (default - true)
* `GOHAN_CACHE_TIMEOUT` - how long cache is valid, uses 1h20m10s format (default - 5m)
* `GOHAN_CACHE_PATH` - where to store cache schemas (default - `/tmp/.cached_gohan_schemas`)

### Usage

Usage can be briefly described with:

```
gohan client schema_id command [arguments...] [resource_identifier]
```

Each part of above command is described below.

### Schema_id

`schema_id` determines which resource type we want to manage.

If used `schema_id` is not defined in Gohan, Gohan CLI Client will return `Command not found` error.

### Command

Commands are identical for each resources:

* `list` - List all resources
* `show` - Show resource details
* `create` - Create resource
* `set` - Update existing resource
* `delete` - Delete resource

### Arguments

Arguments should be specified in `--name value` format.
Passing JSON null value `--name "<null>"`
Passing JSON not null value `--name '{"name": [{"name": "value"}, {"name": "value"}]}'`

### Common arguments

In addition to resource related commands, some formatting commands are available:

* `--output-format [json/table]` - specifies in which format results should be shown. Can also be specified with :code:`GOHAN_OUTPUT_FORMAT` environment variable.

  - `json`, e.g:

```
      {
        "name": "Resource name",
        "description": "Resource very long and meaningful description",
        "some_list": [
          "list_element_1",
          "list_element_2"
        ]
      }
    ..
```

  - :code:`table`, e.g:

```
      +-------------+--------------------------------------+
      |    FIELD    |               TYPE                   |
      +-------------+--------------------------------------+
      | name        | Resource name                        |
      | description | Resource description                 |
      | some_list   | ["list_element_1", "list_element_2"] |
      +-------------+--------------------------------------+
```

* :code:`--verbosity [0-3]` - specifies how much information Gohan Client should show - handy for dubugging. Can also be specified with :code:`GOHAN_VERBOSITY` environment variable.

  - :code:`0` - no additional debug information is shown
  - :code:`1` - Sent request url and method is shown
  - :code:`2` - same as level :code:`1` + request & response body
  - :code:`3` - same as level :code:`2` + used auth token

### Resource identifier

Some commands (:code:`show, set, delete`) are executed on one resource only. To identify whis reource,
it's name or id could be used:

```
  gohan client network show network-id
  gohan client network show "Network Name"
  gohan client subnet create --network "Network Name"
  gohan client subnet create --network network-id
  gohan client subnet create --network_id network-id
```

### Custom commands

Any custom actions specified in schema are also supported as commands by gohan client and should be invoked as follows:

```
gohan client schema_id command [common arguments] [command input] [resource identifier]
```

Where `common arguments` and `resource identifier` are described aboved and `command input` is passed as JSON value.

# Sync with backend

Gohan stores an event log recording create, update and delete database operations.
This is done in the database transaction, so we can assume that
this event log data is consistent with the resource data.

The event log data contains the following information. (see schema in gohan.json)

- id -- an unique ID of the event
- type -- the type of the event
- path -- the path of the resource related to the event
- timestamp -- the time at which the event occurred
- version -- the version of the resource after this event occurred
- body -- the contents of the resource after this event occurred

The Gohan server will select one master node using the etcd backend CAS API.
Only the master node will then poll the event log table, pushing to the backend.

We may support mysql binlog api for better performance in future.

## State updates

Gohan will keep track of the state version of any resource associated to
a schema with metadata containing the key ``state_versioning`` set to
``true``. In such a case Gohan will remember the config version and
the state version of the resource. During creation the config version of
such a resource will be set to ``1``. On delete and update the version is
bumped by one. The state version is ``0`` originally and later read from
the sync backend and updated asynchronously. Both the versions are returned
in GET requests, together with additional information about the state.

For example say a simplistic resource with the following schema is created:

```
    - description: Just a named object
      id: named_object
      parent: ""
      metadata:
        state_versioning: true
      singular: named_object
      plural: named_object
      prefix: /v1.0
      schema:
        properties:
          name:
            default: ""
            permission:
            - create
            - update
            title: Name
            type: string
            unique: false
          id:
            permission:
            - create
            title: ID
            type: string
            format: uuid
        properties_order:
        - id
        - name
        required:
        - name
        type: object
      title: Named Object
```

and the ``name`` is set to ``Alice``. Then Gohan, through the standard event sync,
writes the following JSON object to the backend under the key
``config/v1.0/named_object/someGeneratedUuid``:

```
    {
      "body": {
        "id": "someGeneratedUuid",
        "name": "Alice"
      },
      "version": 1
    }
```

A worker program might now read this information, create a corresponding
southbound resource and write the following to the backend under the key
``state/v1.0/named_object/someGeneratedUuid``:

```
    {
      "version": 1,
      "error": "",
      "state": "Alice exists"
    }
```

Gohan will read this information and update the database accordingly.

Any state updates made when the state version already equals the config version
will be ignored.

## Monitoring updates

After a resource has been created in the southbound, one might monitor its
status. This is done using a very similar approach to status updates.
Monitoring updates the ``monitoring`` field in the database, which is returned
together with the rest of the state.

A continuation of the above example follows. After the resource has been created
in the southbound a worker program might monitor its status and then write
the result of this monitoring under the key
``monitoring/v1.0/named_object/someGeneratedUuid`` as the following JSON:

```
    {
      "version": 1,
      "monitoring": "Alice is well"
    }
```

Gohan will read this information and update the database accordingly.

Any monitoring updates made when the state version does not yet equal
the config version or the version in the JSON data doesn't match with
the config version will be ignored.