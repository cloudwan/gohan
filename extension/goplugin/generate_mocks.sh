#!/usr/bin/env bash

file="environment_mock_gen.go"
mockgen -package goext github.com/cloudwan/gohan/extension/goext ICore,ILogger,ISchemas,ISync,IDatabase,ITransaction,IHTTP,IAuth,IConfig,IUtil,IFile > ${file}
sed -i '/goext"$/d;s/goext\.//g' ${file}
mv ${file} ../goext/
