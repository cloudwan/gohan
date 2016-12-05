CRON Job example
------------------

In this example, we show how we can
execute extension pediorically.

CRON depends on etcd for distributed locks,
so you need to configure etcd for Gohan.

``` yaml
cron:
    - path: cron://cron_job_sample
      timing: "*/5 * * * * *"

etcd:
    - http://127.0.0.1:2379
```

In extension, you need to specify path which maches specified in the config file.