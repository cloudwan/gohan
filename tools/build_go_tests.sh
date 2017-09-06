#!/bin/sh

# build goext plugins used in unit tests
( cd extension/goplugin/test_data && make clean && make BUILD_OPTS=$@ )

