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
	"path/filepath"
	"reflect"
	"strings"

	"github.com/robertkrimen/otto"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/dbutil"
	"github.com/cloudwan/gohan/db/options"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	gohan_otto "github.com/cloudwan/gohan/extension/otto"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync/noop"
	// Import otto underscore lib
	_ "github.com/robertkrimen/otto/underscore"
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
	schemaDir       string
	testFileName    string
	testSource      []byte
	dbConnection    db.DB
	dbTransactions  []transaction.Transaction
}

// NewEnvironment creates a new test environment based on provided DB connection
func NewEnvironment(testFileName string, testSource []byte) *Environment {
	env := &Environment{
		mockedFunctions: []string{},
		schemaDir:       filepath.Dir(testFileName),
		testFileName:    testFileName,
		testSource:      testSource,
	}
	return env
}

// InitializeEnvironment creates new transaction for the test
func (env *Environment) InitializeEnvironment() error {
	var err error

	env.dbConnection, err = newDBConnection(env.memoryDbConn())
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %s", err)
	}
	envName := strings.TrimSuffix(
		filepath.Base(env.testFileName),
		filepath.Ext(env.testFileName))
	env.Environment = gohan_otto.NewEnvironment(envName, env.dbConnection, &middleware.FakeIdentity{}, noop.NewSync())
	env.SetUp()
	env.addTestingAPI()

	env.Load(env.testFileName, string(env.testSource))

	err = env.loadSchemaIncludes()

	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to load schema includes for '%s': %s", env.testFileName, err)
	}

	err = env.loadSchemas()

	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to load schemas for '%s': %s", env.testFileName, err)
	}

	err = env.registerEnvironments()

	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to register environments for '%s': %s", env.testFileName, err)
	}

	err = env.loadExtensions()

	if err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to load extensions for '%s': %s", env.testFileName, err)
	}

	if err = dbutil.InitDBWithSchemas("sqlite3", env.memoryDbConn(), db.DefaultTestInitDBParams()); err != nil {
		schema.ClearManager()
		return fmt.Errorf("Failed to init DB: %s", err)
	}

	return nil
}

func (env *Environment) memoryDbConn() string {
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", env.testFileName)
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
	env.Environment.ClearEnvironment()
	schema.ClearManager()
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
	connection, err := dbutil.ConnectDB("sqlite3", dbfilename, db.DefaultMaxOpenConn, options.Default())
	if err != nil {
		return nil, err
	}
	return connection, nil
}

