# Gohan #

Project Site: http://gohan.cloudwan.io/

Gohan is an REST API framework which has

- Schemas: Gohan provides a REST-based API server, database backend, CLI, and WebUI generated from a JSON schema.
- Extensions: Gohan supports custom logic using Go, JavaScript, or Gohan DSL.
- Policies: Gohan supports RBAC-based policy enforcement for your API.
- Integrations: Gohan can integrate with 3rd-party system using Sync (etcd) and OpenStack Keystone.

[![gohan demo](https://github.com/cloudwan/gohan_website/raw/master/gohan_demo.gif)](https://www.youtube.com/watch?v=cwI44wQcxHU)

[![Join the chat at https://gitter.im/cloudwan/gohan](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/cloudwan/gohan?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) [![GoDoc](https://godoc.org/github.com/cloudwan/gohan?status.svg)](https://godoc.org/github.com/cloudwan/gohan)

[![wercker status](https://app.wercker.com/status/cab137b4bfdd05c97cfface7ac12c039/m "wercker status")](https://app.wercker.com/project/bykey/cab137b4bfdd05c97cfface7ac12c039)

[![Coverage Status](https://coveralls.io/repos/github/cloudwan/gohan/badge.svg?branch=master)](https://coveralls.io/github/cloudwan/gohan?branch=master)

# Getting started

You can satisfy the first two steps of **Setup** on Heroku using this button:

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/cloudwan/gohan.git)

## Setup
* Download Gohan binary and sample configuration: https://github.com/cloudwan/ansible-gohan/releases
* Start server: `./gohan server --config-file etc/gohan.yaml`

see more [document](./docs/source/installation.rst)

## WebUI client
```
http://localhost:9091/ (or https://$APPNAME.herokuapp.com/ )
```

Login with this ID/password

```
ID: admin
Password: gohan
```

You can also access schema editing webUI by adding `?type=metaschema` on URL.

```
http://localhost:9091/webui/?type=metaschema
```

## Try CLI client
### Local
```
export GOHAN_SCHEMA_URL=/gohan/v0.1/schemas
export GOHAN_REGION=RegionOne
export GOHAN_SERVICE_NAME=gohan
export OS_PASSWORD=gohan
export OS_USERNAME=admin
export GOHAN_CACHE_SCHEMAS=true
export GOHAN_CACHE_TIMEOUT=5m
export GOHAN_CACHE_PATH=~/.cached-gohan-schemas
export OS_AUTH_URL=http://localhost:9091/v2.0

./gohan client
```

### Heroku
```
export GOHAN_SCHEMA_URL=/gohan/v0.1/schemas
export GOHAN_REGION=RegionOne
export GOHAN_SERVICE_NAME=gohan
export OS_PASSWORD=gohan
export OS_USERNAME=admin
export GOHAN_CACHE_SCHEMAS=true
export GOHAN_CACHE_TIMEOUT=5m
export GOHAN_CACHE_PATH=~/.cached-gohan-schemas
export OS_AUTH_URL=https://$APPNAME.herokuapp.com/v2.0
export GOHAN_ENDPOINT_URL=https://$APPNAME.herokuapp.com

./gohan client
```

gohan cli provides various functions

see more [commands document](./docs/source/commands.rst)

# Configuration guide

see [config document](./docs/source/config.rst)

# Schema

You can define your resource model using a JSON schema.
Alternatively you can use YAML format as described in this example.

```yaml
  schemas:
  - id: network
    plural: networks
    prefix: /v2.0
    schema:
      properties:
        description:
          default: ""
          permission:
          - create
          - update
          title: Description
          type: string
          unique: false
        id:
          format: uuid
          permission:
          - create
          title: ID
          type: string
          unique: false
        name:
          permission:
          - create
          - update
          title: Name
          type: string
          unique: false
        tenant_id:
          format: uuid
          permission:
          - create
          title: Tenant
          type: string
          unique: false
      propertiesOrder:
      - id
      - name
      - description
      - tenant_id
      type: object
    singular: network
    title: Network
```

see more

- [schema document](./docs/source/schema.rst)

# API

- [api document](./docs/source/api.rst)

# DB

- [db document](./docs/source/database.rst)

# Appliation policy

Gohan can use OpenStack Keystone as an identity management system. You can
configure API access policy based on role information in Keystone.

A policy has the following properties:

- `id`: identity of the policy
- `principal`: Keystone Role
- `action`: one of `create`, `read`, `update`, `delete` for CRUD operations on
resource or any custom actions defined by schema performed on a
  resource or `*` for all actions
- `effect`: Allow API access or not
- `resource`: target resource you can specify target resource using "path" and "properties"
- `condition` : additional condition (see below)
- `tenant_id` : regexp matching the tenant, defaults to `.*`

```yaml
  policies:
  - action: '*'
    effect: allow
    id: admin_statement
    principal: admin
    resource:
      path: .*
```

see more [policy document](./docs/source/policy.rst)

# Extension

You can add your custom logic for each CRUD event using Javascript or
Gohanscript (YAML based DSL).

see more [extension document](./docs/source/extension.rst)

# Integrate Gohan with your system

Every CRUD event will be pushed to a sync layer (currently *etcd* is supported), so your worker can be synchronized.

You can also use Gohan as a worker. Gohan supports AMQP (OpenStack
  notification), SNMP (experimental), and CRON, to execute extensions.

see more [sync document](./docs/source/sync.rst)

# Examples

See more at https://github.com/cloudwan/gohan_apps

# Development

- Setup Go env
- Install development tools

```
make deps
```

- make & make install

```
make
make install
```

- Send a pull request to Github

# Why Gohan?

- Gohan is a REST-based API server to evolve your cloud service very rapidly and enable painless CRUD operations.

- Gohan makes your system architecture simple
- Gohan enables you to create / modify new services very rapidly with minimum coding
- Gohan can be easily integrate into your system using a sync layer and custom extensions

## Single process for REST based micro-services

Traditional “fat” services are difficult to manage: subsystems become tightly coupled making it hard to introduce changes. Micro-service based architectures are meant to solve this issue; however they come with additional problems such as process management and orchestration. (see Criticism on https://en.wikipedia.org/wiki/Microservices)

Gohan enables you to define many micro-services within a single, unified process, so that you can keep your system architecture and deployment model simple. Moreover, Gohan supports transactional management for micro-services orchestration.

## Scheme-driven service development

Similar structure and code everywhere. Typical schema-driven development tools
reduces the number of code lines by automatic code generation. The down side of
the code generation method is adding complexity to the code management.

In the Gohan project, we are challenged to have powerful schema-driven
development without those complexities.

## Single place to keep service related policy, config, and code

Cloud service management is not simple. A typical cloud architecture has three layers:
* Controller
* Communication
* Agent

When you develop a new service on this architecture, configuration and code will be distributed across three layers. This makes it hard to diagnose issues and manage services.

In Gohan, each layer can understand service definition file, so that you can unify service related configuration and code on one place.

# License
Apache2

# How to contribute

* Sign our CLA and send scan for info@cloudwan.io

(Individual version) https://github.com/cloudwan/gohan/blob/master/docs/cla.txt
(Company version) https://github.com/cloudwan/gohan/blob/master/docs/ccla.txt

* Create an issue in github
* Send PR for github

We recommend to rebase mulitple commit for 1.

# Additional resources
See more documentation at http://gohan.cloudwan.io/gohan/ or
 [document](./docs/source/)