# Gohan : API gateway server

[![Join the chat at https://gitter.im/cloudwan/gohan](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/cloudwan/gohan?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Coverage Status](https://coveralls.io/repos/github/cloudwan/gohan/badge.svg?branch=master)](https://coveralls.io/github/cloudwan/gohan?branch=master)
[![Circle CI](https://circleci.com/gh/cloudwan/gohan.svg?&style=shield)](https://circleci.com/gh/cloudwan/gohan)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudwan/gohan)](https://goreportcard.com/report/github.com/cloudwan/gohan)

Gohan is an API Gateway Server written by Go that makes it easy for developers to create and maintain REST-style API.
An API Gateway Server has a benefit consolidating various API operations such as authentication, authorization based on policies, logging and input validation on a single place on top of so-called microservice architecture.

Gohan also makes transactional operations, involving multiple micro services, easy to operate. An approach Gohan using is quite simple. Persistent REST API Resources in the RDBMS using transaction, then sync resource status with backend microservices using etcd or MySQL binlog API.  Using well-proven RDBMs transaction, we can protect correctness of resources. A strategy let backend microservices sync with correct resource data in the RDBMS makes entire system fault-torrent from various RPC failures. Note that Developers should design _ backend microservices idempotent manner, to handle the cases the same RPC invoked multiple times.

see [Pet Store Example] (./etc/example_schema.yaml)

[![GoDoc](https://godoc.org/github.com/cloudwan/gohan?status.svg)](https://godoc.org/github.com/cloudwan/gohan)

Documentation

- [Introduction](docs/introduction.md)
- [Installation](docs/installation.md)
- [Development](docs/development.md)
- [Schema](docs/schema.md)
- [Namespace](docs/namespace.md)
- [Database](docs/database.md)
- [Policy](docs/policy.md)
- [Extension](docs/extension.md)
- [JavaScript Extension](docs/js_extension.md)
- [Gohan Script Extension](docs/gohan_extension.md)
- [CLI](docs/cli.md)