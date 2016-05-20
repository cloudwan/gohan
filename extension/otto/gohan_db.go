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

package otto

import (
	"fmt"

	"github.com/dop251/otto"

	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
)

func init() {
	gohanDBInit := func(env *Environment) {
		vm := env.VM
		builtins := map[string]interface{}{
			"gohan_db_transaction": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_transaction", 0)
				tx, err := env.DataStore.Begin()
				if err != nil {
					ThrowOttoException(&call, "failed to start a transaction")
				}
				value, _ := vm.ToValue(tx)
				return value
			},
			"gohan_db_list": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) < 4 {
					defaultOrderKey, _ := otto.ToValue("") // no sorting
					call.ArgumentList = append(call.ArgumentList, defaultOrderKey)
				}
				if len(call.ArgumentList) < 5 {
					defaultLimit, _ := otto.ToValue(0) // no limit
					call.ArgumentList = append(call.ArgumentList, defaultLimit)
				}
				if len(call.ArgumentList) < 6 {
					defaultOffset, _ := otto.ToValue(0) // no offset
					call.ArgumentList = append(call.ArgumentList, defaultOffset)
				}
				VerifyCallArguments(&call, "gohan_db_list", 6)

				transaction, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer transaction.Close()
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				filter, err := GetMap(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				orderKey, err := GetString(call.Argument(3))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				rawLimit, err := GetInt64(call.Argument(4))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				limit := uint64(rawLimit)
				rawOffset, err := GetInt64(call.Argument(5))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				offset := uint64(rawOffset)

				resp, err := GohanDbList(transaction, schemaID, filter, orderKey, limit, offset)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_db_fetch": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_fetch", 4)
				transaction, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer transaction.Close()
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				ID, err := GetString(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				tenantID, err := GetString(call.Argument(3))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				resp, err := GohanDbFetch(transaction, schemaID, ID, tenantID)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				if resp == nil {
					otto.NullValue()
				}
				value, _ := vm.ToValue(resp.Data())
				return value
			},
			"gohan_db_state_fetch": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_state_fetch", 4)
				transaction, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer transaction.Close()
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				ID, err := GetString(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				tenantID, err := GetString(call.Argument(3))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				data, err := GohanDbStateFetch(transaction, schemaID, ID, tenantID)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				value, _ := vm.ToValue(data)
				return value
			},
			"gohan_db_create": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_create", 3)
				transaction, err := GetTransaction(call.Argument(0))
				transaction, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer transaction.Close()
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				dataMap, err := GetMap(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				resource, err := GohanDbCreate(transaction, needCommit, schemaID, dataMap)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				value, _ := vm.ToValue(resource.Data())
				return value
			},
			"gohan_db_update": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_update", 3)
				transaction, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer transaction.Close()
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				dataMap, err := GetMap(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				resource, err := GohanDbUpdate(transaction, needCommit, schemaID, dataMap)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				value, _ := vm.ToValue(resource.Data())
				return value
			},
			"gohan_db_state_update": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_state_update", 3)
				transaction, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer transaction.Close()
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				dataMap, err := GetMap(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				resource, err := GohanDbStateUpdate(transaction, needCommit, schemaID, dataMap)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				value, _ := vm.ToValue(resource.Data())
				return value
			},
			"gohan_db_delete": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_delete", 3)
				transaction, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer transaction.Close()
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				ID, err := GetString(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				err = GohanDbDelete(transaction, needCommit, schemaID, ID)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				return otto.NullValue()
			},
			"gohan_db_query": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_query", 4)
				transaction, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer transaction.Close()
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				sqlString, err := GetString(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				arguments, err := GetList(call.Argument(3))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				resp, err := GohanDbQuery(transaction, needCommit, schemaID, sqlString, arguments)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_db_sql_make_columns": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_query", 1)
				schemaID, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				results, err := GohanDbMakeColumns(schemaID)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				value, _ := vm.ToValue(results)
				return value
			},
		}

		for name, object := range builtins {
			vm.Set(name, object)
		}
	}
	RegisterInit(gohanDBInit)
}

//GohanDbList lists resources in database filtered by filter and paginator
func GohanDbList(transaction transaction.Transaction, schemaID string,
	filter map[string]interface{}, key string, limit uint64, offset uint64) ([]map[string]interface{}, error) {

	schema, err := getSchema(schemaID)
	if err != nil {
		return []map[string]interface{}{}, err
	}

	var paginator *pagination.Paginator
	if key != "" {
		paginator, err = pagination.NewPaginator(schema, key, "", limit, offset)
		if err != nil {
			return []map[string]interface{}{}, fmt.Errorf("Error during gohan_db_list: %s", err.Error())
		}
	}

	resources, _, err := transaction.List(schema, filter, paginator)
	if err != nil {
		return []map[string]interface{}{}, fmt.Errorf("Error during gohan_db_list: %s", err.Error())
	}

	resp := []map[string]interface{}{}
	for _, resource := range resources {
		resp = append(resp, resource.Data())
	}
	return resp, nil
}

