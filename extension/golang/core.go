// Copyright (C) 2017 NTT Innovation Institute, Inc.
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
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
)

type Core struct {
	environment *Environment
}

func (thisCore *Core) RegisterSchemaEventHandler(schemaID string, eventName string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority goext.Priority) {
	thisCore.environment.RegisterSchemaEventHandler(schemaID, eventName, handler, priority)
}

func (thisCore *Core) RegisterEventHandler(eventName string, handler func(context goext.Context, environment goext.IEnvironment) error, priority goext.Priority) {
	thisCore.environment.RegisterEventHandler(eventName, handler, priority)
}

// TriggerEvent Causes the given event to be handled in all environments (across different-language extensions)
func (thisCore *Core) TriggerEvent(event string, context goext.Context) {
	schemaID := ""

	if s, ok := context["schema"]; ok {
		schemaID = s.(goext.ISchema).ID()
	} else {
		log.Panic("TriggerEvent: schema not found")
	}

	envManager := extension.GetManager()
	envManager.HandleEventInAllEnvironments(context, event, schemaID)
}

// HandleEvent Causes the given event to be handled within the same environment
func (thisCore *Core) HandleEvent(event string, context goext.Context) {
	thisCore.environment.HandleEvent(event, context)
}

// New core creates a new Core implementation
func NewCore(environment *Environment) goext.ICore {
	return &Core{environment: environment}
}
