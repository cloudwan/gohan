#!/bin/bash

mkdir dist
rm -rf dist/*
cd dist
xgo --targets=linux/amd64 github.com/cloudwan/gohan
for binary in $(ls); do
    zip -r ${binary}.zip ${binary}
    rm -rf ${binary}
done
