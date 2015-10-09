# Gohan #

Project Site: http://gohan.cloudwan.io/

Gohan is an REST API framework which has

- Schema: Gohan makes REST-based api server, db backend, CLI ,and WebUI from JSON schema.
- Exension: Gohan supports custom logic using Go, JavaScript or Gohan DSL.
- Policy: Gohan supports RBAC-based policy enforcement for your API.
- Integration: Gohan can integrated with 3rd party system using Sync (etcd) and OpenStack Keystone

[![gohan demo](https://github.com/cloudwan/gohan_website/raw/master/gohan_demo.gif)](https://www.youtube.com/watch?v=cwI44wQcxHU)

[![Join the chat at https://gitter.im/cloudwan/gohan](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/cloudwan/gohan?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

[![GoDoc](https://godoc.org/github.com/cloudwan/gohan?status.svg)](https://godoc.org/github.com/cloudwan/gohan)

[![wercker status](https://app.wercker.com/status/cab137b4bfdd05c97cfface7ac12c039/m "wercker status")](https://app.wercker.com/project/bykey/cab137b4bfdd05c97cfface7ac12c039)

# Getting started #

You can do step1 and step2 on Heroku using this button.

## Setup ##

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/cloudwan/gohan.git)

 or

* Download Gohan binary + sample configuraion https://github.com/cloudwan/ansible-gohan/releases

* Start server

```
./gohan server --config-file etc/gohan.yaml
```

## WebUI client ##

```
https://localhost:9443/webui/ (or https://$APPNAME.herokuapp.com/webui/ )
```

login with this ID/Password

```
ID: admin
Password: gohan
```

You can also access schema editing webui by adding "?type=metaschema" on URL.

```
https://localhost:9443/webui/?type=metaschema
```

## Try CLI client ##

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

For Heroku

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

# Example #

## Define your resource model ##

You can define your resource model using JSON schema.
You can use YAML format as described in this example.

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

## Define your application policy ##

Gohan can use OpenStack Keystone as Identity management system.
You can configure API access policy based on role information on Keystone.

Policy has following properties.

- id : Identitfy of the policy
- principal : Keystone Role
- action: one of `create`, `read`, `update`, `delete` for CRUD operations
  on resource or any custom actions defined by schema performed on a
  resource or `*` for all actions
- effect : Allow api access or not
- resource : target resource
  you can specify target resource using "path" and "properties"
- condition : addtional condition (see below)
- tenant_id : regexp matching the tenant, defaults to ``.*``


```yaml
  policies:
  - action: '*'
    effect: allow
    id: admin_statement
    principal: admin
    resource:
      path: .*
```

## Implement custom logic ##

you can add your custom logic for each CRUD event.

Javascript

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

[Experimental] Donburi (Ansible inspired Gohan DSL)

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

You can also find an example for Go based extension in here
https://github.com/cloudwan/gohan/tree/master/exampleapp

see more description in doc

## Integrate Gohan with your system ##

Every CRUD event will be pushed to sync layer (currently etcd is supproted), so your worker can be syncroniced.

You can also use Gohan as a worker.
Gohan support AMQP (OpenStack notification), SNMP (experimental) and CRON, and execute extension.

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

# More examples #

see more on https://github.com/cloudwan/gohan_apps

# Development #

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

- Send a pull request for github

# Why Gohan? #

- Gohan is a REST-based api server to evolve your cloud service very rapidly and enable painless operation

- Gohan makes your system architeture simple

- Gohan enables you to create / modify new service very rapidly with minimum coding

- Gohan can be easiliy integrated to your system using sync layer and custom extension

## SINGLE PROCESS for REST based "micro" services ##

Traditional “fat” services are difficult to manage: subsystems become tightly coupled making it hard to introduce changes. Micro-service based architectures are meant to solve this issue; however they come with additional problems such as process management and orchestration. (see Criticism on https://en.wikipedia.org/wiki/Microservices)

Gohan enables you to define many "micro" services within single unified process, so that you can keep your system architecture and deployment model simple. Moreover, Gohan supports transactional management for microservices orchestration.

## SCHEMA-DRIVEN Service Develoment ##

Similar structure code everywhere.
Typical Schema-driven development tools reduces number of code lines by automatic code generation. Down side of the code generation method is adding complexity to the code management.

In Gohan project, we are challenging to have powerful schema-driven development without those complexity.

## SINGLE PLACE to keep Service related poilcy, config and code ##

Cloud service management is not simple.
Typical cloud architecutre has three layers, i.e. Controller, Communication and Agent layer. When you develop a new service on this archtecutre, configurations and codes will be distributed across three layers. This makes hard to diagnose issue and manage services.

In Gohan, each layer can understand service definition file, so that you can unify service related configuraion and code on one place.

# License #

Apache2

# Docs #

see more docs http://gohan.cloudwan.io/gohan/
