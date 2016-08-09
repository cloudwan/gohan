==============
Gohan Schema
==============

Gohan server provides REST API based on gohan schema definitions.
You can write gohan schema using json or YAML format. We will use YAML format in this document because of human reability.

You can take a look example schema definitions in etc/examples/

Here is an example defining network and subnet resource.

.. code-block:: yaml

  schemas:
  - id: network
    namespace: neutron
    plural: networks
    prefix: /v2.0
    metadata:
      template_format: "template/network.yaml"
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
        providor_networks:
          default: {}
          permission:
          - create
          - update
          properties:
            segmentaion_type:
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
          default: []
          items:
            type: string
          permission:
          - create
          - update
          title: RouteTargets
          type: array
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
      - providor_networks
      - route_targets
      - tenant_id
      type: object
    singular: network
    title: Network
  - id: subnet
    parent: network
    plural: subnets
    prefix: /v2.0/neutron
    schema:
      properties:
        cidr:
          permission:
          - create
          title: Cidr
          type: string
          unique: false
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
          title: TenantID
          type: string
          unique: false
      propertiesOrder:
      - id
      - name
      - description
      - cidr
      - tenant_id
      type: object
    singular: subnet
    title: subnet


Schemas
-----------------------


We will have a list of schemas to define a setup resources.
Each schema will have following properties.

- id          -- resource id (unique)
- singular    -- singular form of the schema name
- plural      -- plural form of schema name
- title       -- use visible label of resource title
- description -- a description of the schema
- schema      -- json schema

Schemas might also have any of the following optional properties.

- parent    -- the id of the parent schema
- on_parent_delete_cascade -- cascading delete when parent resource deleted
- namespace -- resource namespace
- prefix    -- resource path prefix
- metadata  -- application specific schema metadata (object)
- type      -- can be abstract or empty string (see more in schema inheritance)
- extends   -- list of base schemas

You need these information to define REST API.
Please see json schema specification http://json-schema.org/

Note that each resource must have unique "id" attribute for identity for the
each resources. You should also define "tenant_id" attribute if you want to
use owner-based access control described in policy section later. In case
no tenant_id is specified and owner-based access control is not enabled,
tenant_id will be assigned based on the authentication token used.

"singular" and "plural" attributes are used for wrapping returned resources
in additional dictionary during show and list calls respectively.
"plural" is also used during access URL constructions.

Namespace is an optional parameter that can be used to group schemas. If
a namespace has been specified, full namespace prefix will be prepended to the
schema prefix- see :ref:`namespace section <section-namespace>` for details.

You can use following properties in json schema.

Schema Inheritance
-------------------------------

You can define an abstract schema by setting type="abstract".
Schemas can be derived from an abstract schema by using the keyword "extends".
JSON schema, metadata and action will be merged when a schema is extended.
prefix value and parent value will be set if not specified.

.. code-block:: yaml

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
            segmentaion_type:
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



Metadata
-------------------------------

- type (string)

  you can specify schema type. For example, gohan-webui will use
  this value to determine wheather we show this schema link in the side menu

- nosync (boolean)

  if nosync is true, we don't sync this resource for sync backend.

- state_versioning (boolean)

  whether to support :ref:`state versioning <subsection-state-update>`, defaults to false.

- sync_key_template (string)

  configurable sync key path for schemas based on properties, for example: /v1.0/devices/{{device_id}}/virtual_machine/{{id}},
  it must contain '{{id}}'

Properties
-------------------------------

We need to define properties of resource using following parameters.

- title

  User visible label of the property

- format

  Additional validation hint for this property
  you can use defined attribute on http://json-schema.org/latest/json-schema-validation.html#anchor107

- type

  properties type.
  you can select from (string, number, integer, boolean, array and object)
  Note that schema itself should be object type.
  This can also be a two element list in case, attribute can be specified as null, e.g. ["string", "null"]

- default

  defualt value of the property

- enum

  You can specify list of allowed values

- required

  List of required attributes to specified during creation


Following properties are extended from json schema v4.

