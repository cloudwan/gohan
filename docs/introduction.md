# Introduction

Gohan is an API Gateway Server written by Go that makes it easy for developers to create and maintain REST-style API.
An API Gateway Server has a benefit consolidating various API operations such as authentication, authorization based on policies, logging and input validation on a single place on top of so-called microservice architecture.

Gohan also makes transactional operations, involving multiple micro services, easy to operate. An approach Gohan using is quite simple. Persistent REST API Resources in the RDBMS using transaction, then sync resource status with backend microservices using etcd or MySQL binlog API.  Using well-proven RDBMs transaction, we can protect correctness of resources. A strategy let backend microservices sync with correct resource data in the RDBMS makes entire system fault-torrent from various RPC failures. Note that Developers should design _ backend microservices idempotent manner, to handle the cases the same RPC invoked multiple times.