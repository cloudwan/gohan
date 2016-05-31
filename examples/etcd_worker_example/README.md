etcd based worker example
--------------------------

In this example, we show how we can
execute extension on etcd notification.

You need configure watch keys and events in order to use this function.

``` yaml
watch:
    keys:
      - config/v1.0/store/
    events:
      - config/v1.0/store/pets
    worker_count: 4

etcd:
    - http://127.0.0.1:4001
```

Then you can execute extension using sync:// + event path.

```
extensions:
- id: sync
  path: sync://config/v1.0/store/pets
  code_type: gohanscript
  code: |
    tasks:
    - debug:
schemas: []
```