Gohan MYSQL example
--------------------

In this example, we show mysql config example.

You can run gohan + mysql with this command.

MYSQL_PASSWORD=$MYSQL_PASSWORD gohan server --config-file gohan.yaml

``` yaml
# database connection configuraion
database:
    type: "mysql"
    connection: "root:{{.MYSQL_PASSWORD}}@/gohan"
```

You can see more mysql configuraion here
https://github.com/go-sql-driver/mysql

Simple test script
------------------

You can test concurrent db asccess using test.sh.
This example create network resource with 3 process.

``` shell
./test.sh &
./test.sh &
./test.sh &
```
