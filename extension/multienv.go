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
	baseEnvs []Environment
	childEnv []Environment
}

//NewEnvironment create new gohan extension environment based on context
func NewEnvironment(envs []Environment) *MultiEnvironment {
	env := &MultiEnvironment{
		baseEnvs: envs,
		childEnv: make([]Environment, len(envs)),
	}
	env.SetUp()
	return env
}

//SetUp initialize environment
func (env *MultiEnvironment) SetUp() {
}

//LoadExtensionsForPath for returns extensions for specific path
func (env *MultiEnvironment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
	// FIXME(p-kozlowski) this method is invoked only on startup, before the first cloning
	// also, it is not safe to invoke it later. therefore, it should not be a part of public interface, rather
	// a part of initialization
	for _, env := range env.baseEnvs {
		err := env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, path)
		if err != nil {
			return err
		}
	}
	return nil
}

//HandleEvent handles event
func (env *MultiEnvironment) HandleEvent(event string, context map[string]interface{}) error {
	for i := range env.childEnv {
		err := env.handleEventInChild(i, event, context)
		if err != nil {
			return err
		}
	}
	return nil
}

func (env *MultiEnvironment) handleEventInChild(childIdx int, event string, context map[string]interface{}) error {
	if !env.baseEnvs[childIdx].IsEventHandled(event, context) {
		return nil
	}
	env.cloneChildIfNeeded(childIdx)
	return env.childEnv[childIdx].HandleEvent(event, context)
}

func (env *MultiEnvironment) cloneChildIfNeeded(childIdx int) {
	if env.childEnv[childIdx] == nil {
		env.childEnv[childIdx] = env.baseEnvs[childIdx].Clone()
	}
}

//Clone makes clone of the environment
func (env *MultiEnvironment) Clone() Environment {
	return NewEnvironment(env.baseEnvs)
}

// IsEventHandled returns whether a given event is handled by this environment
func (env *MultiEnvironment) IsEventHandled(event string, context map[string]interface{}) bool {
	for _, env := range env.baseEnvs {
		if env.IsEventHandled(event, context) {
			return true
		}
	}
	return false
}
