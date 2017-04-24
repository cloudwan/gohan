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

package goext

type Handler func(context Context, environment *Environment) error
type HandlerList []Handler
type PriorityHandlerList map[Priority]HandlerList
type EventPriorityHandlerList map[string]PriorityHandlerList

type HandlerSchema func(context Context, resource Resource, environment *Environment) error
type HandlerSchemaListOfHandlers []HandlerSchema
type HandlerSchemaListOfPriorities map[Priority]HandlerSchemaListOfHandlers
type HandlerSchemaListOfSchemas map[string]HandlerSchemaListOfPriorities
type HandlersSchemaListOfEvents map[string]HandlerSchemaListOfSchemas

// Environment is the only scope of gohan available for a golang extension
// no other packages must not be imported and used
type Environment struct {
	// event handlers (event -> priority -> list -> handler)
	Handlers       EventPriorityHandlerList
	HandlersSchema HandlersSchemaListOfEvents

	// modules
	Core    CoreInterface
	Logger  LoggerInterface
	Schemas SchemasInterface
}

type EnvContainer interface {
	Env() *Environment
}
