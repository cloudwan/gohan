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

package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync/etcdv3"
	"github.com/robertkrimen/otto"

	//Import otto underscore lib
	_ "github.com/robertkrimen/otto/underscore"

	gohan_otto "github.com/cloudwan/gohan/extension/otto"
)

const (
	pathVar           = "PATH"
	schemasVar        = "SCHEMAS"
	schemaIncludesVar = "SCHEMA_INCLUDES"
)

// Environment of a single test runner
type Environment struct {
	*gohan_otto.Environment
	mockedFunctions []string
	testFileName    string
	testSource      []byte
	dbFile          *os.File
	dbConnection    db.DB
	dbTransactions  []transaction.Transaction
}

// NewEnvironment creates a new test environment based on provided DB connection
func NewEnvironment(testFileName string, testSource []byte) *Environment {
	env := &Environment{
		mockedFunctions: []string{},
		testFileName:    testFileName,
		testSource:      testSource,
	}
	return env
}

// InitializeEnvironment creates new transaction for the test
func (env *Environment) InitializeEnvironment() error {
	var err error

	_, file := filepath.Split(env.testFileName)
	env.dbFile, err = ioutil.TempFile(os.TempDir(), file)
	if err != nil {
		return fmt.Errorf("Failed to create a temporary file in %s: %s", os.TempDir(), err.Error())
	}
	env.dbConnection, err = newDBConnection(env.dbFile.Name())
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %s", err.Error())
	}
	endpoints := []string{"localhost:2379"}
	etcd, err := etcdv3.NewSync(endpoints, time.Second)
	if err != nil {
		return fmt.Errorf("Failed to connect to etcd: %s", err.Error())
	}
	envName := strings.TrimSuffix(
		filepath.Base(env.testFileName),
		filepath.Ext(env.testFileName))
	env.Environment = gohan_otto.NewEnvironment(envName, env.dbConnection,
							&middleware.FakeIdentity{}, 30 * time.Second, etcd)
	env.SetUp()
	env.addTestingAPI()

	script, err := env.VM.Otto.Compile(env.testFileName, env.testSource)
	if err != nil {
		return fmt.Errorf("Failed to compile the file '%s': %s", env.testFileName, err.Error())
	}

	env.VM.Otto.Run(script)

	err = env.loadSchemaIncludes()

	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to load schema includes for '%s': %s", env.testFileName, err.Error())
	}

	err = env.loadSchemas()

	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to load schemas for '%s': %s", env.testFileName, err.Error())
	}

	err = env.registerEnvironments()

	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to register environments for '%s': %s", env.testFileName, err.Error())
	}

	err = env.loadExtensions()

	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to load extensions for '%s': %s", env.testFileName, err.Error())
	}

	err = db.InitDBWithSchemas("sqlite3", env.dbFile.Name(), true, false)
	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to init DB: %s", err.Error())
	}

	return nil
}

// ClearEnvironment clears mock calls between tests and rollbacks test transaction
func (env *Environment) ClearEnvironment() {
	for _, functionName := range env.mockedFunctions {
		env.setToOtto(functionName, "requests", [][]otto.Value{})
		env.setToOtto(functionName, "responses", []otto.Value{})
	}

	for _, tx := range env.dbTransactions {
		tx.Close()
	}
	toDelete := env.dbFile.Name()
	env.dbFile.Close()
	os.Remove(toDelete)
	env.Sync.Close()
	schema.ClearManager()
	env.Environment.ClearEnvironment()
}

// CheckAllMockCallsMade check if all declared mock calls were made
func (env *Environment) CheckAllMockCallsMade() error {
	for _, functionName := range env.mockedFunctions {
		requests := env.getFromOtto(functionName, "requests").([][]otto.Value)
		responses := env.getFromOtto(functionName, "responses").([]otto.Value)
		if len(requests) > 0 || len(responses) > 0 {
			err := env.checkSpecified(functionName)
			if err != nil {
				return err
			}
			return fmt.Errorf("Expected call to %s(%v) with return value %v, but not made",
				functionName, valueSliceToString(requests[0]), responses[0])
		}
	}
	return nil
}

func newDBConnection(dbfilename string) (db.DB, error) {
	connection, err := db.ConnectDB("sqlite3", dbfilename, db.DefaultMaxOpenConn)
	if err != nil {
		return nil, err
	}
	return connection, nil
}

