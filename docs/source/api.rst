==============
API
==============

In this section, we show how we generate REST API based on schema.

"$plural", "$singular", "$prefix" and "$id" are read directly from schema,
"$namespace_prefix" is computed using namespace information and might be empty
if schema has no namespace specified.

Note: An extension computes actual access URL for each resource and substitutes
prefix property with it during schema listing calls. User can list resources
using this URL and access a single instance of resource by prepending "/$id"
suffix.

List
--------------

List REST API

List supports pagination by optional GET query parameters ``sort_key`` and ``sort_order``.

================  ==========  =============  ================  ====================================================
Query Parameter   Style       Type           Default           Description
================  ==========  =============  ================  ====================================================
sort_key          query       xsd:string     id                Sort key for results
sort_order        query       xsd:string     asc               Sort order - allowed values are ``asc`` or ``desc``
limit             query       xsd:int        0                 Specifies maximum number of results.
                                                               Unlimited for non-positive values
offset            query       xsd:int        0                 Specifies number of results to be skipped
<parent>_id       query       xsd:string     N/A               When resources which have a parent are listed,
                                                               <parent>_id can be specified to show only parent's children.
<property_id>     query       xsd:string     N/A               filter result by property (exact match). You can use multiple filters.
================  ==========  =============  ================  ====================================================

When specified query parameters are invalid, server will return HTTP Status Code ``400`` (Bad Request)
with error message explaining the problem.

To make navigation easier, each ``List`` response contains additional header ``X-Total-Count``
indicating number of all elements without applying ``limit`` or ``offset``.

Example:
GET http://$GOHAN/[$namespace_prefix/]$prefix/$plural?sort_key=name&limit=2

Response will be

HTTP Status Code: 200

.. code-block:: javascript

  {
    "$plural": [
       {
         "attr1": XX,
         "attr2": XX
       }
    ]
  }

Child resources access
------------------------

Child resources can be accessed in two ways:

Full path
  In order to access a child resource in that way, we need to know all it parents.

  e.g. POST http://$GOHAN/[$namespace_prefix/]$prefix/[$ancestor_plural/$ancestor_id/]$plural

Short path
  If we don't know resource full path, it can be accessed with $parent_id.

  e.g. POST http://$GOHAN/[$namespace_prefix/]$prefix/$plural?$parent_id=<parent_id>

GET
--------------

Show REST API

GET http://$GOHAN/[$namespace_prefix/]$prefix/$plural/$id

Response will be

HTTP Status Code: 200

.. code-block:: javascript

  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }


CREATE
--------------------------------------

CREATE Resource REST API

POST http://$GOHAN/[$namespace_prefix/]$prefix/$plural/

Input

Note that input json can only contain
if you set "create" permission for this

HTTP Status Code: 202

.. code-block:: javascript

  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }


Response will be

.. code-block:: javascript

  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }


Update
--------------------------------------

Update Resource REST API

PUT http://$GOHAN/[$namespace_prefix/]$prefix/$plural/$id

Input

Note that input json can only contain
if you set "update" permission for this

.. code-block:: javascript

  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }


Response will be

HTTP Status Code: 200

.. code-block:: javascript

  {
    "$singular": {
      "attr1": XX,
      "attr2": XX
    }
  }


DELETE
--------------------------------------

Delete Resource REST API

HTTP Status Code: 204

DELETE http://$GOHAN/[$namespace_prefix/]$prefix/$plural/$id


Custom Actions
--------------------------------------

Run custom action on a resource

POST http://$GOHAN/[$namespace_prefix/]$prefix/$plural/$id/$action_path

Input

Input json can only contain parameters defined in input schema definition
It requires "$action" allow policy

.. code-block:: javascript

  {
    "parameter1": XX,
    "parameter2": XX
  }


Response will be

HTTP Status Code: 200

.. code-block:: javascript

  {
    "output1": XX,
    "output2": XX
  }
