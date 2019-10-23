#!/bin/bash
set -e

# Run Unit Test for mysql

if [ $MYSQL_TEST == "true" ] && [ $CIRCLECI != "true" ]; then
  # set MYSQL_TEST true if you want to run test against Mysql.
  # you need running mysql on local without root password for testing.
  mysql -uroot -e "drop database if exists gohan_test; create database gohan_test;"
fi

DATA_DIR=`mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir'`
etcd -data-dir $DATA_DIR --listen-peer-urls http://127.0.0.1:2380 --listen-client-urls http://127.0.0.1:2379 --advertise-client-urls http://127.0.0.1:2379 &
ETCD_PID=$!
trap "kill $ETCD_PID" EXIT

go test -race -covermode=atomic -coverprofile=profile.cov -coverpkg=$(go list ./... | grep -vE 'integration_tests|error_test' | tr '\n' ',') $(go list ./... | grep -v 'goplugin')
go test -race -covermode=atomic -coverprofile=profile.tmp $(go list ./... | grep 'goplugin')
tail -n +2 profile.tmp >> profile.cov
rm profile.tmp
go tool cover -func profile.cov
