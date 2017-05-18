#!/bin/bash

mkdir dist
rm -rf dist/*
cd dist
xgo --targets=windows/amd64,darwin/amd64,linux/amd64 github.com/cloudwan/gohan
for binary in $(ls); do
    zip -r ${binary}.zip ${binary}
    rm -rf ${binary}
done
