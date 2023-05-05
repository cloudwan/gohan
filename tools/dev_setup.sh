#!/bin/bash

go install github.com/kardianos/govendor@latest
go install golang.org/x/lint/golint@latest
go install github.com/gordonklaus/ineffassign@latest
go install github.com/kevinburke/go-bindata/go-bindata@v3.2.0
go install github.com/client9/misspell/cmd/misspell@latest
go install github.com/golang/mock/mockgen@latest
