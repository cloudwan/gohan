#!/bin/bash

echo "[gohan.yaml] path '/v1.0/store/p[a-zA-Z0-9]*' is whitelisted"

# 1
echo -e "\e[35m\e[1m"
echo "[1] test a non-existing path '/v1.0/store/rats'"
echo "    expected result: (401) Unauthorized"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v1.0/store/rats

# 2
echo -e "\e[35m\e[1m"
echo "[2] test an existing and NOT whitelisted path '/v1.0/store/orders'"
echo "    expected result: (401) Unauthorized"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v1.0/store/orders

# 3
echo -e "\e[35m\e[1m"
echo "[1] testing an existing and whitelisted path '/v1.0/store/pets'"
echo "    expected result: (200) OK"
echo -e "\e[0m"
curl -v 127.0.0.1:9091/v1.0/store/pets

