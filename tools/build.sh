#!/bin/bash

set -xe

# go run ./extension/gohanscript/tools/gen.go genlib -t extension/gohanscript/templates/lib.tmpl -p github.com/cloudwan/gohan/extension/gohanscript/lib -e autogen -ep extension/gohanscript/autogen
go build -ldflags "-X main.BuildVersion=`git rev-parse HEAD`" main.go

