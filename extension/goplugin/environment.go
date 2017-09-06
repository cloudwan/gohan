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
	"encoding/json"
	"fmt"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
	"time"

	"sort"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	logger "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/sync"
	"github.com/twinj/uuid"
)

var log = logger.NewLogger()

// Handler is a generic handler
type Handler func(context goext.Context, environment goext.IEnvironment) error

// Handlers is a list of generic handlers
type Handlers []Handler

// PrioritizedHandlers is a prioritized list of generic handlers
type PrioritizedHandlers map[goext.Priority]Handlers

// EventPrioritizedHandlers is a per-event prioritized list of generic handlers
type EventPrioritizedHandlers map[string]PrioritizedHandlers

// SchemaHandler is a schema handler
type SchemaHandler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error

// SchemaHandlers is a list of schema handlers
type SchemaHandlers []SchemaHandler

// PrioritizedSchemaHandlers is a prioritized list of schema handlers
type PrioritizedSchemaHandlers map[goext.Priority]SchemaHandlers

// SchemaPrioritizedSchemaHandlers is a per-schema prioritized list of schema handlers
type SchemaPrioritizedSchemaHandlers map[string]PrioritizedSchemaHandlers

// EventSchemaPrioritizedSchemaHandlers is a per-event per-schema prioritized list of schema handlers
type EventSchemaPrioritizedSchemaHandlers map[string]SchemaPrioritizedSchemaHandlers

// GlobHandlers is a global registry of global handlers
var GlobHandlers EventPrioritizedHandlers

// GlobSchemaHandlers is a global registry of schema handlers
var GlobSchemaHandlers EventSchemaPrioritizedSchemaHandlers

// GlobRawTypes is a global registry of runtime types used to map raw resources
var GlobRawTypes = make(map[string]reflect.Type)

// GlobTypes is a global registry of runtime types used to map resources
var GlobTypes = make(map[string]reflect.Type)

// GlobEnvironments is a global registry of loaded shared environments
var GlobEnvironments = map[string]*Environment{}

// Environment golang based rawEnvironment for gohan extension
type Environment struct {
	// initial
	source          string
	beforeStartInit func() error

	// extension
	extCore     goext.ICore
	extLogger   goext.ILogger
	extSchemas  goext.ISchemas
	extSync     goext.ISync
	extDatabase goext.IDatabase

	// internals
	name  string
	db    db.DB
	ident middleware.IdentityService
	sync  sync.Sync

	// plugin related
	manager *schema.Manager
	plugin  *plugin.Plugin

	schemasFnRaw plugin.Symbol
	schemasFn    func() []string
	schemas      []string

	initFnRaw plugin.Symbol
	initFns   []func(goext.IEnvironment) error

	traceID string
}

