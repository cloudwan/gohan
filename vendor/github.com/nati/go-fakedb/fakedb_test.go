package fakedb

import (
	"database/sql"
	"testing"
)

func TestSQL(t *testing.T) {
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
	if !(id == "alice" && name == "Alice") {
        t.Fail()
    }
	rows.Next()
	rows.Scan(&id, &name)
	if !(id == "bob" && name == "Bob") {
        t.Fail()
    }
	rows, err = db.Query("select count(id) as Count from foo")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var count int64
	rows.Next()
	rows.Scan(&count)
	if count != 2 {
        t.Fail()
    }
}
