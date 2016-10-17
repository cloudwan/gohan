#!/bin/bash

# 1
echo -e "\e[35m\e[1m"
echo "[1] test a non-existing path '/v0.1/missing'"
echo "    expected result: (401) Unauthorized"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v0.1/missing

# 2i
echo -e "\e[35m\e[1m"
echo "[2] test an existing and NOT whitelisted path '/v0.1/admin_only_resources'"
echo "    expected result: (401) Unauthorized"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v0.1/admin_only_resources

# 3
echo -e "\e[35m\e[1m"
echo "[1] testing an existing and whitelisted path '/v0.1/member_resources'"
echo "    expected result: (200) OK"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v0.1/member_resources
