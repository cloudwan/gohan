#!/bin/bash

for file in `find . -name "*.go" | grep -v vendor | grep -v go-bindata.go | grep -v op.go`; do
  golint $file
  go vet $file
  misspell $file
  ineffassign $file
done