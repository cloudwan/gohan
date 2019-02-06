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

	"github.com/robertkrimen/otto"

	"context"

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
				maxArgs := 1
				setTxIsolationLevel := len(call.ArgumentList) > 0

				if len(call.ArgumentList) > maxArgs {
					ThrowOttoException(&call,
						"Expected no more than %d arguments in %s call, %d arguments given",
						maxArgs, "gohan_db_transaction", len(call.ArgumentList))
				}

				var tx transaction.Transaction
				var err error

				opts := []transaction.Option{}

				if setTxIsolationLevel {
					strIsolationLevel, err := GetString(call.Argument(0))
					if err != nil {
						ThrowOttoException(&call, err.Error())
					}
					opts = append(opts, transaction.IsolationLevel(transaction.Type(strIsolationLevel)))
				}

				tx, err = env.DataStore.BeginTx(opts...)

				if err != nil {
					ThrowOttoException(&call, "failed to start a transaction: %s", err.Error())
				}

				if err = addCloser(call.Otto, tx); err != nil {
					tx.Close()
					ThrowOttoException(&call, fmt.Errorf(
						"Cannot register closer for gohan_db_transaction: %s", err).Error())
				}
				value, _ := vm.ToValue(tx)
				return value
			},
			"gohan_db_list": func(call otto.FunctionCall) otto.Value {
				schema, filter, pg, transaction, needCommit, err := ottoListArgumentsHelper(&call, env)
				if err != nil {
					ThrowOttoException(&call, "Error during ottoListArgumentsHelper: %s", err.Error())
				}

				if needCommit {
					defer transaction.Close()
				}

				resources, _, err := transaction.List(context.Background(), schema, filter, nil, pg)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_list: %s", err.Error())
				}

				resp := parseListResults(resources)
				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_db_lock_list": func(call otto.FunctionCall) otto.Value {
				schema, filter, pg, transaction, needCommit, lockPolicy, err := ottoLockListArgumentsHelper(&call, env)
				if err != nil {
					ThrowOttoException(&call, "Error during ottoLockListArgumentsHelper: %s", err.Error())
				}

				if needCommit {
					defer transaction.Close()
				}

				resources, _, err := transaction.LockList(context.Background(), schema, filter, nil, pg, lockPolicy)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_lock_list: %s", err.Error())
				}

				resp := parseListResults(resources)
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
					panic("Fetch failed: no data returned but err is nil")
				}
				value, _ := vm.ToValue(resp.Data())
				return value
			},
			"gohan_db_lock_fetch": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_lock_fetch", 5)
				tx, needCommit, err := env.GetOrCreateTransaction(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				if needCommit {
					defer tx.Close()
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
				rawLockPolicy, err := GetInt64(call.Argument(4))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				lockPolicy := schema.LockPolicy(rawLockPolicy)

				resp, err := GohanDbLockFetch(tx, schemaID, ID, tenantID, lockPolicy)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				if resp == nil {
					panic("LockFetch failed: no data returned but err is nil")
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

func ottoLockListArgumentsHelper(call *otto.FunctionCall, env *Environment) (sh *schema.Schema, filter map[string]interface{}, pg *pagination.Paginator, tx transaction.Transaction, needCommit bool, lockPolicy schema.LockPolicy, err error) {
	sh, filter, pg, tx, needCommit, err = ottoListArgumentsHelper(call, env)

	if len(call.ArgumentList) < 7 {
		lockPolicy = schema.SkipRelatedResources
	} else {
		var rawLockPolicy int64
		rawLockPolicy, err = GetInt64(call.Argument(6))
		if err != nil {
			return
		}
		lockPolicy = schema.LockPolicy(rawLockPolicy)
	}
	return
}

func ottoListArgumentsHelper(call *otto.FunctionCall, env *Environment) (
	schema *schema.Schema,
	filter map[string]interface{},
	pg *pagination.Paginator,
	tx transaction.Transaction,
	needCommit bool,
	err error) {

	opts := []pagination.OptionPaginator{}

	if len(call.ArgumentList) < 3 {
		ThrowOttoException(call, "Error: not enough arguments for gohana_db_list")
	}

	schemaID, err := GetString(call.Argument(1))
	if err != nil {
		return
	}

	schema, err = getSchema(schemaID)
	if err != nil {
		return
	}

	filter, err = GetMap(call.Argument(2))
	if err != nil {
		return
	}

	if len(call.ArgumentList) > 3 {
		var orderKey string
		orderKey, err = GetString(call.Argument(3))
		if err != nil {
			return
		}
		opts = append(opts, pagination.OptionKey(schema, orderKey))
	}

	if len(call.ArgumentList) > 4 {
		var rawLimit int64
		rawLimit, err = GetInt64(call.Argument(4))
		if err != nil {
			return
		}
		limit := uint64(rawLimit)
		opts = append(opts, pagination.OptionLimit(limit))
	}

	if len(call.ArgumentList) > 5 {
		var rawOffset int64
		rawOffset, err = GetInt64(call.Argument(5))
		if err != nil {
			return
		}
		offset := uint64(rawOffset)
		opts = append(opts, pagination.OptionOffset(offset))
	}

	opts = append(opts, pagination.OptionOrder(pagination.ASC)) // To match previous implementation based on mySql default
	pg, err = pagination.NewPaginator(opts...)
	if err != nil {
		return
	}

	tx, needCommit, err = env.GetOrCreateTransaction(call.Argument(0))
	if err != nil {
		return
	}

	return schema, filter, pg, tx, needCommit, err
}

func parseListResults(resources []*schema.Resource) []map[string]interface{} {
	resp := make([]map[string]interface{}, len(resources))
	for i, resource := range resources {
		resp[i] = resource.Data()
	}
	return resp
}

//GohanDbFetch gets resource from database
func GohanDbFetch(tx transaction.Transaction, schemaID, ID,
	tenantID string) (*schema.Resource, error) {

	schema, err := getSchema(schemaID)
	if err != nil {
		return nil, err
	}
	filter := transaction.IDFilter(ID)
	if tenantID != "" {
		filter["tenant_id"] = tenantID
	}
	resp, err := tx.Fetch(context.Background(), schema, filter, nil)
	if err != nil {
		return nil, fmt.Errorf("Error during gohan_db_fetch: %s", err.Error())
	}
	return resp, nil
}

//GohanDbLockFetch gets resource from database
func GohanDbLockFetch(tx transaction.Transaction, schemaID, ID, tenantID string, policy schema.LockPolicy) (*schema.Resource, error) {
	schema, err := getSchema(schemaID)
	if err != nil {
		return nil, err
	}
	filter := transaction.IDFilter(ID)
	if tenantID != "" {
		filter["tenant_id"] = tenantID
	}
	resp, err := tx.LockFetch(context.Background(), schema, filter, policy, nil)
	if err != nil {
		return nil, fmt.Errorf("Error during gohan_db_lock_fetch: %s", err.Error())
	}
	return resp, nil
}

//GohanDbStateFetch gets resource's state from database
func GohanDbStateFetch(tx transaction.Transaction, schemaID, ID,
	tenantID string) (map[string]interface{}, error) {

	schema, err := getSchema(schemaID)
	if err != nil {
		return map[string]interface{}{}, err
	}
	filter := transaction.IDFilter(ID)
	if tenantID != "" {
		filter["tenant_id"] = tenantID
	}
	resp, err := tx.StateFetch(context.Background(), schema, filter)
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
	if _, err = transaction.Create(context.Background(), resource); err != nil {
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
	if err = transaction.Update(context.Background(), resource); err != nil {
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
	if err = transaction.StateUpdate(context.Background(), resource, nil); err != nil {
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
	if err := transaction.Delete(context.Background(), schema, ID); err != nil {
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
	resources, err := transaction.Query(context.Background(), schema, sqlString, arguments)
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
	results := sql.MakeColumns(schema, schema.GetDbTableName(), nil, false)
	return results, nil
}
