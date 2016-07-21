==============
Extension
==============

You can add additional logic using Gohan Extension.
Extensions has properties:

- id identity of the code
- code contents of code
- code_type javascript, go and gohan script (DSL) are supported
- url placement of code. currenty, file://, http:// and https:// schemes are supported
- path resource path to execute code

Example Code

.. code-block:: yaml

  extensions:
  - code: console.log(Object.keys(context));
    id: test
    path: /v2.0/.*

Javascript Code block
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
  context.params : query params
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

You can test extensions using a testing tool bundled with Gohan with the command
``test_extensions`` (or ``test_ex`` for short). Build and install Gohan, then
run ``gohan test_extensions <paths to files/directories to test>``. The
framework will walk through files and recursively through directories, running
tests in files named ``test_*.js``.

By default, the framework doesn't show logs and results for passing tests, so
you won't see any output if all the tests pass. If you pass a
``-v``/``--verbose`` flag, it will show these messages, and an additional ``All
tests have passed.`` message if all the tests pass.

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


Gohan script
-------------------------

Note: This function is experimental. Any APIs are subject to change.

Gohan script is an Ansible-like MACRO language
for extending your Go code with MACRO functionality.


Example
~~~~~~~

.. code-block:: yaml

extensions:
- id: order
  path: /v1.0/store/orders
  code: |
    tasks:
    - when: event_type == "post_create_in_transaction"
      blocks:
      #  Try debugger
      # - debugger:
      - db_get:
          tx: $transaction
          schema_id: pet
          id: $resource.pet_id
        register: pet
      - when: pet.status != "available"
        blocks:
        - vars:
            exception:
              name: CustomException
              code: 400
              message: "Selected pet isn't available"
        else:
        - db_update:
            tx: $transaction
            schema_id: pet
            data:
                id: $resource.pet_id
                status: "pending"
    - when: event_type == "post_update_in_transaction"
      blocks:
      - when: resource.status == "approved"
        blocks:
        - db_update:
            tx: $transaction
            schema_id: pet
            data:
              id: $resource.pet_id
              status: "sold"


Standalone Example
~~~~~~~~~~~~~~~~~~~

.. code-block:: yaml

    vars:
      world: "USA"
      foods:
      - apple
      - orange
      - banana
    tasks:
    - debug: var=$foods[2]
    - debug: msg="Hello \" {{ world }}"
    - blocks:
        - debug: msg="I like {{ item }}"
          with_items:
        - apple
        - orange
        - banana
        - debug: msg="I like {{ item }}"
          with_items: $foods
        # Unlike Ansible, We treat a value as identifier if a string value starts with "$"
        # otherwise it is a value
        - debug: msg="{{ item.key }} likes {{ item.value }}"
          with_dict:
            Alice: apple
            Bob: orange
        - debug: msg="This shouldn't be called"
          when: 1 == 0
          else:
          - debug: msg="This should be called"
        - fail: "failed"
          when: 1 == 1
          rescue:
          - debug: msg="rescued {{ error }}"
          always:
          - debug: msg="Drink beer!"
    - debug: msg="test {{ 1 == 1 }}"
    - include: lib.yaml
        vars:
        local_vars: hello from imported code


see more detail on extension/gohanscript/test/core_test.yaml

CLI
~~~~~~~

You can run Gohan script code using this

.. code-block:: shell

    gohan run ../examples/sample1.yaml

Tasks
~~~~~~~~~

You can run list of tasks.

.. code-block:: yaml

  tasks:
  - debug: msg="Hello World"
  - debug: msg="This is gohan script"

save this file to hello_world.yaml.

.. code-block:: shell

    $ gohan run hello_world.yaml
    15:17:07.029 ▶ DEBUG  hello_world.yaml:1: Hello World
    15:17:07.029 ▶ DEBUG  hello_world.yaml:2: This is gohan script

Variables
~~~~~~~~~

You can define variables using "vars".

.. code-block:: yaml

  tasks:
  - vars:
      place: "Earth"
      person:
        name: "John"
        age: "30"
  - debug: msg="Hello {{place}}"
  - debug: var=$place
  - debug: msg="Hello {{person.name}} "
  - debug: var=$person.name
  - debug: # show everything

Any string including "{{" get considered as django template. so
you can use variables in their. if string start with "$", it get considered as
a variable identifier.
(We are using pongo2 which supports subset of django template..)

