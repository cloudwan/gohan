#!/bin/bash

#set -e

function lint() {
  for file in `find . -name "*.go" | grep -Ev "vendor|bindata\.go|mocks|op\.go"`; do
    gofmt -s -d $file
    golint $file
    go vet $file
    misspell -error $file
    ineffassign $file
  done
}

output=$(mktemp)
trap 'rm $output' EXIT

lint &> $output

cat $output

if [[ -s $output ]]; then
  exit 1
fi
