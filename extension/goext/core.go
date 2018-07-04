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

package goext

// Handler is a generic handler
type Handler func(context Context, environment IEnvironment) *Error

// SchemaHandler is a schema handler
type SchemaHandler func(context Context, resource Resource, environment IEnvironment) *Error

// ICore is an interface to core parts of Gohan: event triggering and registering
type ICore interface {
	// TriggerEvent causes the given event to be handled in all environments (across different-language extensions)
	TriggerEvent(event string, context Context) error
	// HandleEvent Causes the given event to be handled within the same environment
	HandleEvent(event string, context Context) error

	// RegisterEventHandler registers a global handler
	RegisterEventHandler(event string, handler Handler, priority int)
	// RegisterSchemaEventHandler registers a schema handler
	RegisterSchemaEventHandler(schemaID SchemaID, event string, schemaHandler SchemaHandler, priority int)
}
