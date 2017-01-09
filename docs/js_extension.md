# Javascript extension

Gohan support a extension written by JavaScript thanks to Otto project.
In the gohan extension code, you need to register context using
gohan_register_handler function.
gohan_register_handler talkes event_type (string)_ and handler (function(context)).

```
  gohan_register_handler("pre_show", function(context){
    context.resp = "pre_show event"
  });

  gohan_register_handler("pre_update", function(context){
    context.resp = "pre_update event "
  });
```

context has following items

  context.schema : schema information
  context.path : url path
  context.params : query params
  context.role : user role
  context.auth : auth_context information
  context.http_request : Go HTTP request object
  context.http_response : Go HTTP response writer object


## Build in exception types

In an effort to simplify writing extensions for validation Gohan supports
throwing some exceptions and handles them internally.
Gohan provides the following exception types.

- BaseException(msg)

The base type of exception, should never be raised as itself, only extended.

- CustomException(msg, code)

A BaseException with an additional code property. When thrown will result in
an http response with the provided code and message being written.

One can extend the CustomException. An example follows.

```
  function ValidationException(msg) {
    CustomException.call(this, msg, 400);
    this.name = "ValidationException";
  }
  ValidationException.prototype = Object.create(CustomException.prototype);
```

### Build in javascript functions

Gohan extension supports some build-in functions.

- ``gohan_log_critical(message)``
- ``gohan_log_error(message)``
- ``gohan_log_warning(message)``
- ``gohan_log_notice(message)``
- ``gohan_log_info(message)``
- ``gohan_log_debug(message)``

  log ``message`` in Gohan log.

  ``gohan_log_<lowercase level>(message)`` is equivalent to
  ``gohan_log(MODULE, LOG_LEVEL.<uppercase level>, message)``.

- ``gohan_log(module, log_level, message)``

  log ``message`` in Gohan log (general version).

  - ``module``
    The module to be used for logging. You can use ``LOG_MODULE`` for
    the current log module. See ``gohan_log_module_push``.

  - ``log_level``
    One of ``LOG_LEVEL.CRITICAL``, ``LOG_LEVEL.ERROR``,
    ``LOG_LEVEL.WARNING``, ``LOG_LEVEL.NOTICE``, ``LOG_LEVEL.INFO``,
    ``LOG_LEVEL.DEBUG``.

  Example usage::

    gohan_log(LOG_MODULE, DEBUG, "It works");

  This will print something like the following::

    17:52:40.921 gohan.extension.network.post_list_in_transaction DEBUG  It works

- ``gohan_log_module_push(new_module) : <old log module>``
  Appends ``new_module`` to the current ``LOG_MODULE``.

- ``gohan_log_module_restore(old_module)``
  Restores ``LOG_MODULE`` to ``old_module``. Example usage::

    old_module = gohan_log_module_push("low_level");
    try {
        ...
    } finally {
        gohan_restore_log_module(old_module)
    }

- gohan_http(method, url, headers, data, opaque, timeout)

fetch data from url
method : GET | POST | PUT | DELETE
url : destination url
headers : additional headers (eg. AUTH_TOKEN)
data : post or put data
opaque (optional) : boolean - whether to parse URL or to treat it as raw
timeout (optional) : int - timeout in milliseconds. Default is no timeout.

- gohan_raw_http(method, url, headers, data)

Fetch data from url. It uses GO RoundTripper instead of Client (as in gohan_http),
which allows for more control.
method : GET | POST | PUT | DELETE
url : destination url
headers : additional headers (eg. AUTH_TOKEN)
data : request data (string)

- gohan_config(key, default_value)

get value from Gohan config. If key is not found, default value is returned.
Key should be specified as
`JSON Pointer <http://tools.ietf.org/html/draft-ietf-appsawg-json-pointer-07>`_.

- gohan_db_list(transaction, schema_id, filter_object[, order_key[, limit[, offset]]])

retrive all data from database

- gohan_db_fetch(transaction, schema_id, id, tenant_id)

get one data from db

- gohan_db_query(transaction, schema_id, query_string, arguments)

  Retrieve data with a raw query

  - transaction: The transaction to use. When null will use a new one.
  - schema_id: The ID of the schema of which the query result populates instances
  - query_string: Raw query string such as a SQL SELECT query

    - You can put a "?" as the placeholder of variables in your query string

  - arguments: An array of actual values that replace place holders in query_string

- gohan_db_create(transaction, schema_id, object)

create data in db

- gohan_db_update(transaction, schema_id, object)

update data in db

- gohan_db_state_update(transaction, schema_id, object)

update data in db without informing etcd

- gohan_db_delete(transaction, schema_id, object)

delete data in db

- gohan_db_transaction()

