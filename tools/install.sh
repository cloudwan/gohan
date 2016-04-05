#!/bin/bash

set -xe

go install -ldflags "-X main.BuildVersion=`git rev-parse HEAD`"