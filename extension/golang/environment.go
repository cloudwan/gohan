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
	"encoding/json"
	"fmt"
	"path/filepath"
	"plugin"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	logger "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"
)

var log = logger.NewLogger()

// Global event handlers
type Handler func(context goext.Context, environment goext.IEnvironment) error
type Handlers []Handler
type PrioritizedHandlers map[goext.Priority]Handlers
type EventPrioritizedHandlers map[string]PrioritizedHandlers

// Per-schema event handlers
type SchemaHandler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error
type SchemaHandlers []SchemaHandler
type PrioritizedSchemaHandlers map[goext.Priority]SchemaHandlers
type SchemaPrioritizedSchemaHandlers map[string]PrioritizedSchemaHandlers
type EventSchemaPrioritizedSchemaHandlers map[string]SchemaPrioritizedSchemaHandlers

// Environment golang based rawEnvironment for gohan extension
type Environment struct {
	// event handlers
	// note: these fields are public since test framework uses them; golang extensions will not see these fields
	Handlers       EventPrioritizedHandlers
	SchemaHandlers EventSchemaPrioritizedSchemaHandlers

	// extension
	extCore    goext.ICore
	extLogger  goext.ILogger
	extSchemas goext.ISchemas

	// internals
	name            string
	dataStore       db.DB
	timeLimit       time.Duration
	timeLimits      []*schema.EventTimeLimit
	identityService middleware.IdentityService
	sync            sync.Sync
	resourceTypes   map[string]reflect.Type

	// plugin related
	manager *schema.Manager
	plugin  *plugin.Plugin

	schemasFnRaw plugin.Symbol
	schemasFn    func() []string
	schemas      []string

	initFnRaw plugin.Symbol
	initFn    func(goext.IEnvironment) error
}

// NewEnvironment create new gohan extension rawEnvironment based on context
func NewEnvironment(name string, dataStore db.DB, identityService middleware.IdentityService, sync sync.Sync) *Environment {
	environment := &Environment{
		name:            name,
		dataStore:       dataStore,
		identityService: identityService,
		sync:            sync,
		resourceTypes:   make(map[string]reflect.Type),
	}
	environment.SetUp()
	return environment
}

// SetUp initialize rawEnvironment
func (thisEnvironment *Environment) SetUp() {
	thisEnvironment.Handlers = EventPrioritizedHandlers{}

	thisEnvironment.extCore = NewCore(thisEnvironment)
	thisEnvironment.extLogger = NewLogger(thisEnvironment)
	thisEnvironment.extSchemas = NewSchemas(thisEnvironment)
}

// Core returns an implementation to Core interface
func (thisEnvironment *Environment) Core() goext.ICore {
	return thisEnvironment.extCore
}

// Logger returns an implementation to Logger interface
func (thisEnvironment *Environment) Logger() goext.ILogger {
	return thisEnvironment.extLogger
}

// Schemas returns an implementation to Schemas interface
func (thisEnvironment *Environment) Schemas() goext.ISchemas {
	return thisEnvironment.extSchemas
}

// Load loads script into the environment
func (thisEnvironment *Environment) Load(source, code string) error {
	log.Debug("Loading golang extension: %s", source)

	var err error
	var ok bool

	if filepath.Ext(source) != ".so" {
		return fmt.Errorf("Golang extensions source code must be a *.so file, source: %s", source)
	}

	thisEnvironment.plugin, err = plugin.Open(source)

	if err != nil {
		return fmt.Errorf("Failed to load golang extension: %s", err)
	}

	thisEnvironment.manager = schema.GetManager()

	// Schemas
	thisEnvironment.schemasFnRaw, err = thisEnvironment.plugin.Lookup("Schemas")

	if err != nil {
		return fmt.Errorf("Golang extension does not export Schemas: %s", err)
	}

	thisEnvironment.schemasFn, ok = thisEnvironment.schemasFnRaw.(func() []string)

	if !ok {
		log.Error("Invalid signature of Schemas function in golang extension: %s", source)
		return err
	}

	thisEnvironment.schemas = thisEnvironment.schemasFn()

	for _, schemaPath := range thisEnvironment.schemas {
		if err = thisEnvironment.manager.LoadSchemaFromFile(filepath.Dir(source) + "/" + schemaPath); err != nil {
			return fmt.Errorf("Failed to load schema: %s", err)
		}
	}

	// Init
	thisEnvironment.initFnRaw, err = thisEnvironment.plugin.Lookup("Init")

	if err != nil {
		return fmt.Errorf("Golang extension does not export Init: %s", err)
	}

	log.Debug("Init golang extension: %s", source)

	thisEnvironment.initFn, ok = thisEnvironment.initFnRaw.(func(goext.IEnvironment) error)

	if !ok {
		log.Error("Invalid signature of Init function in golang extension: %s", source)
		return err
	}

	err = thisEnvironment.initFn(thisEnvironment)

	if err != nil {
		log.Error("Failed to initialize golang extension: %s; error: %s", source, err)
		return err
	}

	log.Debug("Golang extension initialized: %s", source)

	return nil
}

