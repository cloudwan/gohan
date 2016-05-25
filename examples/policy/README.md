Gohan Policy example
---------------------

In this example, we show how we can use a policy for API.

Fake Keystone server
---------------------

Gohan provides you a fake keystone server for quick test.
The fake keystone server has following resources.


Tenant

- demo

Users ("gohan" is password for all)

- admin ( demo tenant )
- member (demo tenant )

Policy
-------

We have "member_resource" and "admin_only_resource" schemas in this example.

An admin user have all CRUD access for all resources.
A member user can only see member_resources except
admin_property.

We can use this example policy to implement a policy above.

``` yaml
policies:
# Allow access for schemas
- action: read # limit for only read
  effect: allow # allow access
  id: member_schema # unique id for this policy
  principal: Member # member role
  resource:
    path: /gohan/v0.1/schemas* # resource path
# Allow access for member_resource
- action: '*' # allow any action
  condition:
  - is_owner # access limited only if a member is owner of the resource
  effect: allow # allow access
  id: member_policy
  principal: Member
  resource:
    path: /v0.1/member_resources*
    properties: # limit properties here
    - id
    - name
    - description
    - tenant_id
    # admin_only_resource is excluded here
```
