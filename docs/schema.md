# Schema

Developers defines resource model itself, and Gohan derives APIs, CLIs, and Docs from it. It is a so-called model-based development and conceptual difference from OpenAPI where developers define API.

We will have a list of schemas to define a resource model.
Each schema will have following properties.

- id          -- resource id (must be unique)
- singular    -- a singular form of the schema name
- plural      -- plural form of the schema name
- title       -- use the visible label of resource title
- description -- a description of the schema
- schema      -- JSON schema (see Spec on http://json-schema.org/)

Schemas might also have any of the following optional properties.

- parent    -- the id of the parent schema
- on_parent_delete_cascade -- cascading delete when parent resource deleted
- namespace -- resource namespace for grouping
- prefix    -- resource path prefix
- metadata  -- application specific schema metadata (object)
- type      -- can be an abstract or empty string (see more in schema inheritance)
- extends   -- list of base schemas
- order_properties_before -- to order properties before properties of extended schemas. 

## Schema Inheritance

Gohan supports mix-in of multiple schemas.
Developers can make a schema as abstract schema specifying type=abstract. The developer can mix-in abstract schema.

```yaml
  schemas:
  - description: base
    type: abstract
    id: base
    metadata:
      state_versioning: true
    plural: bases
    prefix: /v2.0
    schema:
      properties:
        description:
          description: Description
          default: ""
          permission:
          - create
          - update
          title: Description
          type: string
          unique: false
        id:
          description: ID
          permission:
          - create
          title: ID
          type: string
          unique: false
        name:
          description: Name
          permission:
          - create
          - update
          title: Name
          type: string
          unique: false
        tenant_id:
          description: Tenant ID
          permission:
          - create
          title: Tenant
          type: string
          unique: false
      indexes:
        unique_name_per_tenant_id:
          columns:
          - tenant_id
          - name
          type: "unique"
      propertiesOrder:
      - id
      - name
      - description
      - tenant_id
      type: object
    singular: base
    title: base
  - description: Network
    id: network
    extends:
    - base
    plural: networks
    schema:
      properties:
        providor_networks:
          description: Providor networks
          default: {}
          permission:
          - create
          - update
          properties:
            segmentation_type:
              enum:
              - vlan
              - vxlan
              - gre
              type: string
            segmentation_id:
              minimum: 0
              type: integer
          title: Provider Networks
          type: object
          unique: false
        route_targets:
          description: Route targets
          default: []
          items:
            type: string
          permission:
          - create
          - update
          title: RouteTargets
          type: array
          unique: false
        shared:
          description: Shared
          permission:
          - create
          - update
          title: Shared
          type: boolean
          unique: false
          default: false
      propertiesOrder:
      - providor_networks
      - route_targets
      - shared
      type: object
    singular: network
    title: Network
```

## Metadata

- nosync (boolean)

  We don't sync this resource for sync backend when this option is true.

- state_versioning (boolean)

  whether to support state versioning <subsection-state-update>, defaults to false.

- sync_key_template (string)

  configurable sync key path for schemas based on properties, for example: /v1.0/devices/{{device_id}}/virtual_machine/{{id}},

- sync_plain (boolean)

  Write plain data, which is not JSON marshaled in the Gohan format, to the sync backend if true.
  By default, which is false, Gohan writes sync data in JSON with its own format. The format has `body` and `version` properties, then the value of the resource, which is also JSON object, is stored in the `body` property with escaping (e.g. `{"body": "{\"id\":1,\"property\":\"value\"}", "version": 1}`).
  However, this format is not always supported by your worker. Therefore you sometimes want Gohan to write sync data in a simpler way. This option allow you to get the body JSON data without encapsulation by Gohan (e.g. `{"id":1,"property":"value"}`).
  Note that when you use this option with `sync_property`, you can get the value of a specified property. For instance, when you provide `property` to `sync_property`, you will get `value` for the result. In this case, when the type of the value is `string`, Gohan doesn't marshal the value into a JSON string, which means you don't get `"value"` here. When the value is an array or an object, you will get a JSON marshalled string.

- sync_property (string)

  Write only the value of the specified property to the sync backend.

- sync_skip_config_prefix (bool)

  Some resources don't require two way sync or path needs to be customized exactly with sync_key_template.
  Enabling this flag will cause /config prefix to be skipped when generating sync path for etcd.

- resource_group (string)

  Used in OpenApi documentation it allows to categorized schema according to given `resource_group` by setting appropriate tags.
  If `--split-by-resource-group` option is used to generate OpenApi documentation it will additionally place given schema to `resource_group.js` file.

- locking_policy (list)

  Sets if the resource should be locked in the database (using `SELECT ... FOR UPDATE`) at the beginning of `pre_update_in_transaction` and `pre_delete_in_transaction` event handling. 
  
  Subkeys:
  - delete (string) - locking policy for `pre_delete_in_transaction`
  - update (string) - locking policy for `pre_update_in_transaction`
  
  Allowed values:
  - `lock_related` - locks the resource and related resources
  - `skip_related` - locks the resource but leaves related resources unlocked
  - (empty): default, no locking

## Properties

We need to define properties of a resource using following parameters.

- title

  User visible label of the property

- format

  Additional validation hints for this property
  you can use defined attribute on http://json-schema.org/latest/json-schema-validation.html#anchor107

  Currently following formats are supported:
	- "mac"
	- "cidr"
	- "cidr-or-ipv4"
	- "regex"
	- "uuid"
	- "hyph-uuid"
	- "non-hyph-uuid"
	- "port"
	- "yaml"
	- "text"
	- "version"

- type

 Gohan supports standard JSON schema types including string, number, integer, boolean, array, object and combinations such as ["string", "null"]
  The Schema itself should be the object type.

- default

  the default value of the property

- enum

  You can specify list of allowed values

- required

  List of required attributes to specified during creation


Following properties are extended from JSON schema v4.

- permission

  permission is a list of allowing actions for this property.
  valid values contains "create", "update".
  Gohan generates JSON schema for creating API and update API based on this value.
  Note that we can use this property for only first level properties.

- unique boolean (unique key constraint)

- indexed boolean

  Specify if index should be created in DB for given column 

## type string

type string is for defining a string.
You can use following parameters for a string.

- minLength  min length of string
- maxLength  max length of string
- pattern    regexp pattern for this string
- relation  (extended spec by Gohan)  define resource relation
- relationColumn (extended spec by Gohan) define which column relation references (default: "id")
- relation_property  (extended spec by Gohan) relation resource will be joined in list API request for this property name
- on_delete_cascade  (extended spec by Gohan) cascading delete when related resource deleted

eg.
```yaml
        name:
          permission:
          - create
          - update
          title: Name
          type: string
          unique: false
```

## type boolean

type boolean for boolean value

eg.

```yaml
        admin_state:
          permission:
          - create
          - update
          title: admin_state
          type: boolean
          unique: false
```

## type integer or type number

type integer or type number for numeric properties.
You can use following parameters to define valid range

- maximum (number) and exclusiveMaximum (boolean)
- minimum (number) and exclusiveMinimum (boolean)

eg.

```yaml
        age:
          permission:
          - create
          - update
          title: age
          type: number
          unique: false
```

## type array

type array is for a defining list of elements

### items

  Only allowed for array type
  You can define element type on this property.

eg.

```yaml
        route_targets:
          default: []
          items:
            type: string
          permission:
          - create
          - update
          title: RouteTargets
          type: array
          unique: false
```

## type object

Object type is for a defining object in the resources.
Note that resource itself should be an object.
Following parameters supported in the object type.

- properties

  Only allowed for object type
  You can define properties of this object

- propertiesOrder (extended parameter in gohan)

  Only allowed for object type
  You can define an ordering of properties using propertiesOrder for UI / CLI

eg.

```yaml
        providor_networks:
          default: {}
          permission:
          - create
          - update
          properties:
            segmentation_type:
              enum:
              - vlan
              - vxlan
              - gre
              type: string
            segmentation_id:
              minimum: 0
              type: integer
          required:
          - segmentation_type
          - segmentation_id
          title: Provider Networks
          type: object
          unique: false
```
- order_properties_before

  when resource is extended using 'extends' by default all properties of extended schema gets ordered before the current schema properties. Use this field to order current resource properties before extended resource properties.

## Indexes

Developer can specify on which columns index should be added.

Each index has name and following options:

- columns
    List of column names to be indexed
- type (optional)
    Index type, available options are:
    - mysql - "spatial", "fulltext", "unique"
    - sqlite - "unique"

eg.
```yaml
      indexes:
        index_name:
          columns:
          - tenant_id
          - name
          type: "unique"
```

Parent - child relationship
-------------------------------

Resources can be in a parent-child relationship. It means that the child resource has a foreign key to its parent, and utilized for UI and CLI.

Gohan adds <parent>_id property automatically when Gohan loads schemas.

eg.

```yaml
        schemas:
        - description: Test Device
          id: test_device
          parent: ""
          singular: test_device
          plural: test_devices
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
            required:
            - segmentation_type
            - segmentation_id
            type: object
          title: Test Device
        - description: Test Physical Port
          id: test_port
          parent: "test_device"
          singular: test_port
          plural: test_ports
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
            type: object
          title: Test Physical Port
```

## Custom actions schema

Resources can have custom actions, besides CRUD. To define them, add "actions" section and define JSON schema of allowed input format

eg.

```yaml
        schemas:
        - description: Server
          id: server
          parent: ""
          singular: server
          plural: server
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
              management_ip:
                default: ""
                format: ipv4
                permission:
                - create
                - update
                title: Management IP
                type: string
                unique: false
              id:
                permission:
                - create
                title: ID
                type: string
                format: uuid
          actions:
            reboot:
              path: /:id/reboot
              method: POST
              input:
                type: object
                properties:
                  message:
                    type: string
                  delay:
                    type: string
              output: null
```

Then, register extension to handle it, e.g.

```
  gohan_register_handler("action_reboot", function(context){
    // handle reboot in southbound
  });
```

In order to query above action, POST to /v1.0/servers/:id/action with

```yaml
  {
    "reboot": {
      "message": "Maintenance",
      "delay": "1h"
    }
  }
```

## Custom Isolation Level

Developers can specify the transaction isolation level for API requests when Gohan is configured to connect MySQL database.
The default setting is "read repeatable" for read operations and "serializable" for operations that modify the database (create, update, delete) and sync operation. (state_update, monitoring_update). The default for unspecified action is repeatable read.

```yaml
    isolation_level:
      read:  REPEATABLE READ
      create:  SERIALIZABLE
      update: SERIALIZABLE
      delete: SERIALIZABLE
```

## OpenAPI / Swagger

Gohan schema is supposed to define "Data Model," whereas OpenAPI/Swagger is supposed to define API.

You can generate OpenAPI / Swagger file from Gohan schema so that you can afford swagger utility tools.

```

    gohan openapi --config-file etc/gohan.yaml

    # or you can customize template file using

    gohan openapi --config-file etc/gohan.yaml --template etc/templates/swagger.tmpl
```

then you will get swagger.json.
You can use this file for using swagger utility tools.

For example, you can use go-swagger to generate go related code. (see http://goswagger.io/)

```
    $ swagger validate swagger.json
    The swagger spec at "swagger.json" is valid against swagger specification 2.0
```

# API

In this section, we show how we generate REST API based on a schema.

"$plural", "$singular", "$prefix" and "$id" are read directly from schema,
"$namespace_prefix" is computed using namespace information and might be empty if schema has no namespace specified.

Note: An extension computes actual access URL for each resource and substitutes prefix property with it during schema listing calls. User can list resources using this URL and access a single instance of resource by prepending "/$id"
suffix.

## List REST API

List supports pagination by optional GET query parameters ``sort_key`` and ``sort_order``.

Query Parameter   Style       Type           Default           Description
sort_key          query       xsd:string     id                Sort key for results
sort_order        query       xsd:string     asc               Sort order - allowed values are ``asc`` or ``desc``
limit             query       xsd:int        0                 Specifies maximum number of results.
                                                               Unlimited for non-positive values
offset            query       xsd:int        0                 Specifies number of results to be skipped
<parent>_id       query       xsd:string     N/A               When resources which have a parent are listed,
                                                               <parent>_id can be specified to show only parent's children.
<property_id>     query       xsd:string     N/A               filter result by property (exact match). You can use multiple filters.

When specified query parameters are invalid, server will return HTTP Status Code ``400`` (Bad Request)
with an error message explaining the problem.

To make navigation easier, each ``List`` response contains additional header ``X-Total-Count``
indicating number of all elements without applying ``limit`` or ``offset``.

Example:
GET http://$GOHAN/[$namespace_prefix/]$prefix/$plural?sort_key=name&limit=2

Response will be

HTTP Status Code: 200

```json

  {
    "$plural": [
       {
         "attr1": XX,
         "attr2": XX
       }
    ]
  }

```

### Child resources access

Gohan provides two paths for child resources.

Full path
  To access a child resource in that way, we need to know all it parents.

  e.g. POST http://$GOHAN/[$namespace_prefix/]$prefix/[$ancestor_plural/$ancestor_id/]$plural

Short path

  e.g. POST http://$GOHAN/[$namespace_prefix/]$prefix/$plural?$parent_id=<parent_id>

## GET

Show REST API

GET http://$GOHAN/[$namespace_prefix/]$prefix/$plural/$id

Response will be

HTTP Status Code: 200

```json
  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }
```

## CREATE

CREATE Resource REST API

POST http://$GOHAN/[$namespace_prefix/]$prefix/$plural/

Input

Note that input JSON can only contain if you set "create" permission for this

HTTP Status Code: 202

```json
  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }
```

Response will be

```json
  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }
```

## Update

Update Resource REST API

PUT http://$GOHAN/[$namespace_prefix/]$prefix/$plural/$id

Input

Note that input JSON can only contain if you set "update" permission for this

```json
  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }
```

Response will be

HTTP Status Code: 200

```json
  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }
```

## DELETE

Delete Resource REST API

HTTP Status Code: 204

DELETE http://$GOHAN/[$namespace_prefix/]$prefix/$plural/$id


## Custom Actions

Run custom action on a resource

POST http://$GOHAN/[$namespace_prefix/]$prefix/$plural/$id/$action_path

Input

Input JSON can only contain parameters defined in the input schema definition. It requires "$action" allow policy

```json
  {
    "parameter1": XX,
    "parameter2": XX
  }
```

Response will be

HTTP Status Code: 200

```json
  {
    "output1": XX,
    "output2": XX
  }
```
