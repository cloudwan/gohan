==============
Configuration
==============

In this section we will describe how to configure
gohan server cli.

gohan server command takes "--config-file" parameter to specify
 configuraion file.


.. code-block:: shell

  gohan server --config-file etc/gohan.yaml


Note that there are some example configuraions in the etc directory.
Gohan server configuration is written in YAML format.
(For YAML specification, see http://yaml.org/)

.. code-block:: yaml

  #######################################################
  #  Gohan API Server example configuraion
  ######################################################

  include: gohan.d
  # database connection configuraion
  database:
      # sqlite3 and mysql supported
      type: "sqlite3"
      # connection string
      # it is file path for yaml, json and sqlite3 backend
      connection: "./etc/test.db"
      # please set no_init true in the production env, so that gohan don't initialize table
      # no_init: true
      initial_data:
          - type: "yaml"
            connection: "./etc/examples/heat_template.yaml"
  # schema path
  schemas:
    - "./etc/schema/gohan.json"
  # listen address for gohan
  address: ":9090"
  tls:
    enabled: true
    cert_file: "./etc/cert.pem"
    key_file: "./etc/key.pem"
  # document root of gohan API server
  # Note: only static and schema directoriy will be served
  document_root: "./etc"
  # list of etcd backend servers
  etcd:
      - "http://127.0.0.1:4001"
  # keystone configuraion
  keystone:
      use_keystone: false
      fake: true
      auth_url: "http://localhost:35357/v2.0"
      user_name: "admin"
      tenant_name: "admin"
      password: "gohan"
  # CORS (Cross-origin resource sharing (CORS)) configuraion for javascript based client
  cors: "*"

  # allowed levels  "CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG",
  logging:
      stderr:
          enabled: true
          level: DEBUG
      file:
          enabled: true
          level: INFO
          filename: ./gohan.log


Include
--------------------

You can include config yaml files from specified dirs
Note that overwrapped configuraion will be overrides by configuration loaded
later, so we don't recommend to have duplicated config file key in multiple files.

.. code-block:: yaml

  include: gohan.d

Environment Value
--------------------

You can use Environment value in the configuraion.

.. code-block:: yaml

  address: {{ .GOHAN_IP}}:{{ .GOHAN_PORT}}


Note you need to use {{ and }}, and you need to put . before
your env key.
We are using golang text template package, so please take a
look more at https://golang.org/pkg/text/template/

Database
--------------------

database is backend database configuraion.
You can select from sqlite3 and mysql.
Note that yaml and json is only for development purpose.

This is a sample database configuraion for sqlite3.

.. code-block:: yaml

  # database connection configuraion
  database:
      type: "sqlite3"
      connection: "./sqlite3.db"

Note that you need to initialize database using gohan init-db CLI.

gohan init-db takes following parameters.

-  --database_type, --dt 'sqlite3'	Backend datebase type
-  --database, -d 'gohan.db'		DB Connection String
-  --schema, -s '../schema/gohan.json'	Schema Def
-  --delete-on-create If selected, existing database will be dropped
-  --cascade If selected, FOREIGN KEYs will be created with CASCADE ON DELETE


This is example init database command for sqlite3 database

.. code-block:: yaml

    gohan init-db -s schema/gohan.json -d sqlite3.db


This is a sample database configuraion for mysql.

.. code-block:: yaml

  # database connection configuraion
  database:
      type: "mysql"
      connection: "root:gohan@127.0.0.1/gohan"

This is example init database command for sqlite3 database

.. code-block:: yaml

    gohan init-db -s schema/gohan.json -dt mysql -d "root:gohan/gohan"


You can also specify initial_data for static configs.
gohan server registers content of data on startup time.

.. code-block:: yaml

  database:
      type: "yaml"
      connection: "./etc/example.yaml"
      initial_data:
          - type: "yaml"
            connection: "./etc/examples/initial_datayaml"

Gohan policy determining what to do when there is existing database, can be specified
with drop_on_create option. If it's set to true, database will be dropped before
initialization.

.. code-block:: yaml

  database:
      type: "yaml"
      connection: "./etc/example.yaml"
      drop_on_create: true

Cascade deletion, i.e. creating FOREING KEYs with CASCADE ON DELETE,  can be
activated with cascade switch.

.. code-block:: yaml

  database:
      type: "yaml"
      connection: "./etc/example.yaml"
      cascade: true

Schema
-----------

Gohan works based on schema definitions.
You should specify list of schema file in the configuraion.

.. code-block:: yaml

  schemas:
    - "./etc/schema/gohan.json"


You can specify your own schema here. You can also use gohan meta-schema.
Meta-schema defines gohan schema itself.
You can see it on etc/schema/gohan.json.
When you have meta-schema in the schema configuraion, you can use gohan
meta-schema API and schema editor in webui.

Keystone
--------------

Gohan support OpenStack keystone authentication backend.
(see http://docs.openstack.org/developer/keystone/ )

- use_keystone: boolean

  use keystone or not

- fake: boolean

  use fake keystone server for testing or not

- auth_url

  keytone admin URL

- user_name

  service user name

- tenant_name

  service tenant_name (needed for keystone v2.0 api)

- domain

  service domain name (needed for keystone v3.0 api)

- password

  password for service user

- version

  v2.0 or v3 is suppoted

.. code-block:: yaml

  keystone:
      use_keystone: false
      fake: true
      auth_url: "http://localhost:35357/v2.0"
      user_name: "admin"
      tenant_name: "admin"
      password: "gohan"

CORS
--------------

Gohan supports Cross-Origin Resource Sharing (CORS) for supporting
javascript webui without proxy server.
You need to specify allowd domain pattern in cors parameter.
Note: DO NOT USE * configuraion in production deployment.

.. code-block:: yaml

  cors: "*"

Logging
--------------

You can define logging output in logging configuraion.

Logging level can be specified per log and per module in log. If module is not specified
in "modules", value from "level" is applied.

Allowed log levels: "CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG",


.. code-block:: yaml

  logging:
      stderr:
          enabled: true
          level: DEBUG
      file:
          enabled: true
          level: INFO
          modules:
              - name: gohan.db.sql
                level: DEBUG
              - name: gohan.sync.etcd
                level: CRITICAL
          filename: ./gohan.log

HTTPS
--------------

- enabled

  You can enable HTTPS support by setting this flag to ``true``.
  Disabling this option will cause Gohan to fallback to HTTP.

- cert_file

  Location of X509 certificate file.
  e.g. ``"./etc/cert.pem"``

- key_file

  Location of key file matching with certificate.
  e.g. ``"./etc/key.pem"``

.. code-block:: yaml

  tls:
    enabled: true
    cert_file: "./etc/cert.pem"
    key_file: "./etc/key.pem"

Path
--------------

By default, any path get considerted as file.
You can also specify schemes including file://, http://, https:// and embed://.
For embed scheme, you can read embedded data in gohan binary.
Currenlty, contents under etc/schema and etc/extentions are embedded.

Extension
-----------
- enable extension

You can select extension types you use.

.. code-block:: yaml

  extension:
    default: javascript
    use:
    - javascript
    - gohanscript
    - go

- extension timelimit

  You can make timelimit for extension execution. default is 30 sec

.. code-block:: yaml

  extension:
    timelimit: 30

- extension npm_path

  You can set npm_path for extensions. It should point to directory
  where node_modules are put. Default is current working directory.

.. code-block:: yaml

  extension:
    npm_path: .

Misc
--------------

- address

  address to bind gohan-sever.
  eg. 127.0.0.1:9090, 0.0.0.0:9090 or just :9090

- document_root

  Clients such as webui needs gohan-meta-schema file. We will serve the file from
  configured document_root.

- etcd

  list of etcd backend.


.. code-block:: yaml

  etcd:
      - "http://192.0.0.1:4001"
      - "http://192.0.0.2:4001"

- run job on update from etcd

  You can run extension on update event on etcd using
  sync://{{etcd_path}}.

This is a sample configuraion.

- watch/keys  list of watched keys in etcd. This will be done recursively.
- events list of event we invoke extension
- worker_count: number of concurrent execution tasks

.. code-block:: yaml

  watch:
      keys:
        - v2.0
      events:
        - v2.0/servers/
      worker_count: 4

WARNING: The value of watched etcd keys must be a JSON dictionary.

- amqp

  You can listen notification event from openstack components using
  amqp. You need to specify listen queues and events.

  You can also run extension for amqp based event specifying path for
  amqp://{{event_type}}.

.. code-block:: yaml

  amqp:
      connection: amqp://guest:guest@172.16.25.130:5672/
      queues:
        - notifications.info
        - notifications.error
      events:
        - orchestration.stack

- snmp

 You can listen snmp trap, and execute extesion for that trap.
 extension path should be snmp://

.. code-block:: yaml

  snmp:
    address: "localhost:8888"

- cron

  You can pediorically execute cron job using configuraion.
  extension path should be specified in the path.

.. code-block:: yaml

  cron:
      - path: cron://cron_job_sample
        timing: "*/5 * * * * *"

- Job queue

  Gohan uses background job queue & workers.
  You can decide how many worker can run concurrently.

.. code-block:: yaml

   workers: 100

- schema editor

  You can use gohan server as a schema editor if you specify editable_schema yaml file.
  Gohan updates this file based on scheam REST API call.


.. code-block:: yaml

   editable_schema: ./example_schema.yaml

Graceful Shutdown and Restart
-----------------------------

Gohan support graceful shutdown and restart.

For graceful shutdown, send SIGTERM for gohan process.
For graceful restart, you need to use server-starter utility.

.. code-block:: shell

  gohan glace-server --config-file etc/gohan.yaml

or you can use start_server utility

.. code-block:: shell

  # Installation
  go get github.com/lestrrat/go-server-starter/cmd/start_server

  # Run gohan server with start_server utility
  start_server --port 9091 -- gohan server --config-file etc/gohan.yaml