//LoadExtensionsForPath for returns extensions for specific path
func (thisEnvironment *Environment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
	for _, extension := range extensions {
		if extension.Match(path) {
			code := extension.Code
			if extension.CodeType != "go" {
				continue
			}
			if extension.URL == "" {
				log.Warning("Found golang extension without plugin - ignored")
				continue
			}
			url := strings.TrimPrefix(extension.URL, "file://")
			err := thisEnvironment.Load(url, code)
			if err != nil {
				return err
			}
		}
	}
	// setup time limits for matching extensions
	thisEnvironment.timeLimit = timeLimit
	for _, timeLimit := range timeLimits {
		if timeLimit.Match(path) {
			thisEnvironment.timeLimits = append(thisEnvironment.timeLimits, schema.NewEventTimeLimit(timeLimit.EventRegex, timeLimit.TimeDuration))
		}
	}
	return nil
}

// HandleEvent handles an event
func (thisEnvironment *Environment) HandleEvent(event string, context map[string]interface{}) (err error) {
	context["event_type"] = event

	if err = thisEnvironment.dispatchEvent(event, context); err != nil {
		log.Error("failed to dispatch event: %s", err)
		return err
	}

	schemasHandlersList, ok := thisEnvironment.SchemaHandlers[event]

	if ok {
		for schemaID, handlers := range schemasHandlersList {
			err = thisEnvironment.dispatchEventWithResource(schemaID, handlers, event, context)

			if err != nil {
				log.Error("dispatchEventWithResource Error: %s", err)
				return err
			}
		}
	}

	return
}

