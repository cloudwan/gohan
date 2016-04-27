=================
Sync with backend
=================

Gohan stores an event log recording create, update and delete database operations.
This is done in the database transaction, so we can assume that
this event log data is consistent with the resource data.

The event log data contains the following information. (see schema in gohan.json)

- id -- an unique ID of the event
- type -- the type of the event
- path -- the path of the resource related to the event
- timestamp -- the time at which the event occurred
- version -- the version of the resource after this event occurred
- body -- the contents of the resource after this event occurred

The Gohan server will select one master node using the etcd backend CAS API.
Only the master node will then poll the event log table, pushing to the backend.

We may support mysql binlog api for better performance in future.

.. _subsection-state-update:

State updates
-------------

Gohan will keep track of the state version of any resource associated to
a schema with metadata containing the key ``state_versioning`` set to
``true``. In such a case Gohan will remember the config version and
the state version of the resource. During creation the config version of
such a resource will be set to ``1``. On delete and update the version is
bumped by one. The state version is ``0`` originally and later read from
the sync backend and updated asynchronously. Both the versions are returned
in GET requests, together with additional information about the state.

For example say a simplistic resource with the following schema is created:

.. code-block:: yaml

    - description: Just a named object
      id: named_object
      parent: ""
      metadata:
        state_versioning: true
      singular: named_object
      plural: named_object
      prefix: /v1.0
      schema:
        properties:
          name:
            default: ""
            permission:
            - create
            - update
            title: Name
            type: string
            unique: false
          id:
            permission:
            - create
            title: ID
            type: string
            format: uuid
        properties_order:
        - id
        - name
        required:
        - name
        type: object
      title: Named Object

and the ``name`` is set to ``Alice``. Then Gohan, through the standard event sync,
writes the following JSON object to the backend under the key
``config/v1.0/named_object/someGeneratedUuid``:

.. code-block:: javascript

    {
      "body": {
        "id": "someGeneratedUuid",
        "name": "Alice"
      },
      "version": 1
    }

A worker program might now read this information, create a corresponding
southbound resource and write the following to the backend under the key
``state/v1.0/named_object/someGeneratedUuid``:

.. code-block:: javascript

    {
      "version": 1,
      "error": "",
      "state": "Alice exists"
    }

Gohan will read this information and update the database accordingly.

Any state updates made when the state version already equals the config version
will be ignored.

Monitoring updates
------------------

After a resource has been created in the southbound, one might monitor its
status. This is done using a very similar approach to status updates.
Monitoring updates the ``monitoring`` field in the database, which is returned
together with the rest of the state.

A continuation of the above example follows. After the resource has been created
in the southbound a worker program might monitor its status and then write
the result of this monitoring under the key
``monitoring/v1.0/named_object/someGeneratedUuid`` as the following JSON:

.. code-block:: javascript

    {
      "version": 1,
      "monitoring": "Alice is well"
    }

Gohan will read this information and update the database accordingly.

Any monitoring updates made when the state version does not yet equal
the config version or the version in the JSON data doesn't match with
the config version will be ignored.
