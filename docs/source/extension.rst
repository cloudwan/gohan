==============
Extension
==============

You can add additional logic using Gohan Extension.
Extensions has properties:

- id identity of the code
- code contents of code
- code_type javascript, go and donburi (DSL) are supported
- url placement of code. currenty, file://, http:// and https:// schemes are supported
- path resource path to execute code

Example Code

.. code-block:: yaml

  extensions:
  - code: console.log(Object.keys(context));
    id: test
    path: /v2.0/.*

Javascirpt Code block
---------------------

In the gohan extension code, you need to register context using
gohan_register_handler function.
gohan_register_handler talkes event_type (string)_ and handler (function(context)).

.. code-block:: javascript

  gohan_register_handler("pre_show", function(context){
    context.resp = "pre_show event"
  });

  gohan_register_handler("pre_update", function(context){
    context.resp = "pre_update event "
  });

context has following items

  context.schema : schema information
  context.path : url path
  context.role : user role
  context.auth : auth_context information
  context.http_request : Go HTTP request object
  context.http_response : Go HTTP response writer object


Build in exception types
------------------------

In an effort to simplify writing extensions for validation Gohan supports
throwing some exceptions and handles them internally.
Gohan provides the following exception types.

- BaseException(msg)

The base type of exception, should never be raised as itself, only extended.

- CustomException(msg, code)

A BaseException with an additional code property. When thrown will result in
an http response with the provided code and message being written.

One can extend the CustomException. An example follows.

.. code-block:: javascript

  function ValidationException(msg) {
    CustomException.call(this, msg, 400);
    this.name = "ValidationException";
  }
  ValidationException.prototype = Object.create(CustomException.prototype);


.. _`gohan built in functions`:

Build in javascript functions
-----------------------------

Gohan extension supports some build-in functions.

- console.log(string)

Logging output

- gohan_http(method, url, headers, data, options)

fetch data from url
method : GET | POST | PUT | DELETE
url : destination url
headers : additional headers (eg. AUTH_TOKEN)
data : post or put data
options : dictionary of options for http client
opaque : boolean - whether to parse URL or to treat it as raw

- gohan_db_list(transaction, schema_id, filter_object)

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

Dynamically load modules

.. _event:

Event
----------------

- pre_list

  list event before db operation

  context.response contains response data.
  You can also update response here

  Note you can skip db operation if you set context response in here

- pre_list_in_transaction

  same as pre_list but executed in the db transaction
  context.transaction contains transaction object for db operation

  id : request id
  context.response contains response data.
  You can also update response here

- post_list_in_transaction

  same as post_list but executed in the db transaction
  context.transaction contains transaction object for db operation

  id : request id
  context.response contains response data.
  You can also update response here

- post_list

  list event after db operation.

  context.response contains response data.
  You can also update response here

- pre_show

  show event before db access

  id : request id
  context.response contains response data.
  You can also update response here

  Note you can skip db operation if you set context response in here

- pre_show_in_transaction

  same as pre_show but executed in the db transaction
  context.transaction contains transaction object for db operation

  id : request id
  context.response contains response data.
  You can also update response here

- post_show_in_transaction

  same as post_show but executed in the db transaction
  context.transaction contains transaction object for db operation

  id : request id
  context.response contains response data.
  You can also update response here

- post_show

  show event after db operation

  id : request id
  context.response contains response data.
  You can also update response here

- pre_create

  executed before creation
  Mainly used for validation purpose

  context.resource contains user input data

  Note you can skip db operation if you set context response in here

- pre_create_in_transaction

  same as pre_create but executed in the db transaction
  context.transaction contains transaction object for db operation

- post_create_in_transaction

  after creation in transaction

- post_create

  after create
  context.response contains response data.
  context.transaction contains transaction object for db operation

- pre_update

  executed before update
  Mainly used for validation purpose

  context.resource contains user input data

  Note you can skip db operation if you set context response in here

- pre_update_in_transaction

  same as pre_update but executed in the db transaction
  context.transaction contains transaction object for db operation

- post_update_in_transaction

  after creation in transaction

- post_update

  after update
  context.response contains response data.
  context.transaction contains transaction object for db operation

- pre_delete

  executed before delete
  Mainly used for validation purpose

  context.id contains resource id we are trying to delete
  context.transaction contains transaction object for db operation

- pre_delete_in_transaction

  same as pre_delete but executed in the db transaction
  context.transaction contains transaction object for db operation

- post_delete_in_transaction

  after creation in transaction

- post_delete

  after delete

