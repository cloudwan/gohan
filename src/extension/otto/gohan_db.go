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
	"github.com/dop251/otto"

	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
)

func init() {
	gohanDBInit := func(env *Environment) {
		vm := env.VM
		builtins := map[string]interface{}{
			"gohan_db_list": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_list", 3)
				rawTransaction, _ := call.Argument(0).Export()
				transaction, ok := rawTransaction.(transaction.Transaction)
				var err error
				if !ok {
					dataStore := env.DataStore
					transaction, err = dataStore.Begin()
					if err != nil {
						ThrowOttoException(&call, noTransactionErrorMessage)
					}
					defer transaction.Close()
				}
				schemaID := call.Argument(1).String()
				filterObj, _ := call.Argument(2).Export()
				var filter map[string]interface{}
				if filterObj != nil {
					filter = filterObj.(map[string]interface{})
				}
				manager := schema.GetManager()
				schema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				resources, _, err := transaction.List(schema, filter, nil)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_list: %s", err.Error())
				}
				resp := []map[string]interface{}{}
				for _, resource := range resources {
					resp = append(resp, resource.Data())
				}
				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_db_fetch": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_fetch", 4)
				rawTransaction, _ := call.Argument(0).Export()
				transaction, ok := rawTransaction.(transaction.Transaction)
				var err error
				if !ok {
					dataStore := env.DataStore
					transaction, err = dataStore.Begin()
					if err != nil {
						ThrowOttoException(&call, noTransactionErrorMessage)
					}
					defer transaction.Close()
				}
				schemaID := call.Argument(1).String()
				ID := call.Argument(2).String()
				tenantID := call.Argument(3).String()
				manager := schema.GetManager()
				schema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				var tenantFilter []string
				if tenantID != "" {
					tenantFilter = []string{tenantID}
				}
				resp, err := transaction.Fetch(schema, ID, tenantFilter)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_fetch: %s", err.Error())
				}
				if resp == nil {
					otto.NullValue()
				}
				value, _ := vm.ToValue(resp.Data())
				return value
			},
			"gohan_db_state_fetch": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_state_fetch", 4)
				rawTransaction, _ := call.Argument(0).Export()
				transaction, ok := rawTransaction.(transaction.Transaction)
				if !ok {
					ThrowOttoException(&call, noTransactionErrorMessage)
				}
				schemaID := call.Argument(1).String()
				ID := call.Argument(2).String()
				tenantID := call.Argument(3).String()
				manager := schema.GetManager()
				schema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				var tenantFilter []string
				if tenantID != "" {
					tenantFilter = []string{tenantID}
				}
				resp, err := transaction.StateFetch(schema, ID, tenantFilter)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_state_fetch: %s", err.Error())
				}
				data := map[string]interface{}{
					"config_version": resp.ConfigVersion,
					"state_version":  resp.StateVersion,
					"error":          resp.Error,
					"state":          resp.State,
					"monitoring":     resp.Monitoring,
				}
				value, _ := vm.ToValue(data)
				return value
			},
			"gohan_db_create": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_create", 3)
				rawTransaction, _ := call.Argument(0).Export()
				transaction, ok := rawTransaction.(transaction.Transaction)
				needCommit := false
				var err error
				if !ok {
					dataStore := env.DataStore
					transaction, err = dataStore.Begin()
					needCommit = true
					if err != nil {
						ThrowOttoException(&call, noTransactionErrorMessage)
					}
					defer transaction.Close()
				}
				schemaID := call.Argument(1).String()
				data := ConvertOttoToGo(call.Argument(2))
				dataMap, _ := data.(map[string]interface{})
				manager := schema.GetManager()
				resource, err := manager.LoadResource(schemaID, dataMap)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_create: %s", err.Error())
				}
				if err = transaction.Create(resource); err != nil {
					ThrowOttoException(&call, "Error during gohan_db_create: %s", err.Error())
				}
				if needCommit {
					err = transaction.Commit()
					if err != nil {
						ThrowOttoException(&call, "Error during gohan_db_create: %s", err.Error())
					}
				}
				value, _ := vm.ToValue(resource.Data())
				return value
			},
			"gohan_db_update": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_update", 3)
				rawTransaction, _ := call.Argument(0).Export()
				transaction, ok := rawTransaction.(transaction.Transaction)
				needCommit := false
				var err error
				if !ok {
					dataStore := env.DataStore
					transaction, err = dataStore.Begin()
					needCommit = true
					if err != nil {
						ThrowOttoException(&call, noTransactionErrorMessage)
					}
					defer transaction.Close()
				}
				schemaID := call.Argument(1).String()
				data := ConvertOttoToGo(call.Argument(2))
				dataMap, _ := data.(map[string]interface{})
				manager := schema.GetManager()
				resource, err := manager.LoadResource(schemaID, dataMap)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_update: %s", err.Error())
				}
				if err = transaction.Update(resource); err != nil {
					ThrowOttoException(&call, "Error during gohan_db_update: %s", err.Error())
				}
				if needCommit {
					err = transaction.Commit()
					if err != nil {
						ThrowOttoException(&call, "Error during gohan_db_create: %s", err.Error())
					}
				}
				value, _ := vm.ToValue(resource.Data())
				return value
			},
			"gohan_db_state_update": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_state_update", 3)
				rawTransaction, _ := call.Argument(0).Export()
				transaction, ok := rawTransaction.(transaction.Transaction)
				needCommit := false
				var err error
				if !ok {
					dataStore := env.DataStore
					transaction, err = dataStore.Begin()
					needCommit = true
					if err != nil {
						ThrowOttoException(&call, noTransactionErrorMessage)
					}
					defer transaction.Close()
				}
				schemaID := call.Argument(1).String()
				data := ConvertOttoToGo(call.Argument(2))
				dataMap, _ := data.(map[string]interface{})
				manager := schema.GetManager()
				resource, err := manager.LoadResource(schemaID, dataMap)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_state_update: %s", err.Error())
				}
				if err = transaction.StateUpdate(resource, nil); err != nil {
					ThrowOttoException(&call, "Error during gohan_db_state_update: %s", err.Error())
				}
				if needCommit {
					err = transaction.Commit()
					if err != nil {
						ThrowOttoException(&call, "Error during gohan_db_create: %s", err.Error())
					}
				}
				value, _ := vm.ToValue(resource.Data())
				return value
			},
			"gohan_db_delete": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_delete", 3)
				rawTransaction, _ := call.Argument(0).Export()
				transaction, ok := rawTransaction.(transaction.Transaction)
				needCommit := false
				var err error
				if !ok {
					dataStore := env.DataStore
					transaction, err = dataStore.Begin()
					needCommit = true
					if err != nil {
						ThrowOttoException(&call, noTransactionErrorMessage)
					}
					defer transaction.Close()
				}
				schemaID := call.Argument(1).String()
				ID := call.Argument(2).String()
				manager := schema.GetManager()
				schema, _ := manager.Schema(schemaID)
				if err := transaction.Delete(schema, ID); err != nil {
					ThrowOttoException(&call, "Error during gohan_db_delete: %s", err.Error())
				}
				if needCommit {
					err = transaction.Commit()
					if err != nil {
						ThrowOttoException(&call, "Error during gohan_db_create: %s", err.Error())
					}
				}
				return otto.NullValue()
			},
			"gohan_db_query": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_query", 4)
				rawTransaction, _ := call.Argument(0).Export()
				transaction, ok := rawTransaction.(transaction.Transaction)
				needCommit := false
				var err error
				if !ok {
					dataStore := env.DataStore
					transaction, err = dataStore.Begin()
					needCommit = true
					if err != nil {
						ThrowOttoException(&call, noTransactionErrorMessage)
					}
					defer transaction.Close()
				}
				schemaID := call.Argument(1).String()
				sqlString := call.Argument(2).String()
				rawArguments, _ := call.Argument(3).Export()

				manager := schema.GetManager()
				s, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}

				arguments, ok := rawArguments.([]interface{})
				if !ok {
					ThrowOttoException(&call, "Gievn arguments is not []interface{}")
				}

				resources, err := transaction.Query(s, sqlString, arguments)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_db_query: %s", err.Error())
				}
				if needCommit {
					err = transaction.Commit()
					if err != nil {
						ThrowOttoException(&call, "Error during gohan_db_create: %s", err.Error())
					}
				}
				resp := []map[string]interface{}{}
				for _, resource := range resources {
					resp = append(resp, resource.Data())
				}
				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_db_sql_make_columns": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_db_query", 1)
				schemaID := call.Argument(0).String()
				manager := schema.GetManager()
				s, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				results := sql.MakeColumns(s, false)
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
