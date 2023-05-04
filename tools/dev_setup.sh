#!/bin/bash

go install github.com/kardianos/govendor
go install golang.org/x/lint/golint
go install github.com/gordonklaus/ineffassign
go install github.com/kevinburke/go-bindata/...
go install github.com/client9/misspell/cmd/misspell
go install github.com/golang/mock/mockgen
