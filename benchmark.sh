#!/bin/bash

# Note you need to configure mysql
mysql -uroot -e "drop database if exists gohan_test; create database gohan_test;";
cd server
go test -run none -bench .
