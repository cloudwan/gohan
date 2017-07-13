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

// Schema event handlers
type SchemaHandler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error
type SchemaHandlers []SchemaHandler
type PrioritizedSchemaHandlers map[goext.Priority]SchemaHandlers
type SchemaPrioritizedSchemaHandlers map[string]PrioritizedSchemaHandlers
type EventSchemaPrioritizedSchemaHandlers map[string]SchemaPrioritizedSchemaHandlers

// event handlers
var GlobHandlers EventPrioritizedHandlers
var GlobSchemaHandlers EventSchemaPrioritizedSchemaHandlers
var GlobResourceTypes = make(map[string]reflect.Type)

// Environment golang based rawEnvironment for gohan extension
type Environment struct {

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
	}
	environment.SetUp()
	return environment
}

// SetUp initialize rawEnvironment
func (thisEnvironment *Environment) SetUp() {
	GlobHandlers = EventPrioritizedHandlers{}

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

var GlobRegistry = map[string]bool{}

// Load loads script into the environment
func (thisEnvironment *Environment) Load(source, code string) error {
	if _, ok := GlobRegistry[source]; ok {
		return nil
	}
	GlobRegistry[source] = true

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
				log.Warning("found golang extension '%s' without plugin - ignored", extension.ID)
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

func (thisEnvironment *Environment) dispatchSchemaEvent(prioritizedSchemaHandlers PrioritizedSchemaHandlers, sch Schema, event string, context map[string]interface{}) error {
	if resource, err := thisEnvironment.resourceFromContext(sch, context); err == nil {
		for priority, schemaEventHandlers := range prioritizedSchemaHandlers {
			for index, schemaEventHandler := range schemaEventHandlers {
				if err := schemaEventHandler(context, resource, thisEnvironment); err != nil {
					return fmt.Errorf("failed to dispatch schema event '%s' to schema '%s' at priority '%d' with index '%d': %s",
						event, sch.ID(), priority, index, err)
				}
				thisEnvironment.updateResourceInContext(context, resource)
			}
		}
	} else {
		return fmt.Errorf("failed to parse resource from context with schema '%s' for event '%s': %s", sch.ID(), event, err)
	}

	return nil
}

// HandleEvent handles an event
func (thisEnvironment *Environment) HandleEvent(event string, context map[string]interface{}) error {
	context["event_type"] = event

	// dispatch to schema handlers
	if schemaPrioritizedSchemaHandlers, ok := GlobSchemaHandlers[event]; ok {
		if iSchemaID, ok := context["schema_id"]; ok {
			schemaID := iSchemaID.(string)
			if prioritizedSchemaHandlers, ok := schemaPrioritizedSchemaHandlers[schemaID]; ok {
				if iSchema := thisEnvironment.Schemas().Find(schemaID); iSchema != nil {
					sch := iSchema.(*Schema)
					if err := thisEnvironment.dispatchSchemaEvent(prioritizedSchemaHandlers, *sch, event, context); err != nil {
						return err
					}
				}
			}
		} else {
			// all
			for schemaID, prioritizedSchemaHandlers := range schemaPrioritizedSchemaHandlers {
				if iSchema := thisEnvironment.Schemas().Find(schemaID); iSchema != nil {
					sch := iSchema.(*Schema)
					if err := thisEnvironment.dispatchSchemaEvent(prioritizedSchemaHandlers, *sch, event, context); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("could not find schema: %s", schemaID)
				}
			}
		}
	}

	// dispatch to generic handlers
	if prioritizedEventHandlers, ok := GlobHandlers[event]; ok {
		for priority, eventHandlers := range prioritizedEventHandlers {
			for index, eventHandler := range eventHandlers {
				if err := eventHandler(context, thisEnvironment); err != nil {
					return fmt.Errorf("failed to dispatch event '%s' at priority '%d' with index '%d': %s",
						event, priority, index, err)
				}
			}
		}
	}

	return nil
}

func (thisEnvironment *Environment) updateResourceInContext(context goext.Context, resource interface{}) error {
	if resource == nil {
		context["resource"] = nil
		return nil
	}

	if _, ok := context["resource"]; !ok {
		return nil
	}

	if _, ok := context["resource"].(map[string]interface{}); !ok {
		return fmt.Errorf("failed to convert context resource to map during update in context")
	}

	if resourceMap, ok := thisEnvironment.resourceToMap(resource).(map[string]interface{}); ok {
		for key, value := range resourceMap {
			if _, ok := context["resource"].(map[string]interface{})[key]; ok {
				context["resource"].(map[string]interface{})[key] = value
			}
		}
	} else {
		return fmt.Errorf("failed to convert resource to map during update in context")
	}

	return nil
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

func (thisEnvironment *Environment) resourceFromContext(sch Schema, context map[string]interface{}) (res goext.Resource, err error) {
	rawSchema := sch.rawSchema

	resourceType, ok := GlobResourceTypes[rawSchema.ID]
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
func (thisEnvironment *Environment) RegisterEventHandler(event string, handler func(context goext.Context, environment goext.IEnvironment) error, priority goext.Priority) {
	if GlobHandlers == nil {
		GlobHandlers = EventPrioritizedHandlers{}
	}

	if GlobHandlers[event] == nil {
		GlobHandlers[event] = PrioritizedHandlers{}
	}

	if GlobHandlers[event][priority] == nil {
		GlobHandlers[event][priority] = Handlers{}
	}

	GlobHandlers[event][priority] = append(GlobHandlers[event][priority], handler)
}

// RegisterSchemaEventHandler register an event handler for a schema
func (thisEnvironment *Environment) RegisterSchemaEventHandler(schemaID string, event string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority goext.Priority) {
	if GlobSchemaHandlers == nil {
		GlobSchemaHandlers = EventSchemaPrioritizedSchemaHandlers{}
	}

	if GlobSchemaHandlers[event] == nil {
		GlobSchemaHandlers[event] = SchemaPrioritizedSchemaHandlers{}
	}

	if GlobSchemaHandlers[event][schemaID] == nil {
		GlobSchemaHandlers[event][schemaID] = PrioritizedSchemaHandlers{}
	}

	if GlobSchemaHandlers[event][schemaID][priority] == nil {
		GlobSchemaHandlers[event][schemaID][priority] = SchemaHandlers{}
	}

	GlobSchemaHandlers[event][schemaID][priority] = append(GlobSchemaHandlers[event][schemaID][priority], handler)
}

// RegisterResourceType registers a runtime type for a given name
func (thisEnvironment *Environment) RegisterResourceType(name string, typeValue interface{}) {
	targetType := reflect.TypeOf(typeValue)
	GlobResourceTypes[name] = targetType
}

// ResourceType returns a runtime type for a given named resource
func (thisEnvironment *Environment) ResourceType(name string) reflect.Type {
	return GlobResourceTypes[name]
}

// Clone makes a clone of the rawEnvironment
func (thisEnvironment *Environment) Clone() extension.Environment {
	return &Environment{
		// extension
		extCore:    thisEnvironment.extCore,
		extLogger:  thisEnvironment.extLogger,
		extSchemas: thisEnvironment.extSchemas,

		// internals
		name:            thisEnvironment.name,
		dataStore:       thisEnvironment.dataStore,
		timeLimit:       thisEnvironment.timeLimit,
		timeLimits:      thisEnvironment.timeLimits,
		identityService: thisEnvironment.identityService,
		sync:            thisEnvironment.sync,

		// plugin related
		manager: thisEnvironment.manager,
		plugin:  thisEnvironment.plugin,

		schemasFnRaw: thisEnvironment.schemasFnRaw,
		schemasFn:    thisEnvironment.schemasFn,
		schemas:      thisEnvironment.schemas,

		initFnRaw: thisEnvironment.initFnRaw,
		initFn:    thisEnvironment.initFn,
	}
}