// NewEnvironment create new gohan extension rawEnvironment based on context
func NewEnvironment(name string, dataStore db.DB, ident middleware.IdentityService, sync sync.Sync) *Environment {
	newEnvironment := &Environment{
		name:  name,
		db:    dataStore,
		ident: ident,
		sync:  sync,
	}
	return newEnvironment
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

// Sync returns an implementation to Sync interface
func (thisEnvironment *Environment) Sync() goext.ISync {
	return thisEnvironment.extSync
}

// Database returns an implementation to IDatabase interface
func (thisEnvironment *Environment) Database() goext.IDatabase {
	return thisEnvironment.extDatabase
}

//bind sets environment bindings
func (thisEnvironment *Environment) bind() {
	thisEnvironment.extCore = NewCore(thisEnvironment)
	thisEnvironment.extLogger = NewLogger(thisEnvironment)
	thisEnvironment.extSchemas = NewSchemas(thisEnvironment)
	thisEnvironment.extSync = NewSync(thisEnvironment)
	thisEnvironment.extDatabase = NewDatabase(thisEnvironment)
}

// Start starts already loaded environment
func (thisEnvironment *Environment) Start() error {
	var err error

	if thisEnvironment.source == "" {
		panic("golang extension is not loaded")
	}

	log.Debug("Starting golang environment: %s", thisEnvironment.source)

	// Before start init
	if thisEnvironment.beforeStartInit != nil {
		log.Debug("Calling before start init golang environment: %s", thisEnvironment.source)

		if err = thisEnvironment.beforeStartInit(); err != nil {
			log.Error("Failed to before start init golang extension: %s; error: %s", thisEnvironment.source, err)
			return err
		}
	} else {
		log.Debug("Before start init is not set for golang environment: %s", thisEnvironment.source)
	}

	// Manager
	thisEnvironment.manager = schema.GetManager()

	// bind
	thisEnvironment.bind()

	// Generating TraceID
	thisEnvironment.traceID = uuid.NewV4().String()

	// Init
	log.Debug("Start golang extension: %s", thisEnvironment.source)

	for _, initFn := range thisEnvironment.initFns {
		err = initFn(thisEnvironment)

		if err != nil {
			log.Error("Failed to start golang extension: %s; error: %s", thisEnvironment.source, err)
			return err
		}
	}

	log.Debug("Golang extension started: %s", thisEnvironment.source)

	return nil
}

// Load loads script into the environment
func (thisEnvironment *Environment) Load(source string, beforeStartInit func() error) (bool, error) {
	if existingEnvironment, ok := GlobEnvironments[source]; ok {
		log.Debug("Golang extension already in registry: %s", source)

		// link to existing
		thisEnvironment.source = existingEnvironment.source
		thisEnvironment.beforeStartInit = existingEnvironment.beforeStartInit

		// extension
		thisEnvironment.extCore = existingEnvironment.extCore
		thisEnvironment.extLogger = existingEnvironment.extLogger
		thisEnvironment.extSchemas = existingEnvironment.extSchemas
		thisEnvironment.extSync = existingEnvironment.extSync
		thisEnvironment.extDatabase = existingEnvironment.extDatabase

		// internals
		thisEnvironment.name = existingEnvironment.name
		thisEnvironment.db = existingEnvironment.db
		thisEnvironment.ident = existingEnvironment.ident
		thisEnvironment.sync = existingEnvironment.sync

		// plugin related
		thisEnvironment.manager = existingEnvironment.manager
		thisEnvironment.plugin = existingEnvironment.plugin

		thisEnvironment.initFnRaw = existingEnvironment.initFnRaw
		thisEnvironment.initFns = existingEnvironment.initFns

		return false, nil
	}
	GlobEnvironments[source] = thisEnvironment

	log.Debug("Loading golang extension: %s", source)

	thisEnvironment.source = source
	thisEnvironment.beforeStartInit = beforeStartInit

	var err error
	var ok bool

	if filepath.Ext(source) != ".so" {
		return false, fmt.Errorf("golang extensions plugin must be a *.so file, file: %s", source)
	}

	thisEnvironment.plugin, err = plugin.Open(source)

	if err != nil {
		return false, fmt.Errorf("failed to load golang extension: %s", err)
	}

	// Init
	thisEnvironment.initFnRaw, err = thisEnvironment.plugin.Lookup("Init")

	if err != nil {
		return false, fmt.Errorf("golang extension does not export Init: %s", err)
	}

	initFn, ok := thisEnvironment.initFnRaw.(func(goext.IEnvironment) error)

	if !ok {
		return false, fmt.Errorf("invalid signature of Init function in golang extension: %s", source)
	}

	thisEnvironment.initFns = append(thisEnvironment.initFns, initFn)

	return true, nil
}

//LoadExtensionsForPath for returns extensions for specific path
func (thisEnvironment *Environment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
	for _, extension := range extensions {
		if extension.Match(path) {
			if extension.CodeType != "goext" {
				continue
			}
			url := strings.TrimPrefix(extension.URL, "file://")
			if url == "" {
				log.Warning("ignore golang extension '%s' without plugin", extension.ID)
				continue
			}
			loaded, err := thisEnvironment.Load(url, nil)
			if err != nil {
				return err
			}
			if loaded {
				if err = thisEnvironment.Start(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (thisEnvironment *Environment) dispatchSchemaEvent(prioritizedSchemaHandlers PrioritizedSchemaHandlers, sch Schema, event string, context map[string]interface{}) error {
	thisEnvironment.Logger().Debugf("Starting event: %s, schema: %s", event, sch.rawSchema.ID)
	defer thisEnvironment.Logger().Debugf("Finished event: %s, schema: %s", event, sch.rawSchema.ID)
	if resource, err := thisEnvironment.resourceFromContext(sch, context); err == nil {
		for _, priority := range sortSchemaHandlers(prioritizedSchemaHandlers) {
			for index, schemaEventHandler := range prioritizedSchemaHandlers[priority] {
				if err := schemaEventHandler(context, resource, thisEnvironment); err != nil {
					return fmt.Errorf("failed to dispatch schema event '%s' to schema '%s' at priority '%d' with index '%d': %s",
						event, sch.ID(), priority, index, err)
				}
				thisEnvironment.updateContextFromResource(context, resource)
			}
		}
	} else {
		return fmt.Errorf("failed to parse resource from context with schema '%s' for event '%s': %s", sch.ID(), event, err)
	}

	return nil
}

func sortSchemaHandlers(schemaHandlers PrioritizedSchemaHandlers) []goext.Priority {
	priorities := []goext.Priority{}
	for priority := range schemaHandlers {
		priorities = append(priorities, priority)
	}
	sort.Ints(priorities)
	return priorities
}

func sortHandlers(handlers PrioritizedHandlers) []goext.Priority {
	priorities := []goext.Priority{}
	for priority := range handlers {
		priorities = append(priorities, priority)
	}
	sort.Ints(priorities)
	return priorities
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
		for _, priority := range sortHandlers(prioritizedEventHandlers) {
			for index, eventHandler := range prioritizedEventHandlers[priority] {
				if err := eventHandler(context, thisEnvironment); err != nil {
					return fmt.Errorf("failed to dispatch event '%s' at priority '%d' with index '%d': %s",
						event, priority, index, err)
				}
			}
		}
	}

	return nil
}

func (thisEnvironment *Environment) updateContextFromResource(context goext.Context, resource interface{}) error {
	if resource == nil {
		context["resource"] = nil
		return nil
	}

	if _, ok := context["resource"]; !ok {
		return nil
	}

	if _, ok := context["resource"].(map[string]interface{}); !ok {
		return fmt.Errorf("failed to convert context resource to map during update context from resource")
	}

	if resourceMap, ok := thisEnvironment.resourceToMap(resource).(map[string]interface{}); ok {
		for key, value := range resourceMap {
			if _, ok := context["resource"].(map[string]interface{})[key]; ok {
				context["resource"].(map[string]interface{})[key] = value
			}
		}
	} else {
		return fmt.Errorf("failed to convert resource to map during update context from resource")
	}

	return nil
}

func (thisEnvironment *Environment) updateResourceFromContextR(resource interface{}, resourceData map[string]interface{}) error {
	resourceValue := reflect.ValueOf(resource)
	resourceElem := resourceValue.Elem()
	resourceElemType := resourceElem.Type()

	if resourceElemType.Kind() != reflect.Struct {
		panic("resource must be a struct")
	}

	for i := 0; i < resourceElemType.NumField(); i++ {
		resourceFieldType := resourceElemType.Field(i)
		resourceFieldTagDB := resourceFieldType.Tag.Get("db")
		resourceField := resourceElem.Field(i)
		val := reflect.ValueOf(resourceData[resourceFieldTagDB])

		if resourceFieldType.Type.Kind() == reflect.Struct {
			if _, ok := resourceData[resourceFieldTagDB].(map[string]interface{}); ok {
				thisEnvironment.updateResourceFromContextR(resourceField.Interface(), resourceData[resourceFieldTagDB].(map[string]interface{}))
			} else if strings.Contains(resourceFieldType.Type.String(), "goext.Null") {
				if resourceData[resourceFieldTagDB] != nil {
					if val.Type() == resourceFieldType.Type {
						resourceField.Set(val)
					} else {
						resourceField.Field(0).Set(val)
						resourceField.Field(1).Set(reflect.ValueOf(true))
					}
				} else {
					resourceField.Field(1).Set(reflect.ValueOf(false))
				}
			} else {
				resourceField.Set(val)
			}
		} else {
			if val.IsValid() {
				resourceField.Set(val)
			}
		}
	}

	return nil
}

func (thisEnvironment *Environment) updateResourceFromContext(resource interface{}, context goext.Context) error {
	if resource == nil {
		return nil
	}

	if _, ok := context["resource"]; !ok {
		return nil
	}

	if resourceData, ok := context["resource"].(map[string]interface{}); ok {
		return thisEnvironment.updateResourceFromContextR(resource, resourceData)
	}

	return fmt.Errorf("failed to convert context resource to map during update resource from context")
}

func (thisEnvironment *Environment) resourceToMap(resource interface{}) interface{} {
	resourceValue := reflect.ValueOf(resource)
	resourceElem := resourceValue.Elem()
	resourceElemType := resourceElem.Type()

	if resourceElemType.Kind() == reflect.Struct {
		data := make(map[string]interface{})

		for i := 0; i < resourceElemType.NumField(); i++ {
			resourceFieldType := resourceElemType.Field(i)
			resourceFieldTagDB := resourceFieldType.Tag.Get("db")
			resourceFieldInterface := resourceElem.Field(i).Interface()

			data[resourceFieldTagDB] = thisEnvironment.resourceToMap(&resourceFieldInterface)
		}

		return data
	}

	return resourceElem.Interface()
}

func (thisEnvironment *Environment) resourceFromContext(sch Schema, context map[string]interface{}) (res goext.Resource, err error) {
	rawSchema := sch.rawSchema

	resourceType, ok := GlobRawTypes[rawSchema.ID]
	if !ok {
		return nil, fmt.Errorf("No type registered for title: %s", rawSchema.ID)
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
					if value.Type() != field.Type() && field.Kind() == reflect.Int && value.Kind() == reflect.Float64 { // reflect treats number(N, 0) as float
						field.SetInt(int64(value.Float()))
					} else {
						field.Set(value)
					}
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

// RegisterRawType registers a runtime type of raw resource for a given name
func (thisEnvironment *Environment) RegisterRawType(name string, typeValue interface{}) {
	targetType := reflect.TypeOf(typeValue)
	GlobRawTypes[name] = targetType
}

// RawType returns a runtime type for a given named raw resource
func (thisEnvironment *Environment) RawType(name string) reflect.Type {
	return GlobRawTypes[name]
}

// RegisterType registers a runtime type of resource for a given name
func (thisEnvironment *Environment) RegisterType(name string, typeValue interface{}) {
	targetType := reflect.TypeOf(typeValue)
	GlobTypes[name] = targetType
}

// ResourceType returns a runtime type for a given named resource
func (thisEnvironment *Environment) ResourceType(name string) reflect.Type {
	return GlobTypes[name]
}

// Stop stops the environment to its initial state
func (thisEnvironment *Environment) Stop() {
	log.Info("Stop environment")

	// reset globals
	GlobHandlers = nil
	GlobSchemaHandlers = nil
	GlobRawTypes = make(map[string]reflect.Type)
	GlobEnvironments = map[string]*Environment{}

	// reset locals
	thisEnvironment.extCore = nil
	thisEnvironment.extLogger = nil
	thisEnvironment.extSchemas = nil
	thisEnvironment.extSync = nil
	thisEnvironment.extDatabase = nil
}

// Reset clear the environment to its initial state
func (thisEnvironment *Environment) Reset() {
	log.Info("Reset environment")

	thisEnvironment.Stop()
	thisEnvironment.Start()
}

// Clone makes a clone of the rawEnvironment
func (thisEnvironment *Environment) Clone() extension.Environment {
	env := &Environment{
		source:          thisEnvironment.source,
		beforeStartInit: thisEnvironment.beforeStartInit,

		// internals
		name:  thisEnvironment.name,
		db:    thisEnvironment.db,
		ident: thisEnvironment.ident,
		sync:  thisEnvironment.sync,

		// plugin related
		manager: thisEnvironment.manager,
		plugin:  thisEnvironment.plugin,

		schemasFnRaw: thisEnvironment.schemasFnRaw,
		schemasFn:    thisEnvironment.schemasFn,
		schemas:      thisEnvironment.schemas,

		initFnRaw: thisEnvironment.initFnRaw,
		initFns:   thisEnvironment.initFns,

		traceID: uuid.NewV4().String(),
	}
	env.bind()

	return env
}

// IsEventHandled returns whether a given event is handled by this environment
func (thisEnvironment *Environment) IsEventHandled(event string, context map[string]interface{}) bool {
	return true
}
