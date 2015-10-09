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

// +build v8
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

package v8

import (
	ext "github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
	"github.com/twinj/uuid"
)

var inits = []func(env *Environment){}

//RegisterInit registers init code
func RegisterInit(init func(env *Environment)) {
	inits = append(inits, init)
}

//Environment is a interface for extension environment
type Environment struct {
	rpc         *RPC
	goCallbacks []ext.GoCallback
}

//NewEnvironment makes new v8 environment
func NewEnvironment() *Environment {
	env := &Environment{
		rpc:         NewRPC(),
		goCallbacks: []ext.GoCallback{},
	}
	for _, init := range inits {
		init(env)
	}
	return env
}

//Load loads script for environment.
func (env *Environment) Load(source, code string) error {
	return env.rpc.Load(source, code)
}

//RegistObject register method
func (env *Environment) RegistObject(objectID string, object interface{}) {
	env.rpc.RegistObject(objectID, object)
}

//Clone returns self
func (env *Environment) Clone() ext.Environment {
	return env
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
				err := env.rpc.Load(extension.URL, code)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

//HandleEvent kicks event in loaded environment
func (env *Environment) HandleEvent(event string, context map[string]interface{}) error {
	contextID := uuid.NewV4().String()
	env.rpc.contexts[contextID] = context
	context["contextID"] = contextID
	defer delete(env.rpc.contexts, contextID)
	updatedContext := env.rpc.BlockCall("gohan_handler", "handle_event", []interface{}{contextID, context, event})
	updatedContextMap, ok := updatedContext.(map[string]interface{})
	if !ok {
		return nil
	}
	for key, value := range updatedContextMap {
		context[key] = value
	}
	return nil
}

//RunDonburi runs Donburi code
func (env *Environment) runDonburi(yamlCode string) error {
	return nil
}
