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

# Getting started

You can satisfy the first two steps of **Setup** on Heroku using this button:

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/cloudwan/gohan.git)

## Setup
* Download Gohan binary and sample configuration: https://github.com/cloudwan/ansible-gohan/releases
* Start server: `./gohan server --config-file etc/gohan.yaml`

## WebUI client
```
https://localhost:9443/webui/ (or https://$APPNAME.herokuapp.com/webui/ )
```

Login with this ID/password

```
ID: admin
Password: gohan
```

You can also access schema editing webUI by adding `?type=metaschema` on URL.

```
https://localhost:9443/webui/?type=metaschema
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
export OS_AUTH_URL=https://localhost:9443/v2.0

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

# Examples
## Define your resource model

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

## Define your application policy

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

## Implement custom logic

You can add your custom logic for each CRUD event.

### Javascript

```yaml
extensions:
- code: |
    gohan_register_handler("pre_create", function (context){
       console.log("Hello world")
    });
  event: list
  id: test
  path: /v2.0/network.*
```

### [Experimental] Donburi (Ansible inspired Gohan DSL)

```yaml
extensions:
- id: network
  code_type: donburi
  code: |
    tasks:
      - contrail:
          schema: "virtual-network"
          allow_update: []
          id: "{{ .resource.contrail_virtual_network }}"
          properties:
            parent_type: "project"
            fq_name:
              - default-domain
              - "{{ .tenant_name }}"
              - "{{ .resource.id }}"
        register: network_response
      - vars:
          status: "ACTIVE"
        when: network_response.status_code == 200
        else:
          - vars:
              status: "ERROR"
          - vars:
              response_code: 409
            when: network_response.status_code != 404 && event_type == "pre_delete"
      - update:
          schema: "network"
          properties:
            id: "{{ .resource.id }}"
            contrail_virtual_network: '{{ index .network_response "data" "virtual-network" "uuid" }}'
            status: "{{ .status }}"
  path: "/v2.0/network.*"
```

You can also find an example for Go based extensions in here
https://github.com/cloudwan/gohan/tree/master/exampleapp

See more information in the documentation.

## Integrate Gohan with your system

Every CRUD event will be pushed to a sync layer (currently *etcd* is supported), so your worker can be synchronized.

You can also use Gohan as a worker. Gohan supports AMQP (OpenStack
  notification), SNMP (experimental), and CRON, to execute extensions.

```yaml
# Watch etcd and execute extension
- id: sync_notification
  code_type: donburi
  path: "sync://v2.0/servers/"
  code: |
    tasks:
      - debug: "synced {{ .action }} "
# Watch RabbitMQ
- id: amqp_notification
  code_type: donburi
  path: "amqp://orchestration.stack"
  code: |
    tasks:
      - vars:
         stack_id: "{{ .event.payload.stack_name }}"
         state: "{{ .event.payload.state }}"
      - eval: "stack_id = stack_id.slice(7)"
      - vars:
          state: "ACTIVE"
        when: state == "CREATE_COMPLETE"
      - update:
          schema: "server"
          properties:
            id: "{{ .stack_id }}"
            status: "{{ .state }}"
        rescue:
          - debug: "{{ .error }}"
# Watch SNMP
- id: snmp
  code_type: donburi
  path: "snmp://"
  code: |
    tasks:
      - debug: "remote host: {{ .remote }} {{ .trap }} "
      - debug: "traps: {{ .item.key }} {{ .item.value }} "
        with_dict: "trap"
# CRON Job
- id: cron_job
  code_type: donburi
  path: "cron://cron_job_sample"
  code: |
    tasks:
      - debug: "cron job"
```

# More examples

See more at https://github.com/cloudwan/gohan_apps

# Development

- Setup Go env
- Install development tools

```
go get github.com/tools/godep
go get github.com/golang/lint/golint
go get github.com/coreos/etcd
go get golang.org/x/tools/cmd/cover
go get golang.org/x/tools/cmd/vet
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

# Additional resources
See more documentation at http://gohan.cloudwan.io/gohan/
