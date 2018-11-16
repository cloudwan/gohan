# Policy

You can configure API access policy using this resource.
A policy has following properties.

- id : ID of the policy
- principal : Keystone Role
- action: one of `create`, `read`, `update`, `delete` for CRUD operations
  on the resource or any custom actions defined by schema performed on a
  resource or `*` for all actions
- effect : Allow API access or not. `Deny` keyword (case insensitive) block access, any other option (including lack of this property) allows
- resource : target resource
  you can specify target resource using "path" and "properties"
- condition : additional condition (see below)
- tenant_id : regexp matching the tenant, defaults to ``.*``
- scope: type of the token's scope. Must be a list of strings. If not provided, the policy matches on all token types. Possible values are:
  - "tenant" - matches tokens scoped to tenant,
  - "domain" - matches tokens scoped to domain,
  - "admin" - matches tokens scoped to admin project.

## Conditions

Gohan supports several types of conditions

- `is_owner` - Gohan will enforce access privileges for the resources specified in the policy. By default access to resources of all other tenants would be blocked.
- `is_domain_owner` - Gohan will allow access only to resources from the same domain.

- belongs_to - Gohan will apply the policy if the user tries to access resources belonging to the tenant specified in condition (see the example below). The condition has no effect if the access privileges are not enforced by specifying the `is_owner` condition. The full condition looks like:

  - `action: (*|create|read|update|delete)`
     `tenant_id: 8bab8453-1bc9-45af-8c70-f83aa9b50453`
     `type: belongs_to`

Example policy

```yaml
  policies:
  - action: '*'
    effect: allow
    id: admin_statement
    principal: admin
    scope:
    - admin
    resource:
      path: .*
  - action: '*'
    condition:
    - is_owner
    effect: allow
    id: admin_domain_statement
    principal: admin
    scope:
    - domain
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

```yaml
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

- `and` and `or` - allows creating more complicated policy filters.

`and` checks that all conditions have been met.

`or` checks that at least one of the conditions has been met.

Both of this conditions might be nested and might be used separately.
Please note that unlike `type: property`, described above, these conditions affect the SQL query.

`and` and `or` contain a list of conditions that have to be met.
Those conditions may include:

- `is_owner` - restricts access only to the owner of the resource
- `is_domain_owner` - restricts access only to resources from the same domain as current user
- `and` - list of conditions that all have to be met
- `or` - list of conditions from which at least one have to be met
- `match` - dictionary to match

`match` has to contain the following properties:

- `property` - name of the resource property which has to be checked
- `type` - condition that has to be met for the match - currently only `eq` (equal) and `neq` (not equal) operators are available
- `value` - allowed values

`value` may consist of one or multiple values.
For one element exact match is required.
For a list, all the values from the list are checked and only one is required.

Example below presents policy for which member is able to read all own resources.
For the resources of other members he will only see resources for which `state` property is equal to `UP`
and `level` is equal to 2 or 3.

```yaml
policy:
  - action: read
    effect: allow
    id: member
    principal: Member
    condition:
      - or:
        - is_owner
        - and:
          - match:
              property: state
              type: eq
              value: UP
          - match:
              property: level
              type: eq
              value:
                - 2
                - 3
```

## Resource paths with no authorization (nobody resource paths)

With a special type of policy one can define a resource path that do not require authorization.
In this policy only 'id', 'principal' and 'resource.path' properties are used. Policy 'principal'
is always set to 'Nobody'.

```yaml
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

## Resource properties

It is possible to filter allowed fields using `properties` or `blacklistProperties` properties.
It is not possible to use both `properties` and `blacklistProperties` at the same time.
In case of having both types of properties error is returned on the server start.
- `properties` defines properties exposed to the user, the other properties are forbidden.
- `blacklistProperties` defines forbidden properties, the other properties are exposed.

```yaml
- action: read
  effect: allow
  id: visible_properties_read
  principal: Visible
  condition:
  - type: property
    match:
      is_public:
      - true
  resource:
    path: /v2.0/visible_properties_test.*
    properties:
    - a
- action: read
  effect: allow
  id: visible_properties_read
  principal: Hidden
  condition:
  - type: property
    match:
      is_public:
      - true
  resource:
    path: /v2.0/visible_properties_test.*
    blacklistProperties:
    - id
    - a
    - is_public
```

## Effect property

While checking access to given method Gohan will check all policies.
If any policy matches it is later check if it is allowed.
Even one policy that matches method and is marked as not allowed is sufficient to block access.
In an example below admin has access to all methods in all paths except `delete` action in `/v2.0/restricted_method.*`.

```yaml
- action: '*'
  effect: allow
  id: admin_allow_all
  principal: admin
  resource:
    path: .*
- action: delete
  effect: deny
  id: admin_deny_delete
  principal: admin
  resource:
    path: /v2.0/restricted_method.*
```

## Attach policy

Attach policy defines allowed values for a relation property
(a resource "attaches" to another resource by having a relation to it).
Such policy defines restrictions for a property between a _source_ and a _target_ resource.
The syntax is slightly different than in the case of a standard policy:

- id, principal, resource, tenant_id: as in the case of a standard policy
- action: Must be `__attach__`
- effect: If the condition in the policy allows for the access, or denies it.
  Possible values (case insensitive, default is `allow`):
    - `allow`: policy denies if source or target conditions are false
    - `deny`: policy denies if source and target conditions are true
- resource: Like in the case of a standard policy, but considers the source resource
- relation_property: Name of the relation property the policy is applied to.
  Can be `*`, then it applies to all properties of the source resource.
  Note that nested relations are also supported - for the syntax, see examples.
- target_condition: Condition for the target resource (for syntax, see the 'Condition' section)

Attach policies are checked during creation/update of the _source_ resource.
An attach policy is applied only if relation field is non-nil.
If any of the policies deny access, then modification is forbidden and a 404 response is returned.
In contrast to standard policies, the fact that no attach policies apply to the resource does not restrict access to the resource.

Attach policies can also support relations defined via properties nested in other properties of array or object type. Some examples can be seen below.

An example:

```yaml
- action: '__attach__'
  id: attach_if_accessible
  effect: allow
  principal: Member
  relation_property: some_relation_id
  target_condition:
  - or:
    - is_owner
    - match:
        property: accessibility
        type: eq
        value: everybody
  resource:
    path: /v2.0/attacher.*
- action: '__attach__'
  id: deny_if_blocked
  effect: deny
  principal: Member
  relation_property: some_relation_id
  target_condition:
  - and:
    - is_owner
    - match:
        property: block_flag
        type: eq
        value: true
  resource:
    path: /v2.0/attacher.*
# relation_property should contain fully qualified name to the property, if it is nested.
# The example refers to the field "some_relation", which could be defined in schema like this:
# properties:
#   some_object:
#     type: object
#     properties:
#       some_relation:
#         ...
- action: '__attach__'
  id: relation_nested_in_object
  effect: deny
  principal: Member
  relation_property: some_object.some_relation
  target_condition:
  - is_owner
  resource:
    path: /v2.0/attacher.*
# Properties nested in arrays are also supported.
# The example refers to the field "some_relation", which could be defined in schema like this:
# properties:
#   some_array:
#     type: array
#     items:
#       type: object
#       properties:
#         some_relation:
#           ...
- action: '__attach__'
  id: relation_nested_in_array
  effect: deny
  principal: Member
  relation_property: some_array.[].some_relation
  target_condition:
  - is_owner
  resource:
    path: /v2.0/attacher.*
```