- pre_state_update_in_transaction

  executed before a state update triggerred by a backend event

  context.resource contains the resource associated with the update,
  context.state contains the state changes,
  context.config_version contains the current config version

- post_state_update_in_transaction

  as above, but after the state update

- pre_monitoring_update_in_transaction

  executed before a monitoring update triggerred by a backend event

  context.resource contains the resource associated with the update,
  context.monitoring contains the new monitoring information

- post_monitoring_update_in_transaction

  as above, but after the monitoring update

- notification

  executed when you receive amqp/snmp/cron notification

Testing javascript extensions
-----------------------------

You can test extensions using a testing tool bundled with Gohan through new
command ``testextensions`` (or simply ``te``). Build and install Gohan, then
run ``gohan testextensions <paths to files/directories to test>``. The
framework will walk through files and recursively through directories, matching
files named ``test_.*.js`` and running tests.

Test file contents
^^^^^^^^^^^^^^^^^^
Each test file must specify schema and path for preloading extensions:

* var SCHEMA - path to the schema that stores extensions to be tested
* var PATH - path for preloading extensions

Additionally each file can specify:

* one setUp() function that will be called before each test
* one tearDown() function that will be called after each test
* multiple test_<name>() functions that will be called by the framework
* multiple helper functions and variables, with names not starting with prefix
  ``test_``

Framework API
^^^^^^^^^^^^^
Test framework provides all built in function mentioned in subsection
describing `gohan built in functions`_.

To avoid making HTTP requests during tests, ``gohan_http`` function is a mock.
You can pass values that will be returned for given arguments during subsequent
calls by calling ``gohan_http.Expect(argument, ...).Return(value)``. One call to
``gohan_http.Expect(arguments, ...).Return(value)`` provides one response of
``gohan_http`` (FIFO queue). If no return value, or wrong arguments are provided
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

* ``MockTransaction() : <mock transaction>`` - return a mock transaction that
  can be used with built-in Gohan methods. Each test is run using a separate
  database that is deleted after ``tearDown()``, so there is no need to
  clean up the database between tests. Multiple calls to ``MockTransaction()``
  within a single ``setUp()``, test, ``tearDown()`` routine when no call to
  ``CommitMockTransaction()`` has been made will yield the same transaction.

* ``CommitMockTransaction()`` - commit and close the last nonclosed
  transaction. After this call any calls to ``MockTransaction()`` return
  a new transaction.

* ``MockPolicy() : <mock policy>`` - return a mock policy that
  can be used with built-in Gohan methods.

* ``MockAuthorization() : <mock authorization>`` - return a mock authorization that
  can be used with built-in Gohan methods.

Example
^^^^^^^
A sample test may look like this:

.. code-block:: javascript

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

Javascript Backend
-------------------------

Currenly, gohan is using Otto. Otto is a pure golang implementation
for javascript.
Gohan also have experimental support for v8. v8 runs js code 100-1000 times faster than Otto.

TODOs
- no build-in are implemented yet

In order to make v8 version of Gohan, you need v8worker https://github.com/ry/v8worker installed in your env. (see more instruction on the repository).

In order to enable v8 support on extension. then set ENABLE_V8=true

.. code-block:: shell

  ENABLE_V8=true make


Go based extension
-------------------------

You can extend gohan extension by native go.
You can use "go" for code_type and specify your callback id in code.
Also, you can register go struct & call it from javascript.

.. code-block:: yaml

  extensions:
  - code: exampleapp_callback
    code_type: go
    id: example
    path: .*
  - code: exampleapp_callback
    code_type: go
    id: example
    path: .*
  - code: |
      gohan_register_handler("pre_list", function (context){
        var exampleModule = require("exampleapp");
        exampleModule.HelloWorld("example app!",
          {"hobby": "sleeping"});
      });
    id: example_js
    path: .*


.. code-block:: go

  //Register go callback
  extension.RegisterGoCallback("exampleapp_callback",
  	func(event string, context map[string]interface{}) error {
  		fmt.Printf("callback on %s : %v", event, context)
  		return nil
  })

  exampleModule := &ExampleModule{}
  //Register go based module for javascript
  extension.RegisterModule("exampleapp", exampleModule)


We have exampleapp with comments in exampleapp directory.
You can also, import github.com/cloudwan/server module and
have your own RunServer method to have whole custom route written in go.


Donburi
-------------------------

Note: This function is experimental. Any APIs are subject to change.

Gohan support Donburi which is a yaml based DSL to support extension.
Donburi is heavyly inspired by Ansible yaml script.
The goal of Donburi is pain-less extension using YAML.

please take a look detail on github site
https://github.com/cloudwan/donburi