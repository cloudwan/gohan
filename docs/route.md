# Route

One can define special policies for routes. Currently there is only
one setting which is a list of paths that do not require authorization.

## NoAuth Paths

These are the paths (defined by a list of regular expressions) which are never
checked against the current authorization. One of the possible usages is to allow
a device to register itself in gohan without knowing the authorization
credentials in advance.

Example entry in gohan configuration file:
```
route:
    noauth_paths:
        - "/v1.0/oran[a-z]*"
```
