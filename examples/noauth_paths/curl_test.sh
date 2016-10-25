#!/bin/bash

# 1
echo -e "\e[35m\e[1m"
echo "[1] test a non-existing path '/v1.0/nonexisting'"
echo "    expected result: (401) Unauthorized"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v1.0/nonexisting

# 2
echo -e "\e[35m\e[1m"
echo "[2] test an existing and NOT whitelisted path '/v1.0/apples'"
echo "    expected result: (401) Unauthorized"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v1.0/apples

# 3
echo -e "\e[35m\e[1m"
echo "[1] testing an existing and whitelisted path '/v1.0/oranges'"
echo "    expected result: (200) OK"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v1.0/oranges
