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

package goplugin

import (
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/twinj/uuid"
)

// Core is an implementation of ICore interface
type Core struct {
	environment *Environment
}

// RegisterSchemaEventHandler registers a schema handler
func (thisCore *Core) RegisterSchemaEventHandler(schemaID string, eventName string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority goext.Priority) {
	thisCore.environment.RegisterSchemaEventHandler(schemaID, eventName, handler, priority)
}

// RegisterEventHandler registers a global handler
func (thisCore *Core) RegisterEventHandler(eventName string, handler func(context goext.Context, environment goext.IEnvironment) error, priority goext.Priority) {
	thisCore.environment.RegisterEventHandler(eventName, handler, priority)
}

// TriggerEvent causes the given event to be handled in all environments (across different-language extensions)
func (thisCore *Core) TriggerEvent(event string, context goext.Context) error {
	schemaID := ""

	if s, ok := context["schema"]; ok {
		schemaID = s.(goext.ISchema).ID()
	} else {
		log.Panic("TriggerEvent: missing schema in context")
	}

	envManager := extension.GetManager()
	return envManager.HandleEventInAllEnvironments(context, event, schemaID)
}

// HandleEvent Causes the given event to be handled within the same environment
func (thisCore *Core) HandleEvent(event string, context goext.Context) error {
	return thisCore.environment.HandleEvent(event, context)
}

// NewUUID create a new unique ID
func (thisCore *Core) NewUUID() string {
	return uuid.NewV4().String()
}

// NewCore allocates Core
func NewCore(environment *Environment) goext.ICore {
	return &Core{environment: environment}
}
