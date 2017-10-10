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
	"fmt"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
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
	schemaID, err := getSchemaId(context)
	if err != nil {
		log.Panic(err)
	}

	// JS extensions expect context["schema"] to be a *schema.Schema.
	// If a schema is already set, we should overwrite it with proper type and
	// restore it once we're done with handling the event
	defer restoreOriginalSchema(context)()
	context["schema"] = core.env.Schemas().Find(schemaID).(*Schema).raw

	// as above, if present, context["transaction"] should be a transaction.Transaction
	defer restoreOriginalTransaction(context)()
	ensureRawTxInContext(context)

	envManager := extension.GetManager()
	if env, found := envManager.GetEnvironment(schemaID); found {
		return env.HandleEvent(event, context)
	}
	return nil
}

func getSchemaId(context goext.Context) (string, error) {
	rawSchemaID, ok := context["schema_id"]
	if !ok {
		return "", fmt.Errorf("TriggerEvent: schema_id missing in context")
	}

	return rawSchemaID.(string), nil
}

func restoreOriginalSchema(context goext.Context) func() {
	return restoreContextByKey(context, "schema")
}

func restoreOriginalTransaction(context goext.Context) func() {
	return restoreContextByKey(context, "transaction")
}

func restoreContextByKey(context goext.Context, key string) func() {
	if originalValue, hasValue := context[key]; hasValue {
		return func() {
			context[key] = originalValue
		}
	} else {
		return func() {
			delete(context, key)
		}
	}
}

// makes sure that context["transaction"] is a transaction.Transaction
func ensureRawTxInContext(context goext.Context) {
	if tx, hasTx := contextGetTransaction(context); hasTx {
		contextSetTransaction(context, tx)
	}
}

// HandleEvent Causes the given event to be handled within the same environment
func (core *Core) HandleEvent(event string, context goext.Context) error {
	return core.env.HandleEvent(event, context)
}

// NewCore allocates Core
func NewCore(env *Environment) *Core {
	return &Core{env: env}
}

// Clone allocates a clone of Core; object may be nil
func (core *Core) Clone(env *Environment) *Core {
	if core == nil {
		return nil
	}
	return &Core{
		env: env,
	}
}
