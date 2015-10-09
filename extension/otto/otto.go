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

	"github.com/cloudwan/gohan/db"
	ext "github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"

	"github.com/dop251/otto"

	//Import otto underscore lib
	_ "github.com/dop251/otto/underscore"
)

var inits = []func(env *Environment){}

//GoCallback is type for go based callback

//Environment javascript based environment for gohan extension
type Environment struct {
	VM          *otto.Otto
	goCallbacks []ext.GoCallback
	DataStore   db.DB
	Identity    middleware.IdentityService
}

//NewEnvironment create new gohan extension environment based on context
func NewEnvironment(dataStore db.DB, identity middleware.IdentityService) *Environment {
	vm := otto.New()
	env := &Environment{VM: vm, DataStore: dataStore, Identity: identity}
	env.SetUp()
	return env
}

//SetUp initialize environment
func (env *Environment) SetUp() {
	for _, init := range inits {
		init(env)
	}
	env.goCallbacks = []ext.GoCallback{}
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
	_, err = vm.Run(script)
	if err != nil {
		return err
	}
	return nil
}

//RegisterObject register new object for VM
func (env *Environment) RegisterObject(objectID string, object interface{}) {
	env.VM.Set(objectID, object)
}

//LoadExtensionsForPath for returns extensions for specific path
func (env *Environment) LoadExtensionsForPath(extensions []*schema.Extension, path string) error {
	var err error
	for _, extension := range extensions {
		if extension.Match(path) {
			code := extension.Code
			if extension.CodeType == "donburi" {
				err = env.runDonburi(code)
				if err != nil {
					return err
				}
			} else if extension.CodeType == "go" {
				callback := ext.GetGoCallback(code)
				if callback != nil {
					env.goCallbacks = append(env.goCallbacks, callback)
				}
			} else {
				script, err := env.VM.Compile(extension.URL, code)
				if err != nil {
					return err
				}
				_, err = env.VM.Run(script)
				if err != nil {
					return err
				}
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
func (env *Environment) HandleEvent(event string, context map[string]interface{}) error {
	vm := env.VM
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
	for _, callback := range env.goCallbacks {
		err = callback(event, context)
		if err != nil {
			return err
		}
	}
	return err
}

//Clone makes clone of the environment
func (env *Environment) Clone() ext.Environment {
	newEnv := NewEnvironment(env.DataStore, env.Identity)
	newEnv.VM = env.VM.Copy()
	newEnv.goCallbacks = env.goCallbacks
	return newEnv
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
