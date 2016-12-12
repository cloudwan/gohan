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
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	ext "github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"

	"github.com/ddliu/motto"
	"github.com/robertkrimen/otto"

	"reflect"
	//Import otto underscore lib
	_ "github.com/robertkrimen/otto/underscore"
)

var inits = []func(env *Environment){}
var modules = map[string]interface{}{}

//RegisterModule registers modules
func RegisterModule(name string, module interface{}) {
	modules[name] = module
}

//RequireModule returns module
func RequireModule(name string) (interface{}, error) {
	v, ok := modules[name]
	if ok {
		return v, nil
	} else {
		return nil, fmt.Errorf("Module %s not found in Otto", name)
	}
}

//Environment javascript based environment for gohan extension
type Environment struct {
	Name      string
	VM        *motto.Motto
	DataStore db.DB
	timelimit time.Duration
	Identity  middleware.IdentityService
	Sync      sync.Sync
}

//NewEnvironment create new gohan extension environment based on context
func NewEnvironment(name string, dataStore db.DB, identity middleware.IdentityService,
                    timelimit time.Duration, sync sync.Sync) *Environment {
	vm := motto.New()
	vm.Interrupt = make(chan func(), 1)
	env := &Environment{
		Name:      name,
		VM:        vm,
		DataStore: dataStore,
		Identity:  identity,
		timelimit: timelimit,
		Sync:      sync,
	}
	env.SetUp()
	return env
}

//SetUp initialize environment
func (env *Environment) SetUp() {
	for _, init := range inits {
		init(env)
	}
}

//RegisterInit registers init code
func RegisterInit(init func(env *Environment)) {
	inits = append(inits, init)
}

//Load loads script for environment
func (env *Environment) Load(source, code string) error {
	vm := env.VM
	script, err := vm.Compile(source, code)
	if err != nil {
		return err
	}
	_, err = vm.Otto.Run(script)

	if err != nil {
		return err
	}
	return nil
}

//RegisterObject register new object for VM
func (env *Environment) RegisterObject(objectID string, object interface{}) {
	env.VM.Set(objectID, object)
}

//LoadExtensionsForPath loads extensions for specific path
func (env *Environment) LoadExtensionsForPath(extensions []*schema.Extension, path string) error {
	for _, extension := range extensions {
		if extension.Match(path) {
			code := extension.Code
			if extension.CodeType != "javascript" {
				continue
			}

			script, err := env.VM.Compile(extension.URL, code)
			if err != nil {
				return err
			}
			_, err = env.VM.Otto.Run(script)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func convertNilsToNulls(object interface{}) {
	switch object := object.(type) {
	case map[string]interface{}:
		for key, obj := range object {
			switch obj := obj.(type) {
			case map[string]interface{}:
				convertNilsToNulls(obj)
			case []interface{}:
				convertNilsToNulls(obj)
			case nil:
				object[key] = otto.NullValue()
			}
		}
	case []interface{}:
		for key, obj := range object {
			switch obj := obj.(type) {
			case map[string]interface{}:
				convertNilsToNulls(obj)
			case []interface{}:
				convertNilsToNulls(obj)
			case nil:
				object[key] = otto.NullValue()
			}
		}
	}
}

//HandleEvent handles event
func (env *Environment) HandleEvent(event string, context map[string]interface{}) (err error) {
	vm := env.VM
	context["event_type"] = event
	var halt = fmt.Errorf("exceed timeout for extension execution")

	defer func() {
		if caught := recover(); caught != nil {
			if caughtError, ok := caught.(error); ok {
				if caughtError.Error() == halt.Error() {
					log.Error(halt.Error())
					err = halt
					return
				}
			}

			panic(caught) // Something else happened, repanic!
		}
	}()
	timer := time.NewTimer(env.timelimit)
	successCh := make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
				vm.Interrupt <- func() {
					panic(halt)
				}
				return
			case <-successCh:
				// extension executed successfully
				return
			}
		}
	}()

	//FIXME(timorl): This is needed only because of a bug in Otto, where nils are converted to undefineds instead of nulls.
	convertNilsToNulls(context)
	contextInVM, err := vm.ToValue(context)
	if err != nil {
		return err
	}
	_, err = vm.Call("gohan_handle_event", nil, event, contextInVM)
	for key, value := range context {
		context[key] = ConvertOttoToGo(value)
	}
	if err != nil {
		switch err.(type) {
		case *otto.Error:
			ottoErr := err.(*otto.Error)
			err = fmt.Errorf("%s: %s", event, ottoErr.String())
		default:
			err = fmt.Errorf("%s: %s", event, err.Error())
		}
	}

	timer.Stop()
	successCh <- true
	if err != nil {
		return err
	}
	return err
}

//Clone makes clone of the environment
func (env *Environment) Clone() ext.Environment {
	newEnv := NewEnvironment(env.Name, env.DataStore, env.Identity, env.timelimit, env.Sync)
	newEnv.VM.Otto = env.VM.Copy()
	return newEnv
}

//GetOrCreateTransaction gets transaction from otto value or creates new is otto value is null
func (env *Environment) GetOrCreateTransaction(value otto.Value) (transaction.Transaction, bool, error) {
	if !value.IsNull() {
		tx, err := GetTransaction(value)
		return tx, false, err
	}
	dataStore := env.DataStore
	tx, err := dataStore.Begin()
	if err != nil {
		return nil, false, fmt.Errorf("Error creating transaction: %v", err.Error())
	}
	return tx, true, nil
}

