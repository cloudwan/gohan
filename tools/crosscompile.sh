#!/bin/bash

mkdir dist
cd dist
xgo github.com/cloudwan/gohan
for binary in $(ls); do
    zip -r ${binary}.zip ${binary}
done