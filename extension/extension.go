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
	"sync"
	"time"

	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/singleton"
)

//Environment is a interface for extension environment
type Environment interface {
	LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error
	HandleEvent(event string, context map[string]interface{}) error
	Clone() Environment
	IsEventHandled(event string, context map[string]interface{}) bool
}

//Manager takes care of mapping schemas to Environments.
//This is a singleton class.
type Manager struct {
	environments map[string]Environment
	mu           sync.RWMutex
}

//RegisterEnvironment registers a new environment for the given schema ID
func (manager *Manager) RegisterEnvironment(schemaID string, env Environment) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, ok := manager.environments[schemaID]; ok {
		return fmt.Errorf("Environment already registered for schema '%s'", schemaID)
	}
	manager.environments[schemaID] = env
	return nil
}

//UnRegisterEnvironment removes an environment registered for the given schema ID
func (manager *Manager) UnRegisterEnvironment(schemaID string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, ok := manager.environments[schemaID]; !ok {
		return fmt.Errorf("No environment registered for this schema")
	}
	delete(manager.environments, schemaID)
	return nil
}

//GetEnvironment returns the environment registered for the given schema ID
func (manager *Manager) GetEnvironment(schemaID string) (env Environment, ok bool) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	env, ok = manager.environments[schemaID]
	if ok {
		env = env.Clone()
	}
	return
}

// HandleEventInAllEnvironments handles the event in all registered environments
func (manager *Manager) HandleEventInAllEnvironments(context map[string]interface{}, event string, schemaID string) error {
	for name := range manager.environments {
		err := HandleEvent(context, manager.environments[name], event, schemaID)
		if err != nil {
			return err
		}
	}
	return nil
}

//GetManager gets manager
func GetManager() *Manager {
	return singleton.Get("extension/manager", func() interface{} {
		return &Manager{
			environments: map[string]Environment{},
		}
	}).(*Manager)
}

//ClearManager clears manager
func ClearManager() {
	singleton.Clear("extension/manager")
}

// Error is created when a problem has occurred during event handling. It contains the information
// required to reraise the javascript exception that caused this error.
type Error struct {
	error
	ExceptionInfo map[string]interface{}
}

func measureExtensionTime(timeStarted time.Time, event string, schemaID string) {
	metrics.UpdateTimer(timeStarted, "ext.%s.%s", schemaID, event)
}

//HandleEvent handles the event in the given environment
func HandleEvent(context map[string]interface{}, environment Environment, event string, schemaID string) error {
	defer measureExtensionTime(time.Now(), event, schemaID)
	if err := environment.HandleEvent(event, context); err != nil {
		return err
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

//Errorf makes extension error
func Errorf(code int, name, message string) Error {
	return Error{fmt.Errorf("%v", message),
		map[string]interface{}{
			"code":    code,
			"name":    name,
			"message": message,
		}}
}
