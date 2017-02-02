#!/usr/bin/env bash
set -e

docker run -d --name gohan-mysql \
-e MYSQL_ROOT_PASSWORD=password \
-e MYSQL_USER=gohan \
-e MYSQL_PASSWORD=gohan \
-e MYSQL_DATABASE=gohan_test \
-p 127.0.0.1:3306:3306 \
mysql:5.5