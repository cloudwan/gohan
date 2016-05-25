==============
Policy
==============

You can configure API access policy using this resource.
Policy has following properties.

- id : Identitfy of the policy
- principal : Keystone Role
- action: one of `create`, `read`, `update`, `delete` for CRUD operations
  on resource or any custom actions defined by schema performed on a
  resource or `*` for all actions
- effect : Allow api access or not
- resource : target resource
  you can specify target resource using "path" and "properties"
- condition : additional condition (see below)
- tenant_id : regexp matching the tenant, defaults to ``.*``

----------
Conditions
----------

Gohan supports several types of conditions

- :code:`is_owner` - Gohan will enforce access privileges for the resources
  specified in the policy. By default access to resources of all other tenants
  will be blocked.

- :code:`type: belongs_to` - Gohan will apply the policy if the user tries
  to access resources belonging to the tenant specified in condition (see the
  example below). The condition has no effect if the access privileges are not
  enforced by specifying the :code:`is_owner` condition. The full condition
  looks like:

  - :code:`action: (*|create|read|update|delete)`

    :code:`tenant_id: 8bab8453-1bc9-45af-8c70-f83aa9b50453`

    :code:`type: belongs_to`

Example policy

.. code-block:: yaml

  policies:
  - action: '*'
    effect: allow
    id: admin_statement
    principal: admin
    resource:
      path: .*
  - action: 'read'
    condition:
    - is_owner
    - type: belongs_to
      action: '*'
      tenant_id: 8bab8453-1bc9-45af-8c70-f83aa9b50453
    effect: allow
    id: member_statement
    principal: Member
    resource:
      path: /v2.0/network/[^/]+/?$
      properties:
      - id
      - description
      - name
  - action: '*'
    condition:
    - is_owner
    effect: allow
    id: member_statement2
    principal: Member
    resource:
      path: /v2.0/networks/?$
      properties:
      - id
      - description
      - name
  - action: 'reboot'
    condition:
    - is_owner
    effect: allow
    id: member_statement2
    principal: Member
    resource:
      path: /v2.0/server/?$

- :code: type `property` - You can add condition based on resource value.
  You can specify allowed values in match.
  if it is a value, we check exact match.
  if it is a list, we check if the value is in the list
  if it is a dict, we check if we have a key for this value and, updated value matches it.
  Note that this is only valid for update action.

.. code-block:: yaml
    policy:
      - action: 'read'
        condition:
        - type: property
          match:
            status:
              - ACTIVE
              - CREATE_IN_PROGRESS
              - UPDATE_IN_PROGRESS
              - DELETE_IN_PROGRESS
              - ERROR
        effect: allow
        id: member
        principal: Member
      - action: 'update'
        condition:
        - type: property
          match:
            status:
              ACTIVE:
              - UPDATE_IN_PROGRESS
              - ERROR
        effect: allow
        id: member
        principal: Member
      - action: 'reboot'
        condition:
        - type: property
          match:
            status: ACTIVE
        effect: allow
        id: member
        principal: Member
      - action: 'delete'
        condition:
        - type: property
          match:
            status:
            - ACTIVE
            - ERROR
        effect: allow
        id: member
        principal: Member