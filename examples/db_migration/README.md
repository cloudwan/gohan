DB migration example
---------------------

We show how we can use db versioing with Gohan.
Bascially, Gohan won't create table if it has already exist.
so you can use any DB migration tool.

However, Gohan provides helper utility to generate migration file.

Step1
-------

Prepare goose configuraion file.
We have a example in db/dbconf.yml

``` yaml
development:
    driver: sqlite3
    open: ./gohan.db
```

Step2
--------

Generate initial migration file

``` shell
gohan migrate -path db/migrations/ --schema example_schema.yaml
```

Step3
--------

apply goose file

``` shell
$ goose up
goose: migrating db environment 'development', current version: 0, target: 20151110132025
OK    20151110132025_init_schema.sql

```

Step4
---------

Update schema


Step4
---------

Add goose migration code in db/migrations.
Note that Gohan can't produce goose file from diff of schema, so you need
 to write it.

```
-- +goose Up
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE post;
```

Step5
----------

Keep goose up
