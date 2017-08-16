// Copyright (C) 2015 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/jmoiron/sqlx"

	"context"

	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

//DB is yaml implementation of DB
//This db backend is intended for development and test purpose only
type DB struct {
	filePath string
	data     map[string]interface{}

	// options
	options options.Options
}

//Transaction is yaml implementation of DB
//This db backend is intended for development and test purpose only
type Transaction struct {
	db *DB
}

//NewDB constructor
func NewDB(options options.Options) *DB {
	return &DB{options: options}
}

//Options return DB options
func (db *DB) Options() options.Options {
	return db.options
}

//Connect connec to the db
func (db *DB) Connect(format, conn string, maxOpenConn int) error {
	db.filePath = conn
	db.load()
	return nil
}

//Close db
func (db *DB) Close() {
	// nothing to do
}

//Begin connection starts new transaction
func (db *DB) Begin() (transaction.Transaction, error) {
	return &Transaction{
		db: db,
	}, nil
}

//BeginTx connection starts new transaction with given transaction options
func (db *DB) BeginTx(_ context.Context, _ *transaction.TxOptions) (transaction.Transaction, error) {
	return &Transaction{
		db: db,
	}, nil
}

//Close connection
func (tx *Transaction) Close() error {
	return nil
}

//Closed is unsupported in this db
func (tx *Transaction) Closed() bool {
	return false
}

//GetIsolationLevel is unsupported in this db
func (tx *Transaction) GetIsolationLevel() transaction.Type {
	return ""
}

//RegisterTable register table definition
func (db *DB) RegisterTable(s *schema.Schema, cascade, migrate bool) error {
	return nil
}

//DropTable drop table definition
func (db *DB) DropTable(s *schema.Schema) error {
	return nil
}

func (db *DB) load() error {
	data, err := util.LoadMap(db.filePath)
	if err != nil {
		db.data = map[string]interface{}{}
		return err
	}
	db.data = data
	return nil
}

func (db *DB) write() error {
	return util.SaveFile(db.filePath, db.data)
}

func (db *DB) getTable(s *schema.Schema) []interface{} {
	rawTable, ok := db.data[s.GetDbTableName()]
	if ok {
		return rawTable.([]interface{})
	}
	newTable := []interface{}{}
	db.data[s.GetDbTableName()] = newTable
	return newTable
}

//Commit commits changes to db
//Unsupported in this db
func (tx *Transaction) Commit() error {
	return nil
}

//Create create resource in the db
func (tx *Transaction) Create(resource *schema.Resource) error {
	db := tx.db
	db.load()
	s := resource.Schema()
	data := resource.Data()
	table := db.getTable(s)
	db.data[s.GetDbTableName()] = append(table, data)
	db.write()
	return nil
}

//Update update resource in the db
func (tx *Transaction) Update(resource *schema.Resource) error {
	db := tx.db
	db.load()
	s := resource.Schema()
	data := resource.Data()
	table := db.getTable(s)
	for _, rawDataInDB := range table {
		dataInDB := rawDataInDB.(map[string]interface{})
		if dataInDB["id"] == resource.ID() {
			for key, value := range data {
				dataInDB[key] = value
			}
		}
	}
	db.write()
	return nil
}

//StateUpdate update resource state
func (tx *Transaction) StateUpdate(resource *schema.Resource, _ *transaction.ResourceState) error {
	return tx.Update(resource)
}

//Delete delete resource from db
func (tx *Transaction) Delete(s *schema.Schema, resourceID interface{}) error {
	db := tx.db
	db.load()
	table := db.getTable(s)
	newTable := []interface{}{}
	for _, rawDataInDB := range table {
		dataInDB := rawDataInDB.(map[string]interface{})
		if dataInDB["id"] != resourceID {
			newTable = append(newTable, dataInDB)
		}
	}
	db.data[s.GetDbTableName()] = newTable
	db.write()
	return nil
}

type byPaginator struct {
	data []*schema.Resource
	pg   *pagination.Paginator
}

