==================
Benchmarking Tips
==================

We can use boom for benchmarking gohan.
https://github.com/rakyll/boom

How to install
---------------

.. code-block:: shell

  go get github.com/rakyll/boom


Sample Server Configuraion
---------------------------

.. code-block:: shell

  gohan server --config-file server/server_test_mysql_config.yaml


Sample Benchmark
---------------------------

Note: this benchmark result get done in my local laptop where benchmark client, server and
database working. We need proper benchmark result in doc in future.

POST

.. code-block:: shell

  $ boom -n 1000 -c 100 -m POST -h "X-Auth-Token:admin_token" -d '{"name": "hoge", "tenant_id": "red"}' http://localhost:19090/v2.0/networks/
  1000 / 1000 Boooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo! 100.00 %

  Summary:
    Total:	1.3190 secs.
    Slowest:	0.1941 secs.
    Fastest:	0.0104 secs.
    Average:	0.1303 secs.
    Requests/sec:	758.1509
    Total Data Received:	213000 bytes.
    Response Size per Request:	213 bytes.

  Status code distribution:
    [201]	1000 responses

  Response time histogram:
    0.010 [1]	|
    0.029 [0]	|
    0.047 [0]	|
    0.065 [15]	|∎
    0.084 [21]	|∎∎
    0.102 [70]	|∎∎∎∎∎∎∎∎
    0.121 [183]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    0.139 [318]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    0.157 [311]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    0.176 [68]	|∎∎∎∎∎∎∎∎
    0.194 [13]	|∎

  Latency distribution:
    10% in 0.1007 secs.
    25% in 0.1175 secs.
    50% in 0.1324 secs.
    75% in 0.1447 secs.
    90% in 0.1554 secs.
    95% in 0.1649 secs.
    99% in 0.1766 secs.


Get 10 elements in list

.. code-block:: shell

  $ boom -n 1000 -c 100 -m GET -h "X-Auth-Token:admin_token"  http://localhost:19090/v2.0/networks/?lit=1
  1000 / 1000 Boooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo! 100.00 %

  Summary:
    Total:	1.3845 secs.
    Slowest:	0.2075 secs.
    Fastest:	0.0501 secs.
    Average:	0.1357 secs.
    Requests/sec:	722.3051
    Total Data Received:	240000 bytes.
    Response Size per Request:	240 bytes.

  Status code distribution:
    [200]	1000 responses

  Response time histogram:
    0.050 [1]	|
    0.066 [11]	|∎
    0.082 [34]	|∎∎∎∎
    0.097 [63]	|∎∎∎∎∎∎∎∎
    0.113 [82]	|∎∎∎∎∎∎∎∎∎∎∎
    0.129 [115]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    0.145 [287]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    0.160 [239]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    0.176 [138]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    0.192 [28]	|∎∎∎
    0.208 [2]	|

  Latency distribution:
    10% in 0.0963 secs.
    25% in 0.1224 secs.
    50% in 0.1393 secs.
    75% in 0.1549 secs.
    90% in 0.1649 secs.
    95% in 0.1720 secs.
    99% in 0.1793 secs.
