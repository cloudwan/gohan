.. _section-namespace:

==============
Namespace
==============

You can logically group schemas by creating a single endpoint using this resource.
Namespace has the following properties:
- id : unique identifier of the namespace
- name : name of the namespace
- description : description of the namespace
- prefix : prefix for the namespace
- parent : parent namespace
- version : version of the namspace
- metadata : arbitrary metadata

Example namespace

.. code-block:: yaml

  namespaces:
  - description: Neutron API
    id: neutron
    name: Neutron
    prefix: neutron
  - description: Version 2.0 of Neutron API
    id: neutron_v2
    metadata:
      key: value
    name: Neutron 2.0
    parent: neutron
    prefix: v2.0
    version: "2.0"

The full prefix of a namespace is a concatenation of prefixes of all its
ancestors. Any schema specifying a namespace will have such prefix prepended
to URLs specific to that schema.

Example: If a schema specifies that its namespace is neutron_v2 from the previous
example, a prefix of /neutron/v2.0/ will be prepended to its prefix.

For a top-level namespace ("neutron" from the example above) a root URL is mapped
(../neutron/) that lists all child namespaces. For namespaces that have a parent
specified ("neutron_v2" from the example above) an access URL is mapped using
prefixes from ancestors and its own prefix (../neutron/v2.0) that lists all
schemas belonging to the namespace.
