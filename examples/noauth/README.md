'Nobody resource paths' policy
--------------------

In this example, we show how to define a list of paths that never
require an authorization. A single path is defined as a regular
expression (regex). In order to define a list of not autorized paths,
add the following entry in the schema file:

``` yaml
policies:
- id: no_auth_favicon
  principal: Nobody
  resource:
    path: /favicon.ico
- id: no_auth_member_resources
  action: '*'
  principal: Nobody
  resource:
    path: /v0.1/member_resources*
```

See 'docs/policy.md' for more information.

Test script
------------------

An example script is provided in 'examples/noauth'. It can be run by invoking:
``` bash
./examples/noauth/curl_test.sh
```

In the example, three scenarios are tested:

* a given path does not exist; the result should be 401.

* a given path exists but it requires an authorization; the result should be 401.

* a given path exists and it does not require authorization; the result should be 200.
