#!/bin/bash

mkdir dist
cd dist
xgo --targets=windows/amd64,windows/386,darwin/386,darwin/amd64,linux/amd64 github.com/cloudwan/gohan
for binary in $(ls); do
    zip -r ${binary}.zip ${binary}
    rm ${binary}
done