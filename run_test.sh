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

# Run test coverage on each subdirectories and merge the coverage profile.
echo "mode: count" > profile.cov

# Standard go tooling behavior is to ignore dirs with leading underscors
for dir in $(find . -maxdepth 10 -not -path './.git*' -not -path '*/_*' -type d);
do
result=0
if ls $dir/*.go &> /dev/null; then
    go test $TAGS -covermode=count -coverprofile=$dir/profile.tmp $dir --ginkgo.randomizeAllSpecs --ginkgo.failOnPending --ginkgo.trace --ginkgo.progress
    result=$?
    if [ -f $dir/profile.tmp ]
    then
        cat $dir/profile.tmp | tail -n +2 >> profile.cov
        rm $dir/profile.tmp
    fi
    if [ $result -ne 0 ]; then
        break
    fi
fi
done

if [ $result -eq 0 ]; then
    go tool cover -func profile.cov
fi

kill $ETCD_PID
exit $result