- permission

  permission is a list of allow actions for this property.
  valid values contrains "create", "update".
  Gohan generates json schema for craete API and update API based on this value.
  Note that we can use this property for only first level properties.

- unique boolean (unique key constraint)

- detail (array)

  This parameter will be used in user side. Possible values are strings including read, create, delete, list, update.


type string
-------------------------------

type string is for defining string.
You can use following parameters for string.

- minLength  max length of string
- maxLength  min length of string
- pattern    regexp pattern for this string
- relation  (gohan extened property)  define resource relation
- relation_property  (gohan extened property) relation resource will be joined in list api requiest for this property name
- on_delete_cascade  (gohan extened property) cascading delete when related resource get deleted

eg.

.. code-block:: yaml

        name:
          permission:
          - create
          - update
          title: Name
          type: string
          unique: false

type boolean
-------------------------------

type boolean for boolean value

eg.

.. code-block:: yaml

        admin_state:
          permission:
          - create
          - update
          title: admin_state
          type: boolean
          unique: false


type integer or type number
-------------------------------

type integer or type number for numetric properties.
You can use following parmeters to define valid range

- maximum (number) and exclusiveMaximum (boolean)
- minimum (number) and exclusiveMinimum (boolean)

eg.

.. code-block:: yaml

        age:
          permission:
          - create
          - update
          title: age
          type: number
          unique: false

type array
-------------------------------

type array is for defining list of elements

- items

  Only allowed for array type
  You can define element type on this property.

eg.

.. code-block:: yaml

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


type object
-------------------------------

Object type is for defining object in the resources.
Note that resource itself should be an object.
following parameters supported in object type.

- properties

  Only allowed for object type
  You can define properties of this object

- propertiesOrder (extended parameter in gohan)

  Only allowed for object type
  JSON has no ordering on object key. This could be problematic if you want to
  generate UI. so you can define ordering of properties using propertiesOrder

eg.

.. code-block:: yaml

        providor_networks:
          default: {}
          permission:
          - create
          - update
          properties:
            segmentaion_type:
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



Parent - child relationship
-------------------------------

Resources can be a in parent - child relationship. It means that the child resource has a foreign key to its parent.


Note that there is no need to create <parent>_id property in child schema, it is added automatically when the schema is loaded to gohan.

eg.

.. code-block:: yaml

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


Custom actions schema
-------------------------------

Resources can have custom actions, beside CRUD. In order to define them, add actions section and define jsonschema
of allowed input format

eg.

.. code-block:: yaml

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

Then, register extension to handle it, e.g.

.. code-block:: javascript

  gohan_register_handler("action_reboot", function(context){
    // handle reboot in southbound
  });

In order to query above action, POST to /v1.0/servers/:id/action with

.. code-block:: json

  {
    "reboot": {
      "message": "Maintenance",
      "delay": "1h"
    }
  }

Custom Isolation Level
-------------------------

You can specify the transaction isolation level for api requests.
Currently, this is only supported for mysql.
The default setting is "read repeatable" for read operations and "serializable" for
operations that modify the database (create, update, delete) and sync operations
(state_update, monitoring_update). The default for unspecified action is repeatable read.

.. code-block:: yaml

    isolation_level:
      read:  REPEATABLE READ
      create:  SERIALIZABLE
      update: SERIALIZABLE
      delete: SERIALIZABLE

OpenAPI / Swagger
------------------

Gohan schema is supposed to define "Data Model", whereus OpenAPI/Swagger
is supposed to define "API".

You can generate OpenAPI / Swagger file from gohan schema, so that you can
afford swagger utility tools.

.. code-block:: shell

    gohan openapi --config-file etc/gohan.yaml

    # or you can customize template file using

    gohan openapi --config-file etc/gohan.yaml --template etc/templates/swagger.tmpl

then you will get swagger.json.
You can use this file for using swagger utility tools.

For example, you can use go-swagger to generate go related code. (see http://goswagger.io/)

.. code-block:: shell

    $ swagger validate swagger.json
    The swagger spec at "swagger.json" is valid against swagger specification 2.0