//@todo: this and below function (dispatchEventWithResource) should be somehow unified because they're very similar but operates on different types
func (thisEnvironment *Environment) dispatchEvent(event string, context goext.Context) error {
	handlers, hasHandlers := thisEnvironment.Handlers[event]

	if !hasHandlers {
		return nil
	}

	//@todo: it's a poor idea to sort the prioritized handlers each time the event is being handled.
	priorities := []int{}
	for priority := range handlers {
		priorities = append(priorities, int(priority))
	}

	sort.Ints(priorities)

	for priority := range priorities {
		for _, fn := range handlers[goext.Priority(priority)] {
			err := fn(context, thisEnvironment)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (thisEnvironment *Environment) dispatchEventWithResource(schemaID string, handlers PrioritizedSchemaHandlers, event string, context goext.Context) (err error) {
	//@todo: it's a poor idea to sort the prioritized handlers each time the event is being handled.
	priorities := []int{}
	for priority := range handlers {
		priorities = append(priorities, int(priority))
	}

	sort.Ints(priorities)

	resource, err := thisEnvironment.schemaIDToResource(schemaID, context)

	if err != nil {
		return err
	}

	//defer thisEnvironment.updateResourceInContext(context, resource)

	for priority := range priorities {
		for _, fn := range handlers[goext.Priority(priority)] {
			err := fn(context, resource, thisEnvironment)

			if err != nil {
				return err
			}
		}
	}

	return
}

func (thisEnvironment *Environment) updateResourceInContext(context goext.Context, resource interface{}) {
	for key, value := range thisEnvironment.resourceToMap(resource).(map[string]interface{}) {
		if _, ok := context["resource"].(map[string]interface{})[key]; ok {
			context["resource"].(map[string]interface{})[key] = value
		}
	}
}

func (thisEnvironment *Environment) resourceToMap(res interface{}) interface{} {
	val := reflect.ValueOf(res)
	elem := val.Elem()
	elemType := elem.Type()

	if elemType.Kind() == reflect.Struct {
		data := make(map[string]interface{})

		for i := 0; i < elemType.NumField(); i++ {
			fieldType := elemType.Field(i)
			propertyName := fieldType.Tag.Get("db")
			fieldValue := elem.Field(i).Interface()

			data[propertyName] = thisEnvironment.resourceToMap(&fieldValue)
		}

		return data
	}

	return elem.Interface()
}

func (thisEnvironment *Environment) schemaIDToResource(schemaID string, context map[string]interface{}) (res goext.Resource, err error) {
	manager := schema.GetManager()
	rawSchema, ok := manager.Schema(schemaID)

	if !ok {
		return nil, fmt.Errorf("Could not find schema for ID: %s", schemaID)
	}

	resourceType, ok := thisEnvironment.resourceTypes[rawSchema.ID]
	if !ok {
		return nil, fmt.Errorf("No type registered for schema title: %s", rawSchema.ID)
	}

	resource := reflect.New(resourceType)

	resourceData, ok := context["resource"]

	if ok {
		data := resourceData.(map[string]interface{})
		for i := 0; i < resourceType.NumField(); i++ {
			field := resource.Elem().Field(i)
			fieldType := resourceType.Field(i)
			propertyName := fieldType.Tag.Get("db")
			if propertyName == "" {
				return nil, fmt.Errorf("Missing tag 'db' for resource %s field %s", resourceType.Name(), fieldType.Name)
			}
			property, err := rawSchema.GetPropertyByID(propertyName)
			if err != nil {
				return nil, err
			}
			if fieldType.Type.Kind() == reflect.Struct {
				mapJSON, err := json.Marshal(data[property.ID])
				if err != nil {
					return nil, err
				}
				newField := reflect.New(field.Type())
				fieldJSON := string(mapJSON)
				fieldInterface := newField.Interface()
				err = json.Unmarshal([]byte(fieldJSON), &fieldInterface)
				if err != nil {
					return nil, err
				}
				field.Set(newField.Elem())
			} else {
				value := reflect.ValueOf(data[property.ID])
				if value.IsValid() {
					field.Set(value)
				}
			}

		}
	}

	return resource.Interface(), nil
}

// RegisterEventHandler registers an event handler
func (thisEnvironment *Environment) RegisterEventHandler(eventName string, handler func(context goext.Context, environment goext.IEnvironment) error, priority goext.Priority) {
	if thisEnvironment.Handlers == nil {
		thisEnvironment.Handlers = EventPrioritizedHandlers{}
	}

	if thisEnvironment.Handlers[eventName] == nil {
		thisEnvironment.Handlers[eventName] = PrioritizedHandlers{}
	}

	if thisEnvironment.Handlers[eventName][priority] == nil {
		thisEnvironment.Handlers[eventName][priority] = Handlers{}
	}

	thisEnvironment.Handlers[eventName][priority] = append(thisEnvironment.Handlers[eventName][priority], handler)

	return
}

// RegisterSchemaEventHandler register an event handler for a schema
func (thisEnvironment *Environment) RegisterSchemaEventHandler(schemaID string, eventName string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority goext.Priority) {

	if thisEnvironment.SchemaHandlers == nil {
		thisEnvironment.SchemaHandlers = EventSchemaPrioritizedSchemaHandlers{}
	}

	if thisEnvironment.SchemaHandlers[eventName] == nil {
		thisEnvironment.SchemaHandlers[eventName] = SchemaPrioritizedSchemaHandlers{}
	}

	if thisEnvironment.SchemaHandlers[eventName][schemaID] == nil {
		thisEnvironment.SchemaHandlers[eventName][schemaID] = PrioritizedSchemaHandlers{}
	}

	if thisEnvironment.SchemaHandlers[eventName][schemaID][priority] == nil {
		thisEnvironment.SchemaHandlers[eventName][schemaID][priority] = SchemaHandlers{}
	}

	thisEnvironment.SchemaHandlers[eventName][schemaID][priority] = append(thisEnvironment.SchemaHandlers[eventName][schemaID][priority], handler)

	return
}

// RegisterResourceType registers a runtime type for a given name
func (thisEnvironment *Environment) RegisterResourceType(name string, typeValue interface{}) {
	targetType := reflect.TypeOf(typeValue)
	thisEnvironment.resourceTypes[name] = targetType
}

// ResourceType returns a runtime type for a given named resource
func (thisEnvironment *Environment) ResourceType(name string) reflect.Type {
	return thisEnvironment.resourceTypes[name]
}

// Clone makes a clone of the rawEnvironment
func (thisEnvironment *Environment) Clone() extension.Environment {
	return &Environment{
		name:            thisEnvironment.name,
		dataStore:       thisEnvironment.dataStore,
		identityService: thisEnvironment.identityService,
		sync:            thisEnvironment.sync,
		resourceTypes:   thisEnvironment.resourceTypes,
	}
}
