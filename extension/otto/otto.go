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
	"io"
	"net/http"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	ext "github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"

	"github.com/ddliu/motto"
	"github.com/xyproto/otto"

	"reflect"
	//Import otto underscore lib
	"regexp"

	_ "github.com/xyproto/otto/underscore"
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
	Name        string
	VM          *motto.Motto
	DataStore   db.DB
	timeLimit   time.Duration
	timeLimits  []*schema.EventTimeLimit
	Identity    middleware.IdentityService
	Sync        sync.Sync
	globalStore *GlobalStore
}

//NewEnvironment create new gohan extension environment based on context
func NewEnvironment(name string, dataStore db.DB, identity middleware.IdentityService, sync sync.Sync) *Environment {
	vm := motto.New()
	vm.Interrupt = make(chan func(), 1)
	env := &Environment{
		Name:        name,
		VM:          vm,
		DataStore:   dataStore,
		Identity:    identity,
		Sync:        sync,
		globalStore: NewGlobalStore(),
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
func (env *Environment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
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
	// setup time limits for matching extensions
	env.timeLimit = timeLimit
	for _, timeLimit := range timeLimits {
		if timeLimit.Match(path) {
			env.timeLimits = append(env.timeLimits, schema.NewEventTimeLimit(timeLimit.EventRegex, timeLimit.TimeDuration))
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
	var closeNotifier http.CloseNotifier
	var closeNotify <-chan bool
	if httpResponse, ok := context["http_response"]; ok {
		if closeNotifier, ok = httpResponse.(http.CloseNotifier); ok {
			closeNotify = closeNotifier.CloseNotify()
		}
	}
	context["event_type"] = event
	var timeout = fmt.Errorf("exceed timeout for extension execution for event: %s", event)
	var disconnected = fmt.Errorf("client disconnected for event: %s", event)

	defer func() {
		if caught := recover(); caught != nil {
			if caughtError, ok := caught.(error); ok {
				switch caughtError {
				case timeout, disconnected:
					err = caughtError
					log.Warning(caughtError.Error())
				default:
					panic(caught) // Something else happened, repanic!
				}
			}
		}
	}()
	// take time limit from first passing regex or default
	selectedTimeLimit := env.timeLimit
	for _, timeLimit := range env.timeLimits {
		if timeLimit.Match(event) {
			selectedTimeLimit = timeLimit.TimeDuration
			break
		}
	}
	timer := time.NewTimer(selectedTimeLimit)
	successCh := make(chan bool)
	go func() {
		for {
			select {
			case <-closeNotify:
				vm.Interrupt <- func() {
					panic(disconnected)
				}
				return
			case <-timer.C:
				vm.Interrupt <- func() {
					panic(timeout)
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
	defer func() {
		// cleanup Closers
		if closers, err := getClosers(vm.Otto); err == nil {
			log.Debug("Closers for vm %p: %v", vm.Otto, closers)
			for _, closer := range closers {
				if err = closer.Close(); err != nil {
					log.Error("Error when closing object %T %p : %s", closer, closer, err)
				}
			}
		} else {
			log.Warning(err.Error())
		}
	}()
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

func getClosers(vm *otto.Otto) (closers []io.Closer, err error) {
	closersValue, err := vm.Get("gohan_closers")
	if err != nil {
		log.Error("Error when getting env closers: %s", err)
		return
	}
	closersInterface, err := closersValue.Export()
	if err != nil {
		log.Error("Error when exporting closers: %s", err)
		return
	}
	closers, ok := closersInterface.([]io.Closer)
	if !ok {
		return nil, fmt.Errorf("Object %#v type %t is not []io.Closer", closersInterface, closersInterface)
	}
	return
}

func addCloser(vm *otto.Otto, closer io.Closer) error {
	log.Debug("Registering closer %p in VM %p", closer, vm);
	closers, err := getClosers(vm)
	if err != nil {
		return err
	}
	closers = append(closers, closer)
	// set to vm in case, slice has relocated
	vm.Set("gohan_closers", closers)
	return nil
}

// SetEventTimeLimit overrides the default time limit for a given event for this environment
func (env *Environment) SetEventTimeLimit(eventRegex string, timeLimit time.Duration) {
	env.timeLimits = append(env.timeLimits, schema.NewEventTimeLimit(regexp.MustCompile(eventRegex), timeLimit))
}

//Clone makes clone of the environment
func (env *Environment) Clone() ext.Environment {
	clone := NewEnvironment(env.Name, env.DataStore, env.Identity, env.Sync)
	clone.VM.Otto = env.VM.Copy()
	clone.VM.Otto.Interrupt = make(chan func(), 1)
	clone.timeLimit = env.timeLimit
	clone.timeLimits = env.timeLimits
	// workaround for original env being shared in builtin closures
	// need another fix for this race'y and unsafe behavior
	clone.VM.Otto.Set("gohan_closers", []io.Closer{})
	clone.globalStore = env.globalStore
	return clone
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

func (env *Environment) ClearEnvironment() {
	env.Sync.Close()
	env.DataStore.Close()
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