func (s byPaginator) Len() int {
	return len(s.data)
}
func (s byPaginator) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}
func (s byPaginator) Less(i, j int) bool {
	key := s.pg.Key
	vi := s.data[i].Get(key)
	vj := s.data[j].Get(key)
	var less bool

	switch vi.(type) {
	case int:
		less = vi.(int) < vj.(int)
	case float64:
		less = vi.(float64) < vj.(float64)
	case string:
		less = vi.(string) < vj.(string)
	default:
		panic(fmt.Sprintf("uncomparable type %T", vi))
	}

	if s.pg.Order == pagination.DESC {
		return !less
	}
	return less
}

//List resources in the db
func (tx *Transaction) List(s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator) (list []*schema.Resource, total uint64, err error) {
	db := tx.db
	db.load()
	table := db.getTable(s)
	for _, rawData := range table {
		data := rawData.(map[string]interface{})
		var resource *schema.Resource
		resource, err = schema.NewResource(s, data)
		if err != nil {
			log.Warning("%s %s", resource, err)
			return
		}
		valid := true
		if filter != nil {
			for key, value := range filter {
				if data[key] == nil {
					continue
				}
				property, err := s.GetPropertyByID(key)
				if err != nil {
					continue
				}
				switch value.(type) {
				case string:
					if property.Type == "boolean" {
						dataBool, err1 := strconv.ParseBool(data[key].(string))
						valueBool, err2 := strconv.ParseBool(value.(string))
						if err1 != nil || err2 != nil || dataBool != valueBool {
							valid = false
						}
					} else if data[key] != value {
						valid = false
					}
				case []string:
					if property.Type == "boolean" {
						v, _ := strconv.ParseBool(data[key].(string))
						if !boolInSlice(v, value.([]string)) {
							valid = false
						}
					}
					if !stringInSlice(fmt.Sprintf("%v", data[key]), value.([]string)) {
						valid = false
					}
				default:
					if data[key] != value {
						valid = false
					}
				}
			}
		}
		if valid {
			list = append(list, resource)
		}

		if pg != nil {
			sort.Sort(byPaginator{list, pg})
			if pg.Limit > 0 {
				list = list[:pg.Limit]
			}
		}
	}
	total = uint64(len(list))
	return
}

// LockList locks resources in the db. Not supported in file db
func (tx *Transaction) LockList(s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions, pg *pagination.Paginator, policy schema.LockPolicy) (list []*schema.Resource, total uint64, err error) {
	return tx.List(s, filter, options, pg)
}

//Fetch resources by ID in the db
func (tx *Transaction) Fetch(s *schema.Schema, filter transaction.Filter, options *transaction.ViewOptions) (*schema.Resource, error) {
	list, _, err := tx.List(s, filter, options, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch %s: %s", filter, err)
	}
	if len(list) < 1 {
		return nil, transaction.ErrResourceNotFound
	}
	return list[0], nil
}

// LockFetch fetches & locks a resource. Not supported in file db
func (tx *Transaction) LockFetch(s *schema.Schema, filter transaction.Filter, policy schema.LockPolicy, options *transaction.ViewOptions) (*schema.Resource, error) {
	return tx.Fetch(s, filter, options)
}

//StateFetch is not supported in file databases
func (tx *Transaction) StateFetch(s *schema.Schema, filter transaction.Filter) (state transaction.ResourceState, err error) {
	err = fmt.Errorf("StateFetch is not supported for file databases")
	return
}

//SetIsolationLevel specify transaction isolation level
func (tx *Transaction) SetIsolationLevel(level transaction.Type) error {
	return nil
}

//RawTransaction returns raw transaction
func (tx *Transaction) RawTransaction() *sqlx.Tx {
	panic("Not implemented")
}

// Query with raw string
func (tx *Transaction) Query(s *schema.Schema, query string, arguments []interface{}) (list []*schema.Resource, err error) {
	panic("Not implemented")
}

// Exec executes sql in transaction
func (tx *Transaction) Exec(sql string, args ...interface{}) error {
	panic("Not implemented")
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func boolInSlice(a bool, list []string) bool {
	for _, b := range list {
		v, _ := strconv.ParseBool(b)
		if v == a {
			return true
		}
	}
	return false
}
