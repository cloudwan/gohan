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

package extension

import (
	"time"

	"github.com/cloudwan/gohan/schema"
)

//MultiEnvironment can handle multiple environment
type MultiEnvironment struct {
	childEnv []Environment
}

//NewEnvironment create new gohan extension environment based on context
func NewEnvironment(envs []Environment) *MultiEnvironment {
	env := &MultiEnvironment{
		childEnv: envs,
	}
	env.SetUp()
	return env
}

//SetUp initialize environment
func (env *MultiEnvironment) SetUp() {
}

//LoadExtensionsForPath for returns extensions for specific path
func (env *MultiEnvironment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
	for _, env := range env.childEnv {
		err := env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, path)
		if err != nil {
			return err
		}
	}
	return nil
}

//HandleEvent handles event
func (env *MultiEnvironment) HandleEvent(event string, context map[string]interface{}) error {
	for _, env := range env.childEnv {
		err := env.HandleEvent(event, context)
		if err != nil {
			return err
		}
	}
	return nil
}

//Clone makes clone of the environment
func (env *MultiEnvironment) Clone() Environment {
	newEnv := NewEnvironment([]Environment{})
	for _, env := range env.childEnv {
		newEnv.childEnv = append(newEnv.childEnv, env.Clone())
	}
	return newEnv
}