func (env *Environment) addTestingAPI() {
	builtins := map[string]interface{}{
		"Fail": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) == 0 {
				panic(fmt.Errorf("Fail!"))
			}

			if !call.ArgumentList[0].IsString() {
				panic(fmt.Errorf("Invalid call to 'Fail': format string expected first"))
			}

			format, _ := call.ArgumentList[0].ToString()
			args := []interface{}{}
			for _, value := range call.ArgumentList[1:] {
				args = append(args, gohan_otto.ConvertOttoToGo(value))
			}

			panic(fmt.Errorf(format, args...))
		},
		"MockTransaction": func(call otto.FunctionCall) otto.Value {
			newTransaction := false
			if len(call.ArgumentList) > 1 {
				panic("Wrong number of arguments in MockTransaction call.")
			} else if len(call.ArgumentList) == 1 {
				rawNewTransaction, _ := call.Argument(0).Export()
				newTransaction = rawNewTransaction.(bool)
			}
			transactionValue, _ := call.Otto.ToValue(env.getTransaction(newTransaction))
			return transactionValue
		},
		"CommitMockTransaction": func(call otto.FunctionCall) otto.Value {
			tx := env.getTransaction(false)
			tx.Commit()
			tx.Close()
			return otto.Value{}
		},
		"MockPolicy": func(call otto.FunctionCall) otto.Value {
			policyValue, _ := call.Otto.ToValue(schema.NewEmptyPolicy())
			return policyValue
		},
		"MockAuthorization": func(call otto.FunctionCall) otto.Value {
			authorizationValue, _ := call.Otto.ToValue(schema.NewAuthorization("", "", "", []string{}, []*schema.Catalog{}))
			return authorizationValue
		},
	}
	for name, object := range builtins {
		env.VM.Set(name, object)
	}
	// NOTE: There is no way to return error back to Otto after calling a Go
	// function, so the following function has to be written in pure JavaScript.
	env.VM.Otto.Run(`function GohanTrigger(event, context) { gohan_handle_event(event, context); }`)
	env.mockFunction("gohan_http")
	env.mockFunction("gohan_raw_http")
	env.mockFunction("gohan_db_transaction")
	env.mockFunction("gohan_config")
	env.mockFunction("gohan_sync_fetch")
	env.mockFunction("gohan_sync_watch")
}

func (env *Environment) getTransaction(isNew bool) transaction.Transaction {
	if !isNew {
		for _, tx := range env.dbTransactions {
			if !tx.Closed() {
				return tx
			}
		}
	}
	tx, _ := env.dbConnection.Begin()
	env.dbTransactions = append(env.dbTransactions, tx)
	return tx
}

func (env *Environment) mockFunction(functionName string) {
	env.VM.Set(functionName, func(call otto.FunctionCall) otto.Value {
		responses := env.getFromOtto(functionName, "responses").([]otto.Value)
		requests := env.getFromOtto(functionName, "requests").([][]otto.Value)

		err := env.checkSpecified(functionName)
		if err != nil {
			call.Otto.Call("Fail", nil, err.Error())
		}

		readableArguments := valueSliceToString(call.ArgumentList)

		if len(responses) == 0 {
			call.Otto.Call("Fail", nil, fmt.Sprintf("Unexpected call to %s(%v)", functionName, readableArguments))
		}

		expectedArguments := requests[0]
		actualArguments := call.ArgumentList
		if !argumentsEqual(expectedArguments, actualArguments) {
			call.Otto.Call("Fail", nil, fmt.Sprintf("Wrong arguments for call %s(%v), expected %s",
				functionName, readableArguments, valueSliceToString(expectedArguments)))
		}

		response := responses[0]
		responses = responses[1:]
		env.setToOtto(functionName, "responses", responses)

		requests = requests[1:]
		env.setToOtto(functionName, "requests", requests)

		return response
	})

	env.setToOtto(functionName, "requests", [][]otto.Value{})
	env.setToOtto(functionName, "Expect", func(call otto.FunctionCall) otto.Value {
		requests := env.getFromOtto(functionName, "requests").([][]otto.Value)
		requests = append(requests, call.ArgumentList)
		env.setToOtto(functionName, "requests", requests)

		function, _ := env.VM.Get(functionName)
		return function
	})

	env.setToOtto(functionName, "responses", []otto.Value{})
	env.setToOtto(functionName, "Return", func(call otto.FunctionCall) otto.Value {
		responses := env.getFromOtto(functionName, "responses").([]otto.Value)
		if len(call.ArgumentList) != 1 {
			call.Otto.Call("Fail", nil, "Return() should be called with exactly one argument")
		}
		responses = append(responses, call.ArgumentList[0])
		env.setToOtto(functionName, "responses", responses)

		return otto.NullValue()
	})
	env.mockedFunctions = append(env.mockedFunctions, functionName)
}