start a new DB transaction. You are responsible for managing tranansactions created by this function. Call .Close() or .Commit() after using the return value.


- gohan_model_list(context, schema_id, filter)

  Retrieve data through Gohan.

  - context: You need to have transaction in this dictionary which you can get from given context
  - schema_id: The id of the schema of the objects we want to retrieve.
  - filter: How to filter retrieved objects. Should be a dictionary with each key being either:

    - A property of the schema we are retrieving. Then the value has to either be a string or an array of strings.
      The response is then filtered by removing all entries that do not have the value for the given key in the provided array.
    - Any of the strings 'sort_key', 'sort_order', 'limit', 'offset'. These are interpreted with their values as query parameters.

- gohan_model_fetch(context, schema_id, resource_ids)

  Retrieve a specific resource through Gohan.

  - context: You need to have transaction in this dictionary which you can get from given context
  - schema_id: The id of the schema of the object we want to retrieve.
  - resource_id: The id of the object we want to retrieve.
  - tenant_ids: allowed tenant id

- gohan_model_create(context, schema_id, data)

  Create an object through Gohan.

  - context: You need to have transaction in this dictionary which you can get from given context
  - schema_id: The id of the schema of the object we want to create.
  - data: The data needed to create the object, in the form of a dictionary.

- gohan_model_update(context, schema_id, resource_id, data, tenant_ids)

  Update an object through Gohan.

  - context: You need to have transaction in this dictionary which you can get from given context
  - schema_id: The id of the schema of the object we want to update.
  - resource_id: The id of the object we want to update.
  - data: The data needed to update the object, in the form of a dictionary.
  - tenant_ids: allowed tenant id

- gohan_model_delete(context, schema_id, resource_id)

  Delete an object through Gohan.

  - context: You need to have transaction in this dictionary which you can get from given context
  - schema_id: The id of the schema of the object we want to delete.
  - resource_id: The id of the object we want to delete.

- gohan_schemas()

returns all registered schemas

- gohan_schema_url(schema)

returns the url for the schema

- gohan_policies()

returns all policies

- gohan_uuid()

generate uuid v4

- gohan_sleep(time)

sleep time (ms)

- gohan_execute(comand_name, args)

execute shell command

- gohan_template(template_string, variables)

apply go style template

- gohan_netconf_open(hostname, username)

open netconf session.
(Note: you need set up ssh key configuraion
on both of gohan and target node.)
In gohan, you need to setup ssh/key_file
configuraion.

- gohan_netconf_exec(session, command)

execute netconf command

- gohan_netconf_close(session)

close netconf session

- gohan_ssh_open(hostname, username)

open ssh session.
(Note: you need set up ssh key configuraion
on both of gohan and target node.)
In gohan, you need to setup ssh/key_file
configuraion.

- gohan_ssh_exec(session, command)

execute command on ssh session

- gohan_ssh_close(session)

close ssh session

- require(module)

Dynamically load modules loaded in source code or
installed via npm in node_modules at working directory

- gohan_file_list(dir)

List files in dir

- gohan_file_read(path)

Read file from path

- gohan_file_dir(path)

Check if dir

- gohan_sync_fetch(path)

Fetch a given path from Sync   

- gohan_sync_watch(path, timeout, revision)

Watch a given path in Sync starting from a given revision. This call is blocking no longer
than a given timeout in milliseconds. If no event occurs in the given timeout, the function
returns an empty object.

# Testing javascript extensions

You can test extensions using a testing tool bundled with Gohan with the command
``test_extensions`` (or ``test_ex`` for short). Build and install Gohan, then
run ``gohan test_extensions <paths to files/directories to test>``. The
framework will walk through files and recursively through directories, running
tests in files named ``test_*.js``.

By default, the framework doesn't show logs and results for passing tests, so
you won't see any output if all the tests pass. If you pass a
``-v``/``--verbose`` flag, it will show these messages, and an additional ``All
tests have passed.`` message if all the tests pass.

## Test file contents

Each test file must specify schema and path for preloading extensions:

* var SCHEMA - path to the schema that stores extensions to be tested
* var PATH - path for preloading extensions

Additionally each file can specify:

* one setUp() function that will be called before each test
* one tearDown() function that will be called after each test
* multiple test_<name>() functions that will be called by the framework
* multiple helper functions and variables, with names not starting with prefix
  ``test_``

## Framework API

Test framework provides all built in function mentioned in subsection
describing `gohan built in functions`_.

In unit tests, you can use mocks of ``gohan_http``, ``gohan_config`` and
``gohan_db_transaction``. You can pass values that will be returned for given
arguments during subsequent calls by calling
``gohan_*.Expect(argument, ...).Return(value)``. One call to
``gohan_*.Expect(arguments, ...).Return(value)`` provides one response of
``gohan_*`` (FIFO queue). If no return value, or wrong arguments are provided
for a call then an unexpected call is assumed, which will result in test failures.

