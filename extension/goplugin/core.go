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
	"github.com/cloudwan/gohan/util"
	"github.com/twinj/uuid"
)

// Core is an implementation of ICore interface
type Core struct {
	env *Environment
}

// RegisterSchemaEventHandler registers a schema handler
func (core *Core) RegisterSchemaEventHandler(schemaID string, eventName string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority int) {
	core.env.RegisterSchemaEventHandler(schemaID, eventName, handler, priority)
}

// RegisterEventHandler registers a global handler
func (core *Core) RegisterEventHandler(eventName string, handler func(context goext.Context, environment goext.IEnvironment) error, priority int) {
	core.env.RegisterEventHandler(eventName, handler, priority)
}

// TriggerEvent causes the given event to be handled in all environments (across different-language extensions)
func (core *Core) TriggerEvent(event string, context goext.Context) error {
	schemaID := ""

	if s, ok := context["schema"]; ok {
		schemaID = s.(goext.ISchema).ID()
	} else {
		log.Panic("TriggerEvent: missing schema in context")
	}
	context["schema_id"] = schemaID

	envManager := extension.GetManager()
	return envManager.HandleEventInAllEnvironments(context, event, schemaID)
}

// HandleEvent Causes the given event to be handled within the same environment
func (core *Core) HandleEvent(event string, context goext.Context) error {
	return core.env.HandleEvent(event, context)
}

// NewUUID create a new unique ID
func (core *Core) NewUUID() string {
	return uuid.NewV4().String()
}

// Config gets parameter from config
func (core *Core) Config(key string, defaultValue interface{}) interface{} {
	config := util.GetConfig()
	return config.GetParam(key, defaultValue)
}

// NewCore allocates Core
func NewCore(env *Environment) *Core {
	return &Core{env: env}
}

// Clone allocates a clone of Core; object may be nil
func (core *Core) Clone() *Core {
	if core == nil {
		return nil
	}
	return &Core{
		env: core.env,
	}
}