// Copyright (C) 2016  Juniper Networks, Inc.
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

package lib

import (
	"fmt"
	"strings"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

func init() {
	gohanscript.RegisterStmtParser("transaction", transactionFunc)
}

func transactionFunc(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
	stmts, err := gohanscript.MakeStmts(stmt.File, stmt.RawNode["transaction"])
	if err != nil {
		return nil, err
	}
	runner, err := gohanscript.StmtsToFunc("transaction", stmts)
	if err != nil {
		return nil, err
	}
	return func(context *gohanscript.Context) (interface{}, error) {
		dbVar := util.MaybeString(stmt.RawData["db"])
		if dbVar == "" {
			dbVar = "db"
		}
		rawDB, err := context.Get(dbVar)
		if err != nil {
			return nil, err
		}
		connection := rawDB.(db.DB)
		tx, err := connection.Begin()
		if err != nil {
			return nil, err
		}
		transactionVar := util.MaybeString(stmt.RawData["transaction_var"])
		if transactionVar == "" {
			transactionVar = "transaction"
		}
		context.Set(transactionVar, tx)
		value, err := runner(context)
		if err == nil {
			tx.Commit()
		}
		tx.Close()
		return value, err
	}, nil
}

//GohanSchema returns gohan schema object by schemaID.
func GohanSchema(schemaID string) (*schema.Schema, error) {
	var err error
	manager := schema.GetManager()
	schema, ok := manager.Schema(schemaID)
	if !ok {
		err = fmt.Errorf("Schema %s isn't loaded", schemaID)
	}
	return schema, err
}

//GohanSchemas returns map of schemas.
func GohanSchemas() schema.Map {
	manager := schema.GetManager()
	return manager.Schemas()
}

//GohanPolicies returns all policies
func GohanPolicies() []*schema.Policy {
	manager := schema.GetManager()
	return manager.Policies()
}

//ReadConfig reads configuraion from file.
func ReadConfig(path string) error {
	config := util.GetConfig()
	err := config.ReadConfig(path)
	return err
}

//GetConfig returns config by key.
func GetConfig(key string, defaultValue interface{}) interface{} {
	config := util.GetConfig()
	return config.GetParam(key, defaultValue)
}

//GohanLoadSchema loads schema from path.
func GohanLoadSchema(src string) (interface{}, error) {
	manager := schema.GetManager()
	err := manager.LoadSchemaFromFile(src)
	return nil, err
}

//ConnectDB start connection to db
func ConnectDB(dbType string, connection string, maxOpenConn int) (db.DB, error) {
	return db.ConnectDB(dbType, connection, maxOpenConn, options.Default())
}

//InitDB initializes database using schema.
func InitDB(dbType string, connection string, dropOnCreate bool, cascade bool) error {
	err := db.InitDBWithSchemas(dbType, connection, dropOnCreate, cascade, false)
	return err
}

//DBBegin starts transaction
func DBBegin(connection db.DB) (transaction.Transaction, error) {
	return connection.Begin()
}

//DBCommit commits transaction
func DBCommit(tx transaction.Transaction) error {
	return tx.Commit()
}

//DBClose closes a transaction.
func DBClose(tx transaction.Transaction) error {
	return tx.Close()
}

//DBGet get resource from a db.
func DBGet(tx transaction.Transaction, schemaID string, id string, tenantID string) (map[string]interface{}, error) {
	manager := schema.GetManager()
	schemaObj, ok := manager.Schema(schemaID)
	if !ok {
		return nil, fmt.Errorf("Schema %s not found", schemaID)
	}
	filter := transaction.IDFilter(id)
	if tenantID != "" {
		filter["tenant_id"] = tenantID
	}
	resp, err := tx.Fetch(schemaObj, filter, nil)
	if err != nil {
		return nil, err
	}
	return resp.Data(), err
}

//DBCreate creates a resource in a db.
func DBCreate(tx transaction.Transaction, schemaID string, data map[string]interface{}) error {
	manager := schema.GetManager()
	resource, err := manager.LoadResource(schemaID, data)
	if err != nil {
		return err
	}
	return tx.Create(resource)
}

//DBList lists data from database.
func DBList(tx transaction.Transaction, schemaID string, filter map[string]interface{}) ([]interface{}, error) {
	manager := schema.GetManager()
	schemaObj, ok := manager.Schema(schemaID)
	if !ok {
		return nil, fmt.Errorf("Schema %s not found", schemaID)
	}
	for key, value := range filter {
		switch v := value.(type) {
		case string:
			filter[key] = []string{v}
		case bool:
			filter[key] = []string{fmt.Sprintf("%v", v)}
		case int:
			filter[key] = []string{fmt.Sprintf("%v", v)}
		case []interface{}:
			filterList := make([]string, len(v))
			for _, item := range v {
				filterList = append(filterList, fmt.Sprintf("%v", item))
			}
			filter[key] = filterList
		}
	}
	resources, _, err := tx.List(schemaObj, filter, nil, nil)
	resp := []interface{}{}
	for _, resource := range resources {
		resp = append(resp, resource.Data())
	}
	return resp, err
}

//DBUpdate updates a resource in a db.
func DBUpdate(tx transaction.Transaction, schemaID string, data map[string]interface{}) error {
	manager := schema.GetManager()
	resource, err := manager.LoadResource(schemaID, data)
	if err != nil {
		return err
	}
	return tx.Update(resource)
}

//DBDelete deletes a resource in a db.
func DBDelete(tx transaction.Transaction, schemaID string, id string) error {
	manager := schema.GetManager()
	schemaObj, ok := manager.Schema(schemaID)
	if !ok {
		return fmt.Errorf("Schema %s not found", schemaID)
	}
	return tx.Delete(schemaObj, id)
}

//DBQuery fetchs data from db with additional query
func DBQuery(tx transaction.Transaction, schemaID string, sql string, arguments []interface{}) ([]interface{}, error) {
	manager := schema.GetManager()
	schemaObj, ok := manager.Schema(schemaID)
	if !ok {
		return nil, fmt.Errorf("Schema %s not found", schemaID)
	}
	resources, err := tx.Query(schemaObj, sql, arguments)
	resp := []interface{}{}
	for _, resource := range resources {
		resp = append(resp, resource.Data())
	}
	return resp, err
}

//DBExec closes a transaction.
func DBExec(tx transaction.Transaction, sql string, arguments []interface{}) error {
	return tx.Exec(sql, arguments...)
}

//DBColumn makes partiall part of sql query from schema
func DBColumn(schemaID string, join bool) (string, error) {
	manager := schema.GetManager()
	schemaObj, ok := manager.Schema(schemaID)
	if !ok {
		return "", fmt.Errorf("Schema %s not found", schemaID)
	}
	return strings.Join(sql.MakeColumns(schemaObj, schemaObj.GetDbTableName(), nil, join), ", "), nil
}

//Error returns extension Error
func Error(code int, name, message string) error {
	return extension.Errorf(code, name, message)
}
