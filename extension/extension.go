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

package extension

import (
	"fmt"

	"github.com/cloudwan/gohan/schema"
)

var modules = map[string]interface{}{}

//GoCallback is type for go based callback
type GoCallback func(event string, context map[string]interface{}) error

var goCallbacks = map[string]GoCallback{}

//Environment is a interface for extension environment
type Environment interface {
	Load(source, code string) error
	RegisterObject(objectID string, object interface{})
	LoadExtensionsForPath(extensions []*schema.Extension, path string) error
	HandleEvent(event string, context map[string]interface{}) error
	Clone() Environment
}

var manager *Manager

//Manager takes care of mapping schemas to Environments.
//This is a singleton class.
type Manager struct {
	environments map[string]Environment
}

//RegisterEnvironment registers a new environment for the given schema ID
func (manager *Manager) RegisterEnvironment(schemaID string, env Environment) error {
	if _, ok := manager.environments[schemaID]; ok {
		return fmt.Errorf("Environment already registered for this schema")
	}
	manager.environments[schemaID] = env
	return nil
}

//UnRegisterEnvironment removes an environment registered for the given schema ID
func (manager *Manager) UnRegisterEnvironment(schemaID string) error {
	if _, ok := manager.environments[schemaID]; !ok {
		return fmt.Errorf("No environment registered for this schema")
	}
	delete(manager.environments, schemaID)
	return nil
}

//GetEnvironment returns the environment registered for the given schema ID
func (manager *Manager) GetEnvironment(schemaID string) (env Environment, ok bool) {
	env, ok = manager.environments[schemaID]
	if ok {
		env = env.Clone()
	}
	return
}

//GetManager gets manager
func GetManager() *Manager {
	if manager == nil {
		manager = &Manager{
			environments: map[string]Environment{},
		}
	}
	return manager
}

//ClearManager clears manager
func ClearManager() {
	manager = nil
}

//RegisterModule registers modules
func RegisterModule(name string, module interface{}) {
	modules[name] = module
}

//RequireModule returns module
func RequireModule(name string) interface{} {
	module, ok := modules[name]
	if ok {
		return module
	}
	return nil
}

//RegisterGoCallback register go call back
func RegisterGoCallback(name string, callback GoCallback) {
	goCallbacks[name] = callback
}

//GetGoCallback returns registered go callback
func GetGoCallback(name string) GoCallback {
	callback, ok := goCallbacks[name]
	if !ok {
		return nil
	}
	return callback
}

// Error is created when a problem has occured during event handling. It contains the information
// required to reraise the javascript exception that caused this error.
type Error struct {
	error
	ExceptionInfo map[string]interface{}
}

//HandleEvent handles the event in the given environment
func HandleEvent(context map[string]interface{}, environment Environment, event string) error {
	if err := environment.HandleEvent(event, context); err != nil {
		return fmt.Errorf("extension error: %s", err)
	}
	exceptionInfoRaw, ok := context["exception"]
	if !ok {
		return nil
	}
	exceptionInfo, ok := exceptionInfoRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("extension returned invalid error information")
	}
	exceptionMessage := context["exception_message"]
	return Error{fmt.Errorf("%v", exceptionMessage), exceptionInfo}
}
