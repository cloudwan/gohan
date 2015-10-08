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
- condition : addtional condition (see below)
- tenant_id : regexp matching the tenant, defaults to ``.*``

----------
Conditions
----------

Gohan supports two types of conditions

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
    principal: _member_
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
    principal: _member_
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
    principal: _member_
    resource:
      path: /v2.0/server/?$