In addition to the abovementioned functions, the framework provides the
following API:

* ``Fail(format_string, ...)`` - stop execution of a single test case and
  return an error

* ``GohanTrigger(event_type, context) : <new context>`` - triggers a specified
  type of Gohan event

  * ``event_type`` - one of the event types recognized by Gohan (see
    event_ subsection)

  *  ``context`` - context passed to the event handler

* ``MockTransaction(is_new) : <mock transaction>`` - return a mock transaction that
  can be used with built-in Gohan methods. Each test is run using a separate
  database that is deleted after ``tearDown()``, so there is no need to
  clean up the database between tests. Multiple calls to ``MockTransaction()``
  within a single ``setUp()``, test, ``tearDown()`` routine when no call to
  ``CommitMockTransaction()`` has been made will yield the same transaction.
  ``MockTransaction(true)`` returns forcibly new transaction.

* ``CommitMockTransaction()`` - commit and close the last nonclosed
  transaction. After this call any calls to ``MockTransaction()`` return
  a new transaction.

* ``MockPolicy() : <mock policy>`` - return a mock policy that
  can be used with built-in Gohan methods.

* ``MockAuthorization() : <mock authorization>`` - return a mock authorization that
  can be used with built-in Gohan methods.

## Example
A sample test may look like this:

```
  // Schema file containing extensions to be tested
  var SCHEMA = "../test_schema.yaml";

  /**
   * Sample contents of test_schema.yaml:
   *
   * extensions:
   * - id: network
   *   path: /v2.0/network.*
   *   url: file://./etc/examples/neutron/network.js
   * - id: exceptions
   *   path: ""
   *   url: file://./etc/examples/neutron/exceptions.js
   * - id: urls
   *   path: /gohan/v0.1/schema.*
   *   url: file://./etc/examples/url.js
   * schemas:
   * - description: Network
   *   id: network
   *   parent: ""
   *   plural: networks
   *   schema:
   *     properties:
   *       id:
   *         format: uuid
   *         permission:
   *         - create
   *         title: ID
   *         type: string
   *         unique: true
   *       tenant_id:
   *         format: uuid
   *         permission:
   *         - create
   *         title: Tenant id
   *         type: string
   *         unique: false
   *     propertiesOrder:
   *     - name
   *     - id
   *     - tenant_id
   *   singular: network
   *   title: Network
   */

  // With the following PATH, "network" and "exceptions" extensions will be loaded
  var PATH = "/v2.0/networks";

  /**
   * Sample contents of network.js:
   *
   * // filter removes the network with the unwanted id
   * gohan_register_handler("post_list", function filter(context) {
   *     // This call will be mocked, see testNetworkListFilter below
   *     response = gohan_http("GET", "http://whatisunwanted.com", {}, null);
   *
   *     for (var i = 0; i < context.response.networks.length; i++) {
   *         if (context.response.networks[i].id == response.unwanted) {
   *             context.response.networks.splice(i, 1);
   *             break;
   *         }
   *     }
   * });
   */

  var context;
  var network;

  function setUp() {
      var network_to_create = {
          'id': 'new',
          'tenant_id': 'azerty'
      };
      network = gohan_db_create(MockTransaction(), "network", network_to_create);
      context = {
          'schema': { /* ... */ },
          'http_request': { /* ... */ },
          'http_response': { /* ... */ },
          'path': '/gohan/v0.1/schema',
          'response': {
              'networks': [
                  network,
                  {
                      'id': 'foo',
                      'tenant_id': 'xyz'
                  }
               ]
           }
      }
  }

  function tearDown() {
    gohan_db_delete(MockTransaction(), "network", "new");
  }

  function testNetworkListFilter() {
      // First call to gohan_http will return {'unwanted': 'foo'}
      gohan_http.Expect("GET", "http://whatisunwanted.com", {}, null).Return({'unwanted': 'foo'});
      // Second call to gohan_http will return empty response
      gohan_http.Expect("GET", "http://whatisunwanted.com", {}, null).Return({});
      // Subsequent calls to gohan_http will fail since they are not expected
      var new_context = GohanTrigger('post_list', context);

      if (new_context.response.networks.length != 1) {
         Fail('Expected 1 network but %d found.', new_context.response.networks.length);
      }

      if (new_context.response.networks[0].id != network.id) {
         Fail('Expected network with id "%s" but "%s" found.', network.id, new_context.response.networks[0].id);
      }
  }

  function testSomethingElse() {
      /* ... */
  }
```