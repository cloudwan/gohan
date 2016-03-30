#!/bin/bash
for i in `seq 1 500`;
do
  curl 127.0.0.1:9091/v2.0/networks -XPOST -H "X-Auth-Token: admin_token" -d '{"name": "'$i'"}'  > /dev/null 2>&1
done