func (env *Environment) checkSpecified(functionName string) error {
	responses := env.getFromOtto(functionName, "responses").([]otto.Value)
	requests := env.getFromOtto(functionName, "requests").([][]otto.Value)
	if len(requests) > len(responses) {
		return fmt.Errorf("Return() should be specified for each call to %s", functionName)
	} else if len(requests) < len(responses) {
		return fmt.Errorf("Expect() should be specified for each call to %s", functionName)
	}
	return nil
}

func (env *Environment) getFromOtto(sourceFunctionName, variableName string) interface{} {
	sourceFunction, _ := env.VM.Get(sourceFunctionName)
	variableRaw, _ := sourceFunction.Object().Get(variableName)
	exportedVariable, _ := variableRaw.Export()
	return exportedVariable
}

func (env *Environment) setToOtto(destinationFunctionName, variableName string, variableValue interface{}) {
	destinationFunction, _ := env.VM.Get(destinationFunctionName)
	destinationFunction.Object().Set(variableName, variableValue)
}

func argumentsEqual(a, b []otto.Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func valueSliceToString(input []otto.Value) string {
	values := make([]string, len(input))
	for i, v := range input {
		values[i] = fmt.Sprintf("%v", gohan_otto.ConvertOttoToGo(v))
	}
	return "[" + strings.Join(values, ", ") + "]"
}

func (env *Environment) loadSchemaIncludes() error {
	manager := schema.GetManager()
	schemaIncludeValue, err := env.VM.Get(schemaIncludesVar)
	if err != nil {
		return fmt.Errorf("%s string array not specified", schemaIncludesVar)
	}
	schemaIncludesFilenames, err := gohan_otto.GetStringList(schemaIncludeValue)
	if err != nil {
		return fmt.Errorf("Bad type of %s - expected an array of strings but the type is %s",
			schemaIncludesVar, schemaIncludeValue.Class())
	}
	for _, schemaIncludes := range schemaIncludesFilenames {
		var data []byte

		if data, err = ioutil.ReadFile(schemaIncludes); err != nil {
			return err
		}

		schemas := strings.Split(string(data), "\n")

		for _, schema := range schemas {
			if schema == "" || strings.HasPrefix(schema, "#") {
				continue
			}

			if err = manager.LoadSchemaFromFile(schema); err != nil {
				return err
			}
		}
	}
	return nil
}

func (env *Environment) loadSchemas() error {
	schemaValue, err := env.VM.Get(schemasVar)
	if err != nil {
		return fmt.Errorf("%s string array not specified", schemasVar)
	}
	schemaFilenames, err := gohan_otto.GetStringList(schemaValue)
	if err != nil {
		return fmt.Errorf("Bad type of %s - expected an array of strings but the type is %s",
			schemasVar, schemaValue.Class())
	}

	manager := schema.GetManager()
	for _, schema := range schemaFilenames {
		if err = manager.LoadSchemaFromFile(schema); err != nil {
			return err
		}
	}
	return nil
}

func (env *Environment) registerEnvironments() error {
	manager := schema.GetManager()
	environmentManager := extension.GetManager()
	for schemaID := range manager.Schemas() {
		// Note: the following code ignores errors related to registration
		//       of an environment that has already been registered
		environmentManager.RegisterEnvironment(schemaID, env)
	}
	return nil
}

func (env *Environment) loadExtensions() error {
	manager := schema.GetManager()
	pathValue, err := env.VM.Get(pathVar)
	if err != nil || !pathValue.IsString() {
		return fmt.Errorf("%s string not specified", pathVar)
	}
	pathString, _ := pathValue.ToString()

	return env.LoadExtensionsForPath(manager.Extensions, pathString)
}
