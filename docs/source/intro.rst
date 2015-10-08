==============
Introduction
==============


Gohan is an API server for XaaS Services.
Using gohan webui front-end, you can define new XaaS API on the fly, and
You can use the API without rebooting or redeploying service.
Resource status will be synced with agents, who will realize XaaS service, using
etcd.


Gohan supports following functionalities

- Define resources by JSON Schema + Resource configuration
- REST API Server with RDBMS DB backend based on the schema
- etcd sync backend based on the schema
- Policy configuration
- Extension by Javascript
- Automatic API Generation
