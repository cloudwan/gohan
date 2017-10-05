#!/bin/bash
set -xe

BUILD_VERSION=`git rev-parse HEAD`
BUILD_TIMESTAMP=`date -u '+%Y-%m-%d_%I:%M:%S%p_utc'`
BUILD_HOST=`uname -n`

echo "BUILD_VERSION: ${BUILD_VERSION}"
echo "BUILD_TIMESTAMP: ${BUILD_TIMESTAMP}"
echo "BUILD_HOST: ${BUILD_HOST}"

go build -ldflags "-X main.buildVersion=${BUILD_VERSION} -X main.buildTimestamp=${BUILD_TIMESTAMP} -X main.buildHost=${BUILD_HOST}" $@
