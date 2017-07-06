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
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"plugin"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"

	"encoding/json"

	ext "github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
)

var log = l.NewLogger()

// Environment golang based rawEnvironment for gohan extension
type Environment struct {
	Name       string
	DataStore  db.DB
	timeLimit  time.Duration
	timeLimits []*schema.EventTimeLimit
	Identity   middleware.IdentityService
	Sync       sync.Sync

	// golang extension environment
	extEnvironment goext.Environment
	resourceTypes  map[string]reflect.Type
}

// NewEnvironment create new gohan extension rawEnvironment based on context
func NewEnvironment(name string, dataStore db.DB, identity middleware.IdentityService, sync sync.Sync) *Environment {
	env := &Environment{
		Name:          name,
		DataStore:     dataStore,
		Identity:      identity,
		Sync:          sync,
		resourceTypes: make(map[string]reflect.Type),
	}
	env.SetUp()
	return env
}

// SetUp initialize rawEnvironment
func (environment *Environment) SetUp() {
	environment.extEnvironment.Handlers = goext.EventPriorityHandlerList{}

	environment.extEnvironment.Core = bindCore(environment)
	environment.extEnvironment.Logger = bindLogger(environment)
	environment.extEnvironment.Schemas = bindSchemas(environment)
}

//Load loads script for rawEnvironment
func (environment *Environment) Load(source, code string) error {
	log.Debug("Loading golang extension: %s", source)

	if filepath.Ext(source) != ".so" {
		return fmt.Errorf("Golang extensions source code must be a *.so file, source: %s", source)
	}

	p, err := plugin.Open(source)

	if err != nil {
		return fmt.Errorf("Failed to load golang extension: %s", err)
	}

	// Schemas
	SchemasFnRaw, err := p.Lookup("Schemas")

	mgr := schema.GetManager()

	if err != nil {
		return fmt.Errorf("Golang extension does not export Schemas: %s", err)
	}

	schemasFn, ok := SchemasFnRaw.(func() []string)

	if !ok {
		log.Error("Invalid signature of Schemas function in golang extension: %s", source)
		return err
	}

	schemas := schemasFn()

	for _, schemaPath := range schemas {
		if err = mgr.LoadSchemaFromFile(filepath.Dir(source) + "/" + schemaPath); err != nil {
			return fmt.Errorf("Failed to load schema: %s", err)
		}
	}

	// Init
	ifn, err := p.Lookup("Init")

	if err != nil {
		return fmt.Errorf("Golang extension does not export Init: %s", err)
	}

	log.Debug("Init golang extension: %s", source)

	fn, ok := ifn.(func(*goext.Environment) error)

	if !ok {
		log.Error("Invalid signature of Init function in golang extension: %s", source)
		return err
	}

	err = fn(&environment.extEnvironment)

	if err != nil {
		log.Error("Failed to initialize golang extension: %s; error: %s", source, err)
		return err
	}

	log.Debug("Golang extension initialized: %s", source)

	return nil
}

//LoadExtensionsForPath for returns extensions for specific path
func (environment *Environment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
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
			err := environment.Load(url, code)
			if err != nil {
				return err
			}
		}
	}
	// setup time limits for matching extensions
	environment.timeLimit = timeLimit
	for _, timeLimit := range timeLimits {
		if timeLimit.Match(path) {
			environment.timeLimits = append(environment.timeLimits, schema.NewEventTimeLimit(timeLimit.EventRegex, timeLimit.TimeDuration))
		}
	}
	return nil
}

//HandleEvent
func (environment *Environment) HandleEvent(event string, context map[string]interface{}) (err error) {
	context["event_type"] = event
	handlers, ok := environment.extEnvironment.Handlers[event]

	if ok {
		err = environment.dispatchEvent(handlers, event, context)
	}
	if err != nil {
		log.Error("dispatchEvent Error: %s", err)
		return err
	}

	schemasHandlersList, ok := environment.extEnvironment.HandlersSchema[event]

	if ok {
		for schemaID, handlers := range schemasHandlersList {
			err = environment.dispatchEventWithResource(schemaID, handlers, event, context)

			if err != nil {
				log.Error("dispatchEventWithResource Error: %s", err)
				return err
			}
		}
	}

	return
}

//@todo: this and below function (dispatchEventWithResource) should be somehow unified because they're very similar but operates on different types
func (environment *Environment) dispatchEvent(handlers goext.PriorityHandlerList, event string, context map[string]interface{}) (err error) {
	//@todo: it's a poor idea to sort the prioritized handlers each time the event is being handled.
	priorities := []int{}
	for priority, _ := range handlers {
		priorities = append(priorities, int(priority))
	}

	sort.Ints(priorities)

	for priority := range priorities {
		for _, fn := range handlers[goext.Priority(priority)] {
			err := fn(context, &environment.extEnvironment)

			if err != nil {
				return err
			}
		}
	}

	return
}

