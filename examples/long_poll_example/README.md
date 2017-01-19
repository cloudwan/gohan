long polling example
---------------------

In this example, we show how you can perform long polling of resources. The example uses curl.

> **Note**: etcd must be running (on localhost:4001 by default). You can change this in gohan.yaml.

long polling List API
----------------------

First, perform normal GET request using List API:

```
curl localhost:9091/v1.0/store/pets/ -i
```

Example response:

```
HTTP/1.1 200 OK
Etag: 7b2f681a074dc89a5c918eb9ddbfec79
...
{
  "pets": []
}
```

In the response headers, you will notice ``Etag``:

```
Etag: 7b2f681a074dc89a5c918eb9ddbfec79
```

The value of Etag will be different depending on response content.
Now, in a separate terminal, start a long polling GET request of list of resources, by providing a ``Long-Poll`` header with value of just received ``Etag`` header:

```
curl localhost:9091/v1.0/store/pets/ -H 'Long-Poll: 7b2f681a074dc89a5c918eb9ddbfec79' -i
```

This command should block. In another terminal create a new resource:

```
(in another terminal)
curl http://localhost:9091/v1.0/store/pets/ -H "Content-Type: application/json" -X POST -d '{"name":"fluffy"}'  -i
```

Notice that the long polling request returned, and that it contains a new ``Etag``:

```
HTTP/1.1 200 OK
Etag: 85d39266ed1e01c70265331848edd5d4
...
{
  "pets": [
    {
      "description": null,
      "id": "881e436d-2edc-4c28-97cb-ee448e0f2cc8",
      "name": "fluffy",
      "status": null,
      "tenant_id": "admin"
    }
  ]
}
```

This new ``Etag`` value can be used to perform new long polling request:

```
curl localhost:9091/v1.0/store/pets/ -H 'Long-Poll: 85d39266ed1e01c70265331848edd5d4' -i
```

Trying to modify an existing resource will also cause the long polling request to return:

```
(in another terminal)
curl localhost:9091/v1.0/store/pets/881e436d-2edc-4c28-97cb-ee448e0f2cc8 -H "Content-Type: application/json" -X PUT -d '{"name":"modified_fluffy"}' -i
```

Example response:

```
HTTP/1.1 200 OK
Etag: 323d37c537e500e8a3385a55466653be
...
{
  "pets": [
    {
      "description": null,
      "id": "881e436d-2edc-4c28-97cb-ee448e0f2cc8",
      "name": "modified_fluffy",
      "status": null,
      "tenant_id": "admin"
    }
  ]
}
```

Deletion of existing resource will also cause Gohan to respond (remember to use the new Etag in the request):

```
curl localhost:9091/v1.0/store/pets/ -H 'Long-Poll: 323d37c537e500e8a3385a55466653be' -i
```

```
(in another terminal)
curl http://localhost:9091/v1.0/store/pets/881e436d-2edc-4c28-97cb-ee448e0f2cc8 -X DELETE -i
```

Example response:

```
HTTP/1.1 200 OK
Etag: 7b2f681a074dc89a5c918eb9ddbfec79
...
{
  "pets": []
}
```

long polling Show API
----------------------

To long poll a specific resource, it must already exist. So create a new resource:

```
(in another terminal)
curl http://localhost:9091/v1.0/store/pets/ -H "Content-Type: application/json" -X POST -d '{"name":"reksio"}' -i
```

Use "id" field contained in the response to perform curl request of specific resource.

```
curl localhost:9091/v1.0/store/pets/5596c073-6cd2-448f-b862-1318421025d4 -i
```

Example response: 

```
HTTP/1.1 200 OK
Etag: 564b7733ccc65ce66a6a99dcadf04e70
...
{
  "pet": {
    "description": null,
    "id": "5596c073-6cd2-448f-b862-1318421025d4",
    "name": "reksio",
    "status": null,
    "tenant_id": "admin"
  }
}
```

This will return an ``Etag`` associated with that specific resource. Use it to perform a long polling request:

```
curl localhost:9091/v1.0/store/pets/5596c073-6cd2-448f-b862-1318421025d4 -H 'Long-Poll: 564b7733ccc65ce66a6a99dcadf04e70' -i
```

Creating a new resource will not cause the request to return:

```
(in another terminal)
curl  http://localhost:991/v1.0/store/pets/ -H "Content-Type: application/json" -X POST -d '{"name":"lajka"}' -i
````

But modyfing the long-polled resource - will:

```
(in another terminal)
curl localhost:9091/v1.0/store/pets/5596c073-6cd2-448f-b862-1318421025d4 -H "Content-Type: application/json" -X PUT -d '{"name":"modified_reksio"}' -i
```

Example response:

```
HTTP/1.1 200 OK
Etag: 0b939757e0a3896117449d7c7f502d79
...
{
  "pet": {
    "description": null,
    "id": "5596c073-6cd2-448f-b862-1318421025d4",
    "name": "modified_reksio",
    "status": null,
    "tenant_id": "admin"
  }
}
```

Perform a new long polled request using new Etag:

```
curl localhost:9091/v1.0/store/pets/5596c073-6cd2-448f-b862-1318421025d4 -H 'Long-Poll: 0b939757e0a3896117449d7c7f502d79' -i
```

Deleting queried resource also causes the request to return:

```
(in another terminal)
curl localhost:9091/v1.0/store/pets/5596c073-6cd2-448f-b862-1318421025d4 -X DELETE -i
```

Example response:

```
HTTP/1.1 404 Not Found
{"error":""}
```