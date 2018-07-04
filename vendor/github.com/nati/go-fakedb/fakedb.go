package fakedb

import (
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

var d *Driver

func init() {
	d = &Driver{}
	sql.Register("fakedb", d)
}

type Driver struct {
    header []string
	data   [][]string
}

type Conn struct {
	header []string
	data   [][]string
}

type Stmt struct {
	conn  *Conn
	count bool
	input int
}

type Result struct {
	conn *Conn
}

type Tx struct {
	conn *Conn
}

type DataRows struct {
	conn *Conn
	line int
}

type CountRows struct {
	conn *Conn
	line int
}

func (d *Driver) Open(fileName string) (driver.Conn, error) {
    if d.header == nil {
     	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	csvReader := csv.NewReader(file)
	data, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("missing header")
	}
    d.header = data[0]
    d.data = data[1:]   
    }

	return &Conn{
		header: d.header,
		data:   d.data}, nil
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return &Stmt{conn: c,
		count: strings.Contains(query, "count"),
		input: strings.Count(query, "?")}, nil
}

func (c *Conn) Close() error {
	return nil
}

func (c *Conn) Begin() (driver.Tx, error) {
	return &Tx{conn: c}, nil
}

func (r *Result) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *Result) RowsAffected() (int64, error) {
	return 0, nil
}

func (s *Stmt) Close() error {
	return nil
}

func (s *Stmt) NumInput() int {
	return s.input
}

func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	return &Result{conn: s.conn}, nil
}

func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.count {
		return &CountRows{line: 0, conn: s.conn}, nil
	}
	return &DataRows{line: 0, conn: s.conn}, nil
}

func (r *DataRows) Columns() []string {
	return r.conn.header
}

func (r *DataRows) Close() error {
	return nil
}

func (r *DataRows) Next(dest []driver.Value) error {
	if r.line >= len(r.conn.data) {
		return io.EOF
	}
	for i := range r.conn.header {
		dest[i] = r.conn.data[r.line][i]
	}
	r.line++
	return nil
}

func (r *CountRows) Columns() []string {
	return []string{"count"}
}

func (r *CountRows) Close() error {
	return nil
}

func (r *CountRows) Next(dest []driver.Value) error {
	if r.line > 0 {
		return io.EOF
	}
	dest[0] = int64(len(r.conn.data))
	r.line++
	return nil
}

func (tx *Tx) Commit() error {
	return nil
}

func (tx *Tx) Rollback() error {
	return nil
}
