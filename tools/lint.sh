#!/bin/bash

#set -e

for file in `find . -name "*.go" | grep -v vendor | grep -v bindata.go |grep -v mocks | grep -v op.go`; do
  golint $file
  go vet $file
  misspell -error $file
  ineffassign $file
done
