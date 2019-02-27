#!/bin/bash
set -e

GOHANPATH="$(realpath "$(dirname "$(realpath -P "$0")")/..")"

# sub-bash
(
    # run ETCD
    DATA_DIR=`mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir'`
    etcd -data-dir $DATA_DIR --listen-peer-urls http://127.0.0.1:2380 --listen-client-urls http://127.0.0.1:2379 --advertise-client-urls http://127.0.0.1:2379  1>/dev/null 2>/dev/null &
    ETCD_PID=$!

    # wait for ETCD to start
    while ! (echo -n 1>/dev/null 2>/dev/null > /dev/tcp/localhost/2379); do sleep 1; done

    # append to path gohan binary path
    export PATH=$GOHANPATH:$PATH

    # run gohan server
    gohan server --config-file etc/gohan.yaml 1>/dev/null 2>/dev/null &
    SERVER_PID=$!

    # wait for gohan server to start
    while ! (echo -n 1>/dev/null 2>/dev/null > /dev/tcp/localhost/9091); do sleep 1; done

    # configure gohan client
    source etc/gohan_client.rc

    # enter tools
    cd tools

    # test bash completion
    source ./bash_completion.sh
    ./bash_completion_tests.py

    # test gohan client bash completion
    source ./gohan_client_bash_completion.sh
    ./gohan_client_bash_completion_tests.py

    # kill server
    kill $SERVER_PID

    # kill etcd
    kill $ETCD_PID
)
