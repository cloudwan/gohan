// Copyright (C) 2016  Juniper Networks, Inc.
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

package gohanscript

import (
	"time"

	"github.com/cloudwan/gohan/db"
	ext "github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
)

//Environment gohan script based environment for gohan extension
type Environment struct {
	VM          *VM
	DataStore   db.DB
	timelimit   time.Duration
	Identity    middleware.IdentityService
	goCallbacks []ext.GoCallback
}

//NewEnvironment create new gohan extension environment based on context
func NewEnvironment(dataStore db.DB, identity middleware.IdentityService, timelimit time.Duration) *Environment {
	vm := NewVM()
	env := &Environment{
		VM:        vm,
		DataStore: dataStore,
		Identity:  identity,
		timelimit: timelimit,
	}
	env.SetUp()
	return env
}

//SetUp initialize environment
func (env *Environment) SetUp() {
	env.goCallbacks = []ext.GoCallback{}
}

//Load loads script for environment
func (env *Environment) Load(source, code string) error {
	return nil
}

//LoadExtensionsForPath for returns extensions for specific path
func (env *Environment) LoadExtensionsForPath(extensions []*schema.Extension, path string) error {
	var err error
	for _, extension := range extensions {
		if extension.Match(path) {
			if extension.CodeType == "go" {
				code := extension.Code
				callback := ext.GetGoCallback(code)
				if callback != nil {
					env.goCallbacks = append(env.goCallbacks, callback)
				}
			} else {
				err = env.VM.LoadString(extension.File, extension.Code)
				if err != nil {
					log.Error(extension.Code)
					return err
				}
			}
		}
	}
	return nil
}

//HandleEvent handles event
func (env *Environment) HandleEvent(event string, context map[string]interface{}) (err error) {
	context["event_type"] = event

	successCh := make(chan bool)

	defer func() {
		if caught := recover(); caught != nil {
			if caught == env.VM.timeoutError {
				log.Error(env.VM.timeoutError.Error())
				err = env.VM.timeoutError
				return
			}
			panic(caught) // Something else happened, repanic!
		}
	}()
	timer := time.NewTimer(env.timelimit)

	go func() {
		for {
			select {
			case <-timer.C:
				env.VM.StopChan <- func() {
					panic(env.VM.timeoutError)
				}
				return
			case <-successCh:
				// extention executed successfully
				return
			}
		}
	}()
	err = env.VM.Run(context)
	for _, callback := range env.goCallbacks {
		err = callback(event, context)
		if err != nil {
			return
		}
	}
	timer.Stop()
	successCh <- true
	return
}

//Clone makes clone of the environment
func (env *Environment) Clone() ext.Environment {
	newEnv := NewEnvironment(env.DataStore, env.Identity, env.timelimit)
	newEnv.VM = env.VM.Clone()
	newEnv.goCallbacks = env.goCallbacks
	return newEnv
}
