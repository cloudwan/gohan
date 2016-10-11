Gohan path whitelist example
--------------------

In this example, we show how to define a list of paths that never
require an authorization. A single path is defined as a regular
expression (regex).

In order to define a whitelisted path, add the following entry
in the configuration file:

``` yaml
# path whitelist configuration
path_whitelist:
    - "/v1.0/store/p[a-zA-Z0-9]*"
```
In the example, there is only one path whitelisted. It is a regex that matches
all paths begining with '/v1.0/store/p'

Test script
------------------

An example script is provided in 'examples/path_whitelist'. It can be run by invoking:
``` bash
./examples/path_whitelist/test.sh
```

In the example, three scenarios are tested:

* a given path does not exist; the result should be 403.

* a given path exists but it requires an authorization.

* a given path exists and it is whitelisted - no authorization is needed.

