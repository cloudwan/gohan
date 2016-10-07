#!/bin/bash

#set -e

for file in `find . -name "*.go" | grep -v vendor | grep -v go-bindata.go | grep -v op.go`; do
  golint $file
  go vet $file
  misspell -error $file
  ineffassign $file
done