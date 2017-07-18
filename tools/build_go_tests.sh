#!/bin/sh

# build golang plugins used in unit tests
( cd extension/golang/test_data && make clean && make BUILD_OPTS=$@ )

