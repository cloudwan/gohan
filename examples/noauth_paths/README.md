Gohan NoAuth Path
--------------------

In this example, we show how to define a list of paths that never
require an authorization. A single path is defined as a regular
expression (regex).

In order to define a list of not autorized paths, add the following entry
in the configuration file:

``` yaml
route:
    noauth_paths:
    - "/v1.0/store/p[a-zA-Z0-9]*"
```
In the example, there is only one path not authorized. It is a regex that matches
all paths begining with '/v1.0/store/p'

Test script
------------------

An example script is provided in 'examples/noauth_paths'. It can be run by invoking:
``` bash
./examples/noauth_paths/curl_test.sh
```

In the example, three scenarios are tested:

* a given path does not exist; the result should be 401.

* a given path exists but it requires an authorization; the result should be 401.

* a given path exists and it does not require authorization; the result should be 200.

