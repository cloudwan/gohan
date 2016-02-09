#!/bin/bash

# Run Unit Test for mysql

if [[ $MYSQL_TEST == "true" ]]; then
  # set MYSQL_TEST true if you want to run test against Mysql.
  # you need running mysql on local without root password for testing.
  mysql -uroot -e "drop database if exists gohan_test; create database gohan_test;"
fi

DATA_DIR=`mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir'`
etcd -data-dir $DATA_DIR &
ETCD_PID=$!

if [[ $ENABLE_V8 == "true" ]]; then
	TAGS="-tags v8"
fi

gocov test $TAGS ./...  > coverage.json

kill $ETCD_PID
exit $result