func (environment *Environment) dispatchEventWithResource(schemaID string, handlers goext.HandlerSchemaListOfPriorities, event string, context map[string]interface{}) (err error) {
	//@todo: it's a poor idea to sort the prioritized handlers each time the event is being handled.
	priorities := []int{}
	for priority, _ := range handlers {
		priorities = append(priorities, int(priority))
	}

	sort.Ints(priorities)

	resource, err := environment.schemaIdToResource(schemaID, context)

	if err != nil {
		return err
	}

	for priority := range priorities {
		for _, fn := range handlers[goext.Priority(priority)] {
			err := fn(context, resource, &environment.extEnvironment)

			if err != nil {
				return err
			}
		}
	}

	return
}

func (self *Environment) schemaIdToResource(schemaID string, context map[string]interface{}) (res goext.Resource, err error) {
	manager := schema.GetManager()
	rawSchema, ok := manager.Schema(schemaID)

	if !ok {
		return nil, fmt.Errorf("Could not find schema for ID: %s", schemaID)
	}

	resourceType, ok := self.resourceTypes[rawSchema.ID]
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
			property, err := rawSchema.GetPropertyByID(propertyName)
			if err != nil {
				return nil, err
			}
			if fieldType.Type.Kind() == reflect.Struct {
				mapJson, err := json.Marshal(data[property.ID])
				if err != nil {
					return nil, err
				}
				newField := reflect.New(field.Type())
				fieldJson := string(mapJson)
				fieldIface := newField.Interface()
				err = json.Unmarshal([]byte(fieldJson), &fieldIface)
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

//RegisterEventHandler
func (environment *Environment) RegisterEventHandler(eventName string, handler func(context goext.Context, environment *goext.Environment) error, priority goext.Priority) {
	if environment.extEnvironment.Handlers == nil {
		environment.extEnvironment.Handlers = goext.EventPriorityHandlerList{}
	}

	if environment.extEnvironment.Handlers[eventName] == nil {
		environment.extEnvironment.Handlers[eventName] = goext.PriorityHandlerList{}
	}

	if environment.extEnvironment.Handlers[eventName][priority] == nil {
		environment.extEnvironment.Handlers[eventName][priority] = goext.HandlerList{}
	}

	environment.extEnvironment.Handlers[eventName][priority] = append(environment.extEnvironment.Handlers[eventName][priority], handler)

	return
}

//RegisterSchemaEventHandler
func (environment *Environment) RegisterSchemaEventHandler(schemaID string, eventName string, handler func(context goext.Context, resource goext.Resource, environment *goext.Environment) error, priority goext.Priority) {

	if environment.extEnvironment.HandlersSchema == nil {
		environment.extEnvironment.HandlersSchema = goext.HandlersSchemaListOfEvents{}
	}

	if environment.extEnvironment.HandlersSchema[eventName] == nil {
		environment.extEnvironment.HandlersSchema[eventName] = goext.HandlerSchemaListOfSchemas{}
	}

	if environment.extEnvironment.HandlersSchema[eventName][schemaID] == nil {
		environment.extEnvironment.HandlersSchema[eventName][schemaID] = goext.HandlerSchemaListOfPriorities{}
	}

	if environment.extEnvironment.HandlersSchema[eventName][schemaID][priority] == nil {
		environment.extEnvironment.HandlersSchema[eventName][schemaID][priority] = goext.HandlerSchemaListOfHandlers{}
	}

	environment.extEnvironment.HandlersSchema[eventName][schemaID][priority] = append(environment.extEnvironment.HandlersSchema[eventName][schemaID][priority], handler)

	return
}

func (environment *Environment) RegisterResourceType(name string, typeValue interface{}) {
	targetType := reflect.TypeOf(typeValue)
	environment.resourceTypes[name] = targetType
}

func (environment *Environment) ResourceType(name string) reflect.Type {
	return environment.resourceTypes[name]
}

//Clone makes clone of the rawEnvironment
func (environment *Environment) Clone() ext.Environment {
	return &Environment{
		Name:           environment.Name,
		DataStore:      environment.DataStore,
		Identity:       environment.Identity,
		Sync:           environment.Sync,
		extEnvironment: environment.extEnvironment,
		resourceTypes:  environment.resourceTypes,
	}
}

func (environment *Environment) ExtEnvironment() goext.Environment {
	return environment.extEnvironment
}
