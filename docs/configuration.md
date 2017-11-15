## Configuration

In this section, we will describe how to configure Gohan server.

Gohan server command takes "--config-file" parameter to specify
 configuration file.

```
  gohan server --config-file etc/gohan.yaml
```

Note that there are some example configurations in the etc directory.
Gohan server configuration uses YAML format.
(For YAML specification, see http://yaml.org/)

```yaml
  #######################################################
  #  Gohan API Server example configuration
  ######################################################

  include: gohan.d
  # database connection configuration
  database:
      # sqlite3 and mysql supported
      type: "sqlite3"
      # connection string
      # it is file path for yaml, json and sqlite3 backend
      connection: "./etc/test.db"
      # should database be altered if schema was updated, default true
      # auto_migrate: true
      # please set no_init true in the production env, so that gohan don't initialize table
      # no_init: true
      initial_data:
          - type: "yaml"
            connection: "./etc/examples/heat_template.yaml"
      # if legacy is true table names are created from schema id,
      # otherwise table names are based on schema plural,
      # default true
      legacy: false
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
  # Note: only static and schema directory will be served
  document_root: "./etc"
  # list of etcd backend servers
  etcd:
      - "http://127.0.0.1:2379"
  # timeout in milliseconds.1000 milliseconds by default 
  etcd_timeout_ms: 1000
  # keystone configuration
  keystone:
      use_keystone: false
      fake: true
      auth_url: "http://localhost:35357/v2.0"
      user_name: "admin"
      tenant_name: "admin"
      password: "gohan"
  # CORS (Cross-origin resource sharing (CORS)) configuration for javascript based client
  cors: "*"

  # Profiling configuration
  profiling:
      # if true, gohan add routes /debug/pprof/ for pprof profiling
      enabled: true

  # allowed levels  "CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG",
  logging:
      stderr:
          enabled: true
          level: DEBUG
      file:
          enabled: true
          level: INFO
          filename: ./gohan.log
```

## Include

You can include config YAML files from specified dirs
Note that overwrapped configuration will be overridden by configuration loaded later, so we don't recommend to have duplicated config file key in multiple files.

```
  include: gohan.d
```

## Environment Value


You can use Environment value in the configuration.

```yaml
  address: {{ .GOHAN_IP}}:{{ .GOHAN_PORT}}
```

Note you need to use {{ and }}, and you need to put before
your environment key.
We are using Golang text template package, so please take a
look more at https://golang.org/pkg/text/template/

## Database

This section is for backend database configuration.
You can select from MySQL, sqlite3 and YAML.

Sample database configuration for MySQL.

```yaml
  # database connection configuration
  database:
      type: "mysql"
      connection: "root:gohan@127.0.0.1/gohan"
```

Sample database configuration for sqlite3.

```yaml
  # database connection configuration
  database:
      type: "sqlite3"
      connection: "./sqlite3.db"
```

Sample database configuration for YAML backend.

```yaml
  database:
      type: "yaml"
      connection: "./etc/example.yaml"
      initial_data:
          - type: "yaml"
            connection: "./etc/examples/initial_datayaml"
```

Cascade deletion, i.e. creating FOREIGN KEYs with CASCADE ON DELETE, can be activated with `cascade` switch.

```yaml
  database:
      type: "mysql"
      connection: "root:gohan@127.0.0.1/gohan"
      cascade: true
```

Disable database initialisation, set `no_init: true` in production env, so that gohan don't initialize table.

```yaml
  database:
      type: "mysql"
      connection: "root:gohan@127.0.0.1/gohan"
      no_init: true
```

Disable auto migrations, set `auto_migrate: false` so that gohan don't alter database tables, it can, however, add new tables. 

```yaml
  database:
      type: "mysql"
      connection: "root:gohan@127.0.0.1/gohan"
      auto_migrate: false
```

> You should add **?parseTime=true** for connection setting if you use DB migrate functionality.
> If you do not set this parameter, *gohan migrate status* does not work as expected.
> See: https://bitbucket.org/liamstask/goose/issues/62/scan-error-on-column-index-0-unsupported

#### Automatic retry of deadlocked transactions

Gohan supports automatic retry of database transactions that have failed due to
a database deadlock. Under some circumstances (heavy load, multiple clients)
it is possible that a deadlock occurs in the database and by default,
Gohan returns an error via API. To override this behaviour configure
automatic retry of transactions: set retry count to a value greater than 0
and optionally configure retry intverval in milliseconds.

```yaml
database:
    deadlock_retry_tx:
        count: 5
        interval_msec: 100
```
Retryable transactions are disabled by default (count is set to zero).
See https://dev.mysql.com/doc/refman/5.7/en/innodb-deadlocks-handling.html for
more reading on this topic.

## Schema

Gohan works based on schema definitions.
Developers should specify a list of schema file in the configuration.

```yaml
schemas:
    - "embed://etc/schema/gohan.json"
    - "embed://etc/extensions/gohan_extension.yaml"
    - "./example_schema.yaml"
```

Developers can specify schemas here.
Note that we always need gohan.json and gohan_extension.yaml for WebUI and CLI.

## Keystone

Gohan supports OpenStack Keystone authentication backend.
(see http://docs.openstack.org/developer/keystone/ )

- use_keystone: boolean

  use keystone or not

- fake: boolean

  use fake Keystone server for testing or not

- auth_url

  keystone admin URL

- user_name

  service username

- tenant_name

  service tenant_name (needed for Keystone v2.0 API)

- domain

  service domain name (needed for Keystone v3.0 API)

- password

  password for a service user

- version

  v2.0 or v3 is supported

- use_auth_cache

  enable memory cache which stores keystone authorization responses for configurable time duration.    
  Please note that any token may be revoked before TTL expiration.   
  Revoke operation on cache is not supported. Such tokens will be authorized until TTL expires.  

- cache_ttl

  TTL of each cache entry.   
  Please note that this TTL must not exceed Keystone token expiration time.

```yaml
  keystone:
      use_keystone: false
      fake: true
      auth_url: "http://localhost:35357/v2.0"
      user_name: "admin"
      tenant_name: "admin"
      password: "gohan"
      use_auth_cache: false
      cache_ttl: 15m

```

## CORS

Gohan supports Cross-Origin Resource Sharing (CORS) for supporting
javascript WebUI without a proxy server.
You need to specify allowed domain pattern in CORS parameter.
Note: DO NOT USE * configuration in production deployment.

```yaml
  cors: "*"
```

## Profiling

Gohan runs with pprof profiling feature. You can get profiling results by querying
``http://<gohan_host>:<gohan_port>/debug/pprof/*`` . To get the profiling result,
you need admin credentials.

You can use profiling feature by set ``profiling/enabled`` as ``true``.

```yaml
  profiling:
      enabled: true
```

For the detail of pprof, please see https://golang.org/pkg/net/http/pprof/ .

NOTE: authentication is not provided under ``/debug/pprof/`` path, so when
you enable pprof on production environment, you should block the access to
this URL with a different way.


## Logging

You can define logging output in logging configuration.

Developers can specify Logging level per log and per module in log. If a module is not specified in "modules",  a value from "level" is applied.

Allowed log levels: "CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG",

```yaml
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
```

## HTTPS

- enabled

  You can enable HTTPS support by setting this flag to ``true``.
  Disabling this option will cause Gohan to fallback to HTTP.

- cert_file

  Location of X509 certificate file.
  e.g. ``"./etc/cert.pem"``

- key_file

  Location of the key file is matching with a certificate.
  e.g. ``"./etc/key.pem"``

```yaml
  tls:
    enabled: true
    cert_file: "./etc/cert.pem"
    key_file: "./etc/key.pem"
```

## Supported URL schemas

URL schemes including file://, http://, https:// and embed:// are supported. file:// is default.
For embed scheme, you can read embedded data in Gohan binary.
Files stored under "etc/schema" and "etc/extensions".

## Extension

- enable extension

You can select extension types you use.

```yaml
  extension:
    default: javascript
    use:
    - javascript
    - gohanscript
    - go
```

- extension timelimit

  You can make timelimit for extension execution. Default is 30 sec

```yaml
  extension:
    timelimit: 30
```

- extension npm_path

  You can set npm_path for extensions. It should point to a directory of node_modules. The default is the current working directory.

```yaml
  extension:
    npm_path: .
```

## Runtime metrics

You can configure reporting various runtime metrics (event handling time, extension execution time, sync/state watch processing time).
Currently only Graphite backend is supported.

- enable collecting and reporting runtime metrics
 
 Minimal configuration requires Graphite endpoints and enable flag, example:
```yaml
metrics:
  enabled: true
  graphite:
    endpoints:
      - "192.168.0.2:2003"  
      - "192.168.0.3:2003"  
```

- flush interval
 
 Interval of reports (in seconds), default: 60
```yaml
metrics:
  graphite:
    endpoints:
      - "192.168.0.2:2003"
    flush_interval_sec: 30  
```
 
- prefix
 
 Prepended to all metrics names, default: gohan
```yaml
metrics:
  graphite:
    endpoints:
      - "192.168.0.2:2003"
    prefix: "region.hostname.gohan" 
```   
    
- percentiles

 Percentiles reported in histograms, default: 0.5, 0.75, 0.95, 0.99, 0.999

```yaml
metrics:
  graphite:
    endpoints:
      - "192.168.0.2:2003"
    percentiles: 
      - "0.5"
      - "0.9"
```   

- temporarily disable
 
 If you want to disable collecting and reporting metrics, set enabled to false.
```yaml
metrics:
  enabled: false
  graphite:
    endpoints:
      - "192.168.0.2:2003"
```

## Miscellaneous

- address

  address to bind gohan sever.
  eg. 127.0.0.1:9090, 0.0.0.0:9090 or just :9090

- document_root

  Clients such as WebUI needs gohan-meta-schema file. We will serve the file from configured document_root.

- sync

  Sync type. The default is `etcd`, which means the etcd API version 2.
  `etcdv3` is available for etcd API version 3.

- etcd

  list of etcd backend.

```yaml
  etcd:
      - "http://192.0.0.1:2379"
      - "http://192.0.0.2:2379"
```

- run job on an update from etcd

  You can run extension on update event on etcd using
  sync://{{etcd_path}}.

a sample configuration.

- watch/keys  list of watched keys in etcd. Gohan watches the pathes which start with keys for etcd v3.
  For etcd v2, this will be done recursively.
- events list of an event we invokes an extension
- worker_count: number of concurrent execution tasks

```yaml
  watch:
      keys:
        - v2.0
      events:
        - v2.0/servers/
      worker_count: 4
```

WARNING: The value of watched etcd keys must be a JSON dictionary.

- amqp

  You can listen to notification event from OpenStack components using
  AMQP. You need to specify listen to queues and events.

  You can also run extension for amqp based event specifying path for
  amqp://{{event_type}}.

```yaml
  amqp:
      connection: amqp://guest:guest@172.16.25.130:5672/
      queues:
        - notifications.info
        - notifications.error
      events:
        - orchestration.stack
```

- snmp

 You can listen to snmp trap, and execute extension for that trap.
 An extension path should be snmp://

```yaml
  snmp:
    address: "localhost:8888"
```

- cron

  You can periodically execute CRON job using configuration.

```yaml
  cron:
      - path: cron://cron_job_sample
        timing: "*/5 * * * * *"
```

- Job queue

  Gohan uses background job queue & workers.
  You can decide how many worker can run concurrently.

```yaml
   workers: 100
```

- schema editor

  You can use a Gohan server as a schema editor if you specify editable_schema YAML file.
  Gohan updates this file based on schema REST API call.


```yaml
   editable_schema: ./example_schema.yaml
```

## Graceful Shutdown and Restart

Gohan supports graceful shutdown and restart.

For a graceful shutdown, send SIGTERM for Gohan process.
For graceful restart, you need to use server-starter utility.

```
  gohan glace-server --config-file etc/gohan.yaml
```

or you can use start_server utility

```
  # Installation
  go get github.com/lestrrat/go-server-starter/cmd/start_server
  # Run gohan server with start_server utility
  start_server --port 9091 -- gohan server --config-file etc/gohan.yaml
```
