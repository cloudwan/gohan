#!/bin/bash
set -xe

go build -ldflags "-X main.BuildVersion=`git rev-parse HEAD`" $@

