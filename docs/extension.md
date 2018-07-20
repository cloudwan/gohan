# Extension

You can add additional logic using Gohan Extension.
Extensions has properties:

- id identity of the code
- code contents of a code
- code_type javascript and goext are supported
- URL placement of code. Currently, file://, http:// and https:// schemes are supported
- path resource path to execute code

Example Code

```yaml
  extensions:
  - code: console.log(Object.keys(context));
    id: test
    path: /v2.0/.*
```

Gohan supports two types of extensions:
- javascript
- golang (as a plugin, loadable at runtime) 

Comparison of different types of extensions:

| Name | type-id | language | execution performance | loadable at runtime |
| --- | --- | --- | --- | --- |
| javascript | javascript | javascript | interpreted | yes |
| golang (plugin) | goext | golang | native | yes |

## Event

### pre_list

  list event before DB operation

  context.response contains response data.
  You can also update response here

  Note you can skip DB operation if you set context response in here

### pre_list_in_transaction

  Same as pre_list but executed in the DB transaction context.transaction contains transaction object for DB operation

  id : request id

  context.response contains response data.
  You can also update response here

### post_list_in_transaction

  Same as post_list but executed in the DB transaction context.transaction contains transaction object for DB operation

  id : request id
  context.response contains response data.
  You can also update response here

### post_list

  list event after DB operation.

  context.response contains response data.
  You can also update response here

### pre_show

  show event before DB access

  id : request id
  context.response contains response data.
  You can also update response here

  Note you can skip DB operation if you set context response in here

### pre_show_in_transaction

  Same as pre_show but executed in the DB transaction context.transaction contains transaction object for DB operation

  id : request id
  context.response contains response data.
  You can also update response here

### post_show_in_transaction

  Same as post_show but executed in the DB transaction context.transaction contains transaction object for DB operation

  id : request id
  context.response contains response data.
  You can also update response here

### post_show

  show event after DB operation

  id : request id
  context.response contains response data.
  You can also update response here

### pre_create

  executed before creation
  Mainly used for validation purpose

  context.resource contains user input data

  Note you can skip DB operation if you set context response in here

### pre_create_in_transaction

  Same as pre_create but executed in the DB transaction context.transaction contains transaction object for DB operation

### post_create_in_transaction

  after creation in the transaction

### post_create

  after create
  context.response contains response data.
  context.transaction contains transaction object for db operation

### pre_update

  executed before an update
  Mainly used for validation purpose

  context.resource contains user input data

  Note you can skip DB operation if you set context response in here

### pre_update_in_transaction

  same as pre_update but executed in the db transaction
  context.transaction contains transaction object for db operation

### post_update_in_transaction

  after creation in transaction

### post_update

  after update
  context.response contains response data.
  context.transaction contains transaction object for db operation

### pre_delete

  executed before delete
  Mainly used for validation purpose

  context.id contains resource id we are trying to delete
  context.transaction contains transaction object for db operation

### pre_delete_in_transaction

  same as pre_delete but executed in the db transaction
  context.transaction contains transaction object for db operation

### post_delete_in_transaction

  after creation in transaction

### post_delete

  after delete

### pre_state_update_in_transaction

  executed before a state update triggered by a backend event

  context.resource contains the resource associated with the update,
  context.state contains the state changes,
  context.config_version contains the current config version

### post_state_update_in_transaction

  as above, but after the state update

### pre_monitoring_update_in_transaction

  executed before a monitoring update triggered by a backend event

  context.resource contains the resource associated with the update,
  context.monitoring contains the new monitoring information

### post_monitoring_update_in_transaction

  as above, but after the monitoring update

### notification

  executed when you receive a cron notification

# Context

Each handler receives 'context' object which is associated with http request.
Context type is a map with string keys.  
Each context contains following items:
- path - string - URL Path
- http_request - *http.Request - object representing API call
- http_response - map[string]interface{} - object representing API response
- request_data - map[string]interface{} - API request body
- schema - schema.Schema - schema defining given path
- schema_id - string - schema ID
- params - map[string]interface{} - HTTP request parameters
- sync - sync.Sync - Sync client
- DB - db.DB - DB client
- queue - *job.Queue - jobs queue
- identity_service - middleware.IdentityService - identity service
- service_auth - schema.Authorization - Autorization object
- openstack_client - *gophercloud.ServiceClient - OpenStack client
