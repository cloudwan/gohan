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

echo "mode: count" > profile.cov

# Standard go tooling behavior is to ignore dirs with leading underscors
for dir in $(find . -maxdepth 10 -not -path './.git*' -not -path '*/_*' -not -path './vendor/*' -type d);
do
if ls $dir/*.go &> /dev/null; then
    go test -race -covermode=atomic -coverprofile=$dir/profile.tmp $dir
    result=$?
    if [ -f $dir/profile.tmp ]
    then
        cat $dir/profile.tmp | tail -n +2 >> profile.cov
        rm $dir/profile.tmp
    fi
    if [ $result -ne 0 ]; then
        exit $result
    fi
fi
done

go tool cover -func profile.cov
kill $ETCD_PID