func (env *Environment) ClearEnvironment()  {
	env.Sync.Close()
}

func throwOtto(call *otto.FunctionCall, exceptionName string, arguments ...interface{}) {
	exception, _ := call.Otto.Call("new "+exceptionName, nil, arguments...)
	panic(exception)
}

//ThrowOttoException throws a JavaScript exception that will be passed to Otto
func ThrowOttoException(call *otto.FunctionCall, format string, arguments ...interface{}) {
	throwOtto(call, "Error", fmt.Sprintf(format, arguments...))
}

//VerifyCallArguments verify number of calles
func VerifyCallArguments(call *otto.FunctionCall, functionName string, expectedArgumentsCount int) {
	if len(call.ArgumentList) != expectedArgumentsCount {
		ThrowOttoException(call, "Expected %d arguments in %s call, %d arguments given",
			expectedArgumentsCount, functionName, len(call.ArgumentList))
	}
}

const wrongArgumentType string = "Argument '%v' should be of type '%s'"

//GetString gets string from otto value
func GetString(value otto.Value) (string, error) {
	rawString, _ := value.Export()
	result, ok := rawString.(string)
	if !ok {
		return "", fmt.Errorf(wrongArgumentType, rawString, "string")
	}
	return result, nil
}

//GetMap gets map[string]interface{} from otto value
func GetMap(value otto.Value) (map[string]interface{}, error) {
	rawMap, _ := value.Export()
	result, ok := rawMap.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}, fmt.Errorf(wrongArgumentType, rawMap, "Object")
	}
	for key, value := range result {
		result[key] = ConvertOttoToGo(value)
	}
	return result, nil
}

//GetTransaction gets Transaction from otto value
func GetTransaction(value otto.Value) (transaction.Transaction, error) {
	rawTransaction, _ := value.Export()
	result, ok := rawTransaction.(transaction.Transaction)
	if !ok {
		return nil, fmt.Errorf(wrongArgumentType, rawTransaction, "Transaction")
	}
	return result, nil
}

//GetAuthorization gets Transaction from otto value
func GetAuthorization(value otto.Value) (schema.Authorization, error) {
	rawAuthorization, _ := value.Export()
	result, ok := rawAuthorization.(schema.Authorization)
	if !ok {
		return nil, fmt.Errorf(wrongArgumentType, rawAuthorization, "Authorization")
	}
	return result, nil
}

//GetBool gets bool from otto value
func GetBool(value otto.Value) (bool, error) {
	rawBool, _ := value.Export()
	result, ok := rawBool.(bool)
	if !ok {
		return false, fmt.Errorf(wrongArgumentType, rawBool, "bool")
	}
	return result, nil
}

//GetList gets []interface{} from otto value
func GetList(value otto.Value) ([]interface{}, error) {
	rawSlice, err := value.Export()
	result := make([]interface{}, 0)
	if rawSlice == nil || err != nil {
		return result, err
	}
	typeOfSlice := reflect.TypeOf(rawSlice)
	if typeOfSlice.Kind() != reflect.Array && typeOfSlice.Kind() != reflect.Slice {
		return result, fmt.Errorf(wrongArgumentType, value, "array")
	}
	list := reflect.ValueOf(rawSlice)
	for i := 0; i < list.Len(); i++ {
		result = append(result, ConvertOttoToGo(list.Index(i).Interface()))
	}

	return result, err
}

//GetStringList gets []string  from otto value
func GetStringList(value otto.Value) ([]string, error) {
	var ok bool
	var rawSlice []interface{}
	var stringSlice []string

	rawData, _ := value.Export()
	rawSlice, ok = rawData.([]interface{})

	if ok && len(rawSlice) == 0 {
		return []string{}, nil
	}

	stringSlice, ok = rawData.([]string)

	if !ok {
		return make([]string, 0), fmt.Errorf(wrongArgumentType, rawData, "array of strings")
	}

	return stringSlice, nil
}

//GetInt64 gets int64 from otto value
func GetInt64(value otto.Value) (result int64, err error) {
	result, err = value.ToInteger()
	if err != nil {
		err = fmt.Errorf(wrongArgumentType, value, "int64")
	}
	return
}

//ConvertOttoToGo ...
func ConvertOttoToGo(value interface{}) interface{} {
	ottoValue, ok := value.(otto.Value)
	if ok {
		exportedValue, err := ottoValue.Export()
		if err != nil {
			return err
		}
		return ConvertOttoToGo(exportedValue)
	}
	mapValue, ok := value.(map[string]interface{})
	if ok {
		for key, value := range mapValue {
			mapValue[key] = ConvertOttoToGo(value)
		}
		return mapValue
	}
	listValue, ok := value.([]interface{})
	if ok {
		for key, value := range listValue {
			listValue[key] = ConvertOttoToGo(value)
		}
		return listValue
	}
	return value
}

func getSchema(schemaID string) (*schema.Schema, error) {
	manager := schema.GetManager()
	schema, ok := manager.Schema(schemaID)
	if !ok {
		return nil, fmt.Errorf(unknownSchemaErrorMesssageFormat, schemaID)
	}
	return schema, nil
}
