# Policy

You can configure API access policy using this resource.
A policy has following properties.

- id : ID of the policy
- principal : Keystone Role
- action: one of `create`, `read`, `update`, `delete` for CRUD operations
  on the resource or any custom actions defined by schema performed on a
  resource or `*` for all actions
- effect : Allow API access or not
- resource : target resource
  you can specify target resource using "path" and "properties"
- condition : additional condition (see below)
- tenant_id : regexp matching the tenant, defaults to ``.*``

## Conditions

Gohan supports several types of conditions

- `is_owner` - Gohan will enforce access privileges for the resources specified in the policy. By default access to resources of all other tenants would be blocked.

- belongs_to - Gohan will apply the policy if the user tries to access resources belonging to the tenant specified in condition (see the example below). The condition has no effect if the access privileges are not enforced by specifying the `is_owner` condition. The full condition looks like:

  - `action: (*|create|read|update|delete)`
     `tenant_id: 8bab8453-1bc9-45af-8c70-f83aa9b50453`
     `type: belongs_to`

Example policy

``` yaml
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
```

-  type `property` - You can add a condition based on resource value.

  You can specify allowed values in a match.
  if it is a value, we check exact match.
  if it is a list, we check if the value is in the list
  if it is a dict, we check if we have a key for this value and, updated value matches it.
  Note that this is only valid for update action.

``` yaml
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
```

## Resource paths with no authorization (nobody resource paths)

With a special type of policy one can define a resource path that do not require authorization.
In this policy only 'id', 'principal' and 'resource.path' properties are used. Policy 'principal'
is always set to 'Nobody'.

``` yaml
policies:
- id: no_auth_favicon
  principal: Nobody
  resource:
    path: /favicon.ico
- id: no_auth_member_resources
  action: '*'
  principal: Nobody
  resource:
    path: /v0.1/member_resources*
```

In the above example, the access to favicon is always granted and never requires an authorization.
This feature is useful for web browsers and it is a good practice to set this policy.
In the second policy, no-authorization access is granted to all member resources defined by a path wildcard.