//GohanDbFetch gets resource from database
func GohanDbFetch(transaction transaction.Transaction, schemaID, ID,
	tenantID string) (*schema.Resource, error) {

	schema, err := getSchema(schemaID)
	if err != nil {
		return nil, err
	}
	var tenantFilter []string
	if tenantID != "" {
		tenantFilter = []string{tenantID}
	}
	resp, err := transaction.Fetch(schema, ID, tenantFilter)
	if err != nil {
		return nil, fmt.Errorf("Error during gohan_db_fetch: %s", err.Error())
	}
	return resp, nil
}

//GohanDbStateFetch gets resource's state from database
func GohanDbStateFetch(transaction transaction.Transaction, schemaID, ID,
	tenantID string) (map[string]interface{}, error) {

	schema, err := getSchema(schemaID)
	if err != nil {
		return map[string]interface{}{}, err
	}
	var tenantFilter []string
	if tenantID != "" {
		tenantFilter = []string{tenantID}
	}
	resp, err := transaction.StateFetch(schema, ID, tenantFilter)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("Error during gohan_db_state_fetch: %s", err.Error())
	}
	data := map[string]interface{}{
		"config_version": resp.ConfigVersion,
		"state_version":  resp.StateVersion,
		"error":          resp.Error,
		"state":          resp.State,
		"monitoring":     resp.Monitoring,
	}
	return data, nil
}

//GohanDbCreate adds resource to database
func GohanDbCreate(transaction transaction.Transaction, needCommit bool, schemaID string,
	dataMap map[string]interface{}) (*schema.Resource, error) {

	manager := schema.GetManager()
	resource, err := manager.LoadResource(schemaID, dataMap)
	if err != nil {
		return nil, fmt.Errorf("Error during gohan_db_create: %s", err.Error())
	}
	resource.PopulateDefaults()
	if err = transaction.Create(resource); err != nil {
		return nil, fmt.Errorf("Error during gohan_db_create: %s", err.Error())
	}
	if needCommit {
		err = transaction.Commit()
		if err != nil {
			return nil, fmt.Errorf("Error during gohan_db_create: %s", err.Error())
		}
	}
	return resource, nil
}

//GohanDbUpdate updates resource in database
func GohanDbUpdate(transaction transaction.Transaction, needCommit bool, schemaID string,
	dataMap map[string]interface{}) (*schema.Resource, error) {

	manager := schema.GetManager()
	resource, err := manager.LoadResource(schemaID, dataMap)
	if err != nil {
		return nil, fmt.Errorf("Error during gohan_db_update: %s", err.Error())
	}
	if err = transaction.Update(resource); err != nil {
		return nil, fmt.Errorf("Error during gohan_db_update: %s", err.Error())
	}
	if needCommit {
		err = transaction.Commit()
		if err != nil {
			return nil, fmt.Errorf("Error during gohan_db_update: %s", err.Error())
		}
	}
	return resource, nil
}

//GohanDbStateUpdate updates resource's state in database
func GohanDbStateUpdate(transaction transaction.Transaction, needCommit bool, schemaID string,
	dataMap map[string]interface{}) (*schema.Resource, error) {

	manager := schema.GetManager()
	resource, err := manager.LoadResource(schemaID, dataMap)
	if err != nil {
		return nil, fmt.Errorf("Error during gohan_db_state_update: %s", err.Error())
	}
	if err = transaction.StateUpdate(resource, nil); err != nil {
		return nil, fmt.Errorf("Error during gohan_db_state_update: %s", err.Error())
	}
	if needCommit {
		err = transaction.Commit()
		if err != nil {
			return nil, fmt.Errorf("Error during gohan_db_state_update: %s", err.Error())
		}
	}
	return resource, nil
}

//GohanDbDelete deletes resource from database
func GohanDbDelete(transaction transaction.Transaction, needCommit bool, schemaID, ID string) error {
	schema, err := getSchema(schemaID)
	if err != nil {
		return fmt.Errorf("Error during gohan_db_delete: %s", err.Error())
	}
	if err := transaction.Delete(schema, ID); err != nil {
		return fmt.Errorf("Error during gohan_db_delete: %s", err.Error())
	}
	if needCommit {
		err := transaction.Commit()
		if err != nil {
			return fmt.Errorf("Error during gohan_db_delete: %s", err.Error())
		}
	}
	return nil
}

//GohanDbQuery get resources from database with query
func GohanDbQuery(transaction transaction.Transaction, needCommit bool, schemaID,
	sqlString string, arguments []interface{}) ([]map[string]interface{}, error) {

	schema, err := getSchema(schemaID)
	if err != nil {
		return []map[string]interface{}{}, err
	}
	resources, err := transaction.Query(schema, sqlString, arguments)
	if err != nil {
		return []map[string]interface{}{}, fmt.Errorf("Error during gohan_db_query: %s", err.Error())
	}
	if needCommit {
		err = transaction.Commit()
		if err != nil {
			return []map[string]interface{}{}, fmt.Errorf("Error during gohan_db_query: %s", err.Error())
		}
	}
	resp := []map[string]interface{}{}
	for _, resource := range resources {
		resp = append(resp, resource.Data())
	}
	return resp, nil
}

//GohanDbMakeColumns creates columns for given resource in database
func GohanDbMakeColumns(schemaID string) ([]string, error) {
	schema, err := getSchema(schemaID)
	if err != nil {
		return []string{}, err
	}
	results := sql.MakeColumns(schema, schema.GetDbTableName(), false)
	return results, nil
}
