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

package golang

import (
	"time"

	ext "github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
)

//Environment gohan script based environment for gohan extension
type Environment struct {
	goCallbacks []GoCallback
}

//NewEnvironment create new gohan extension environment based on context
func NewEnvironment() *Environment {
	env := &Environment{}
	env.SetUp()
	return env
}

//SetUp initialize environment
func (env *Environment) SetUp() {
	env.goCallbacks = []GoCallback{}
}

//Load loads script for environment
func (env *Environment) Load(source, code string) error {
	return nil
}

//LoadExtensionsForPath for returns extensions for specific path
func (env *Environment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
	for _, extension := range extensions {
		if extension.Match(path) {
			if extension.CodeType != "go" {
				continue
			}
			code := extension.Code
			callback := GetGoCallback(code)
			if callback != nil {
				env.goCallbacks = append(env.goCallbacks, callback)
			}
		}
	}
	return nil
}

//HandleEvent handles event
func (env *Environment) HandleEvent(event string, context map[string]interface{}) (err error) {
	context["event_type"] = event
	for _, callback := range env.goCallbacks {
		err = callback(event, context)
		if err != nil {
			return
		}
	}
	return
}

//Clone makes clone of the environment
func (env *Environment) Clone() ext.Environment {
	clone := NewEnvironment()
	clone.goCallbacks = env.goCallbacks
	return clone
}

// IsEventHandled returns whether a given event is handled by this environment
func (env *Environment) IsEventHandled(event string, context map[string]interface{}) bool {
	return true
}