.. code-block:: shell

    $ gohan run variable.yaml
    15:21:43.090 ▶ DEBUG  variable.yaml:6 Hello Earth
    15:21:43.091 ▶ DEBUG  variable.yaml:7 Earth
    15:21:43.091 ▶ DEBUG  variable.yaml:8 Hello John
    15:21:43.091 ▶ DEBUG  variable.yaml:9 John
    15:21:43.091 ▶ DEBUG  variable.yaml:10 Dump vars
    15:21:43.091 ▶ DEBUG      person: map[name:John age:30]
    15:21:43.091 ▶ DEBUG      __file__: variable.yaml
    15:21:43.091 ▶ DEBUG      __dir__: .
    15:21:43.091 ▶ DEBUG      place: Earth

Loops
~~~~~~~~~~~~~~

You can loop over the list item.

.. code-block:: yaml

    vars:
        foods:
        - apple
        - orange
        - banana
    tasks:
    - debug: msg="{{ item }}"
      with_items:
      - apple
      - orange
      - banana
    - debug: msg="{{ item }}"
      with_items: $foods

.. code-block:: shell

    $ gohan run with_items.yaml
    15:28:47.736 ▶ DEBUG  with_items.yaml:6 apple
    15:28:47.736 ▶ DEBUG  with_items.yaml:6 orange
    15:28:47.736 ▶ DEBUG  with_items.yaml:6 banana
    15:28:47.736 ▶ DEBUG  with_items.yaml:11 apple
    15:28:47.736 ▶ DEBUG  with_items.yaml:11 orange
    15:28:47.736 ▶ DEBUG  with_items.yaml:11 banana

You can also loop over a dict.

.. code-block:: yaml

    vars:
    person:
        name: "John"
        age: "30"
    tasks:
    - debug: msg="{{ item.key }} {{ item.value }}"
      with_dict:
        name: "John"
        age: "30"
    - debug: msg="{{ item.key }} {{ item.value }}"
      with_dict: $person

.. code-block:: shell

    $ gohan run with_items.yaml
    15:32:42.513 ▶ DEBUG  with_items.yaml:5 name John
    15:32:42.513 ▶ DEBUG  with_items.yaml:5 age 30
    15:32:42.513 ▶ DEBUG  with_items.yaml:9 name John
    15:32:42.513 ▶ DEBUG  with_items.yaml:9 age 30

you can specify loop variable name by specifying loop_var:

.. code-block:: yaml

    tasks:
    - vars:
        result: ""
        persons:
        - name: Alice
          hobbies:
          - mailing
          - reading
        - name: Bob
          hobbies:
          - mailing
          - running
    - blocks:
        - vars:
            result: "{{result}}{{item}}"
          with_items: $person.hobbies
      with_items: $persons
      loop_var: person


Conditional
~~~~~~~~~~~~~~

You can use "when" for conditional.
You can use "else" blocks with "when".

.. code-block:: yaml
    vars:
      number: 1
    tasks:
    - debug: msg="Should be called"
      when: number == 1
    - debug: msg="Should not be called"
      when: number == 0
      else:
      - debug: msg="Should be called"

.. code-block:: shell

    $ gohan run when.yaml
    15:35:55.358 ▶ DEBUG  when.yaml:3 Should be called

Retry
~~~~~~~~~~~~~

You can retry task.

- retry: how many times you will retry a task
- delay: how many seconds you will wait on next retry

.. code-block:: yaml

    tasks:
    - fail: msg="Failed"
      retry: 3
      delay: 3

.. code-block:: shell

    $ gohan run retry.yaml
    15:43:35.720 ▶ WARNING  error: tasks[0]: Failed
    15:43:35.720 ▶ WARNING  error: tasks[0]: Failed
    Failed

Blocks
~~~~~~~~~~~~~

You can group a set of tasks using blocks.
blocks also supports loops, conditional and retries.

.. code-block:: yaml

    tasks:
    - blocks:
      - debug: msg="hello"
      - debug: msg="from in block"

.. code-block:: shell

    $ gohan run blocks.yaml
    15:48:30.231 ▶ DEBUG  blocks.yaml:2 hello
    15:48:30.231 ▶ DEBUG  blocks.yaml:3 from in block

Register
~~~~~~~~~~~~~

You can change variable value using "register".

.. code-block:: yaml

    tasks:
    - http_get: url=https://status.github.com/api/status.json
      register: result
    - debug: msg="{{result.contents.status}}"

.. code-block:: shell

    $ gohan run register.yaml
    15:51:11.005 ▶ DEBUG  [register.yaml line:3 column:2] good

