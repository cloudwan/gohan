# Gohan : YAML-based REST API Service Definition Language #

[![Join the chat at https://gitter.im/cloudwan/gohan](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/cloudwan/gohan?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Coverage Status](https://coveralls.io/repos/github/cloudwan/gohan/badge.svg?branch=master)](https://coveralls.io/github/cloudwan/gohan?branch=master)
[![wercker status](https://app.wercker.com/status/cab137b4bfdd05c97cfface7ac12c039/ "wercker status")](https://app.wercker.com/project/bykey/cab137b4bfdd05c97cfface7ac12c039)
[![Circle CI](https://circleci.com/gh/cloudwan/gohan.svg?&style=shield)](https://circleci.com/gh/cloudwan/gohan)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudwan/gohan)](https://goreportcard.com/report/github.com/cloudwan/gohan)

- API Definition Generation (including Swagger)
- DB Table Generation & OR Mapping
- Support Custom Logic using Gohan Script (Javascript, and Go)
- Extensible Role-Based Access Control
- etcd integration

see [Pet Store Example] (./etc/example_schema.yaml)

# How to install

## Download Binary

* Download [Gohan Release](https://github.com/cloudwan/ansible-gohan/releases)
* Start server:

```
./gohan server --config-file etc/gohan.yaml
```

# Packages

## Ubuntu 14.04 Trusty 64bits server
```
wget -qO - https://deb.packager.io/key | sudo apt-key add -
echo "deb https://deb.packager.io/gh/cloudwan/gohan trusty master" | sudo tee /etc/apt/sources.list.d/gohan.list
sudo apt-get update
sudo apt-get install gohan
```
## CentOS / RHEL 6 64 bits server

```
sudo rpm --import https://rpm.packager.io/key
echo "[gohan]
name=Repository for cloudwan/gohan application.
baseurl=https://rpm.packager.io/gh/cloudwan/gohan/centos6/master
enabled=1" | sudo tee /etc/yum.repos.d/gohan.repo
sudo yum install gohan
```

## Debian 7 Wheezy 64bits server

```
wget -qO - https://deb.packager.io/key | sudo apt-key add -
echo "deb https://deb.packager.io/gh/cloudwan/gohan wheezy master" | sudo tee /etc/apt/sources.list.d/gohan.list
sudo apt-get update
sudo apt-get install gohan
```

## Build

* Install GO >= 1.6
* go get github.com/cloudwan/gohan

## Heroku

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/cloudwan/gohan.git)

# Doc
[Document](http://gohan.cloudwan.io/gohan/)

Quick link

- [Schema](./docs/source/schema.rst)
- [Policy](./docs/source/policy.rst)
- [Extension](./docs/source/extension.rst)
- [Configuraion](./docs/source/config.rst)
- [API](./docs/source/api.rst)
- [DB](./docs/source/database.rst)
- [CLI](./docs/source/commands.rst)
- [Integration](./docs/source/sync.rst)

[![GoDoc](https://godoc.org/github.com/cloudwan/gohan?status.svg)](https://godoc.org/github.com/cloudwan/gohan)

# WebUI
```
http://localhost:9091/ (or https://$APPNAME.herokuapp.com/ )
```

* Admin User

  * ID: admin
  * Password: gohan
  * Tenant: demo

* Member User

  * ID: member
  * Password: gohan
  * Tenant: demo

# Examples

[ example configuraions ](./examples)

# Development Guide

* Install Go >= 1.6
* Install development tools

```
make deps
```

* make & make install

```
make
make install
```

* Send a pull request to Github

# How to contribute

* Sign our CLA and send scan for info@cloudwan.io

    * [Individual CLA](./docs/cla.txt)
    * [Company CLA](./docs/ccla.txt)

* Create an issue in github
* Send PR for github

We recommend to rebase multiple commits

# License
Apache2