func (env *Environment) addTestingAPI() {
	builtins := map[string]interface{}{
		"Fail": func(call otto.FunctionCall) otto.Value {
			if len(call.ArgumentList) == 0 {
				panic(fmt.Errorf("Fail"))
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
			isolationLevel := transaction.RepeatableRead
			if len(call.ArgumentList) > 2 {
				panic("Wrong number of arguments in MockTransaction call.")
			}
			if len(call.ArgumentList) > 0 {
				rawNewTransaction, _ := call.Argument(0).Export()
				newTransaction = rawNewTransaction.(bool)
			}
			if len(call.ArgumentList) > 1 {
				isolationLevel = transaction.Type(call.Argument(1).String())
			}
			tx, err := env.getTransaction(newTransaction, isolationLevel)
			if err != nil {
				gohan_otto.ThrowOttoException(&call, err.Error())
			}
			transactionValue, _ := call.Otto.ToValue(tx)
			return transactionValue
		},
		"CommitMockTransaction": func(call otto.FunctionCall) otto.Value {
			tx, err := env.getTransaction(false, transaction.RepeatableRead)
			if err != nil {
				gohan_otto.ThrowOttoException(&call, err.Error())
			}
			tx.Commit()
			tx.Close()
			return otto.Value{}
		},
		"MockPolicy": func(call otto.FunctionCall) otto.Value {
			policyValue, _ := call.Otto.ToValue(schema.NewEmptyPolicy())
			return policyValue
		},
		"MockAuthorization": func(call otto.FunctionCall) otto.Value {
			authorizationValue, _ := call.Otto.ToValue(schema.NewAuthorizationBuilder().BuildScopedToTenant())
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
	env.mockFunction("gohan_exec")
	env.mockFunction("gohan_config")
	env.mockFunction("gohan_sync_fetch")
	env.mockFunction("gohan_sync_delete")
	env.mockFunction("gohan_sync_watch")
}

func (env *Environment) getTransaction(isNew bool, isolationLevel transaction.Type) (transaction.Transaction, error) {
	if !isNew {
		for _, tx := range env.dbTransactions {
			if !tx.Closed() {
				if tx.GetIsolationLevel() == isolationLevel {
					return tx, nil
				}
				return nil, fmt.Errorf("Requested %s isolation level, got %s", isolationLevel, tx.GetIsolationLevel())
			}
		}
	}
	tx, _ := env.dbConnection.BeginTx(transaction.IsolationLevel(isolationLevel))
	env.dbTransactions = append(env.dbTransactions, tx)
	return tx, nil
}

func getSpecifiedFunction(env *Environment, functionName string, setValue otto.Value) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		exceptions := env.getFromOtto(functionName, "exceptions").([]otto.Value)
		responses := env.getFromOtto(functionName, "responses").([]otto.Value)
		if len(call.ArgumentList) != 1 && setValue == otto.FalseValue() {
			call.Otto.Call("Fail", nil, "Return() should be called with exactly one argument")
		} else if len(call.ArgumentList) != 1 && setValue == otto.TrueValue() {
			call.Otto.Call("Fail", nil, "Throw() should be called with exactly one argument")
		}
		responses = append(responses, call.ArgumentList[0])
		exceptions = append(exceptions, setValue)
		env.setToOtto(functionName, "responses", responses)
		env.setToOtto(functionName, "exceptions", exceptions)
		return otto.NullValue()
	}
}

func (env *Environment) mockFunction(functionName string) {
	env.VM.Set(functionName, func(call otto.FunctionCall) otto.Value {
		responses := env.getFromOtto(functionName, "responses").([]otto.Value)
		requests := env.getFromOtto(functionName, "requests").([][]otto.Value)
		exceptions := env.getFromOtto(functionName, "exceptions").([]otto.Value)
		err := env.checkSpecified(functionName)
		if err != nil {
			call.Otto.Call("Fail", nil, err)
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
		isException := exceptions[0]
		exceptions = exceptions[1:]
		response := responses[0]
		responses = responses[1:]
		requests = requests[1:]
		env.setToOtto(functionName, "requests", requests)
		env.setToOtto(functionName, "exceptions", exceptions)
		env.setToOtto(functionName, "responses", responses)
		if isException == otto.TrueValue() {
			gohan_otto.ThrowOtto(&call, response.String())
		}
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
	env.setToOtto(functionName, "Return", getSpecifiedFunction(env, functionName, otto.FalseValue()))

	env.setToOtto(functionName, "exceptions", []otto.Value{})
	env.setToOtto(functionName, "Throw", getSpecifiedFunction(env, functionName, otto.TrueValue()))
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
		valA, err := a[i].Export()
		if err != nil {
			panic(fmt.Sprintf(
				"Error when exporting otto value for comparison %v", a[i]))
		}
		valB, err := b[i].Export()
		if err != nil {
			panic(fmt.Sprintf(
				"Error when exporting otto value for comparison %v", b[i]))
		}
		if !reflect.DeepEqual(valA, valB) {
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
		schemaPath := env.schemaPath(schemaIncludes)
		if data, err = ioutil.ReadFile(schemaPath); err != nil {
			return err
		}

		schemas := strings.Split(string(data), "\n")
		for _, schema := range schemas {
			if schema == "" || strings.HasPrefix(schema, "#") {
				continue
			}

			schemaPath := env.schemaPath(filepath.Dir(schemaIncludes), schema)
			if err = manager.LoadSchemaFromFile(schemaPath); err != nil {
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
		schemaPath := env.schemaPath(schema)
		if err = manager.LoadSchemaFromFile(schemaPath); err != nil {
			return err
		}
	}
	return nil
}

func (env *Environment) schemaPath(s ...string) string {
	s = append([]string{env.schemaDir}, s...)
	return filepath.Join(s...)
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

	return env.LoadExtensionsForPath(manager.Extensions, manager.TimeLimit, manager.TimeLimits, pathString)
}