Concurrency
~~~~~~~~~~~~~

We support concurrent execution over a loop.

- worker: specify number of max workers

.. code-block:: yaml

    tasks:
    - blocks:
      - http_get: url="https://status.github.com/{{ item }}"
        register: result
      - debug: var=$result.raw_body
    worker: 3
    with_items:
    - /api/status.json
    - /api.json
    - /api/last-message.json

.. code-block:: shell

    $ gohan run worker.yaml
    15:58:49.151 ▶ DEBUG  worker.yaml:4 {"status_url":"https://status.github.com/api/status.json","messages_url":"https://status.github.com/api/messages.json","last_message_url":"https://status.github.com/api/last-message.json","daily_summary":"https://status.github.com/api/daily-summary.json"}
    15:58:49.156 ▶ DEBUG  worker.yaml:4 {"status":"good","body":"Everything operating normally.","created_on":"2016-03-03T22:03:59Z"}
    15:58:49.156 ▶ DEBUG  worker.yaml:4 {"status":"good","last_updated":"2016-03-08T23:58:27Z"}

You can also execute tasks in background.

.. code-block:: yaml

    tasks:
    - background:
      - sleep: 1000
      - debug: msg="called 2"
    - debug: msg="called 1"
    - sleep: 2000
    - debug: msg="called 3"

.. code-block:: shell

    $ gohan run background.yaml
    16:02:55.034 ▶ DEBUG  background.yaml:6 called 1
    16:02:56.038 ▶ DEBUG  background.yaml:4 called 2
    16:02:57.038 ▶ DEBUG  background.yaml:8 called 3


Define function
~~~~~~~~~~~~~~~

You can define function using "define" task.

- name: name of function
- args: arguments
- body: body of code

.. code-block:: yaml

    tasks:
    - define:
        name: fib
        args:
          x: int
        body:
        - when: x < 2
          return: x
        - sub_int: a=$x b=1
          register: $x
        - fib:
            x: $x
          register: a
        - sub_int: a=$x b=1
          register: x
        - fib:
            x: $x
          register: b
        - add_int: a=$a b=$b
          register: result
        - return: result
    - fib: x=10
      register: result2
    - debug: msg="result = {{result2}}"

you can use return task in function block.

.. code-block:: shell

    $ gohan run fib.yaml
    16:07:39.964 ▶ DEBUG  fib.yaml:23 result = 55

Include
~~~~~~~

You can include gohan script

.. code-block:: yaml

    tasks:
    - include: lib.yaml

.. code-block:: shell

    $ gohan run include.yaml
    16:11:20.569 ▶ DEBUG  lib.yaml:0 imported

Debugger mode
~~~~~~~~~~~~~~

You can set breakpoint using "debugger"

.. code-block:: yaml

    vars:
    world: "USA"
    foods:
    - apple
    - orange
    - banana
    tasks:
    - debugger:
    - debug: msg="Hello {{ world }}"

.. code-block:: shell

    10:50:55.052 gohanscript INFO  Debugger port: telnet localhost 40000


Supported command in debugger mode:

- s: step task
- n: next task
- r: return
- c: continue
- p: print current context
- p code: execute miniGo
- l: show current task

You will get separate port per go routine.

Command line argument
~~~~~~~~~~~~~~~~~~~~~~

additional arguments will be stored in variable.
If the value doesn't contain "=", it will be pushed to args.
If the value contains "=", it get splitted for key and value and stored in flags.

.. code-block:: yaml

    tasks:
    - debug: msg="{{ flags.greeting }} {{ args.0 }}"

.. code-block:: shell

$ gohan run args.yaml world greeting=hello
hello world


Run test
~~~~~~~~~~~~~~

You can run gohan build-in test. gohan test code find test gohan code
in the specified directory.

.. code-block:: yaml

    gohan test


Run Gohan script from Go
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: go

	vm := gohan.NewVM()
	_, err := vm.RunFile("test/spec.yaml")
	if err != nil {
		t.Error(err)
	}

Add new task using Go
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

You can auto generate adapter functions using ./extension/gohanscript/tools/gen.go.

.. code-block:: shell

  go run ./extension/gohanscript/tools/gen.go genlib -t extension/gohanscript/templates/lib.tmpl -p github.com/cloudwan/gohan/extension/gohanscript/lib -e autogen -ep extension/gohanscript/autogen


More examples and supported functions
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

please take a look

- extension/gohanscript/lib/tests
- extension/gohanscript/tests
