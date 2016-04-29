# go-fakedb

fake db driver for go sql/driver for benchmark purpose

# Usage

``` go
db, err := sql.Open("fakedb", "example.csv")
if err != nil {
    t.Fatal(err)
}
defer db.Close()
rows, err := db.Query("select id, name from foo")
if err != nil {
    t.Fatal(err)
}
defer rows.Close()
var id string
var name string
rows.Next()
rows.Scan(&id, &name)
    
```

# Test data

if query contains "count" it returns count, otherwise it returns contents in csv file
specified in dsn. Note that the first line must be a header. see [example](./example.csv)