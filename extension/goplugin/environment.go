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
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"plugin"
	"reflect"
	"sort"
	"strings"
	"time"

	gohan_db "github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	gohan_logger "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	gohan_sync "github.com/cloudwan/gohan/sync"
	"github.com/mohae/deepcopy"
	"github.com/twinj/uuid"
)

var log = gohan_logger.NewLogger()

// Handlers is a list of generic handlers
type Handlers []goext.Handler

// PrioritizedHandlers is a prioritized list of generic handlers
type PrioritizedHandlers map[int]Handlers

// EventPrioritizedHandlers is a per-event prioritized list of generic handlers
type EventPrioritizedHandlers map[string]PrioritizedHandlers

// SchemaHandlers is a list of schema handlers
type SchemaHandlers []goext.SchemaHandler

// PrioritizedSchemaHandlers is a prioritized list of schema handlers
type PrioritizedSchemaHandlers map[int]SchemaHandlers

// SchemaPrioritizedSchemaHandlers is a per-schema prioritized list of schema handlers
type SchemaPrioritizedSchemaHandlers map[string]PrioritizedSchemaHandlers

// EventSchemaPrioritizedSchemaHandlers is a per-event per-schema prioritized list of schema handlers
type EventSchemaPrioritizedSchemaHandlers map[string]SchemaPrioritizedSchemaHandlers

// Environment golang based environment for gohan extensions
type Environment struct {
	initFns         map[string]func(goext.IEnvironment) error
	beforeStartHook func(env *Environment) error
	afterStopHook   func(env *Environment)

	coreImpl     *Core
	loggerImpl   *Logger
	schemasImpl  *Schemas
	syncImpl     *Sync
	databaseImpl *Database

	name       string
	traceID    string
	timeLimit  time.Duration
	timeLimits []*schema.EventTimeLimit

	handlers       EventPrioritizedHandlers
	schemaHandlers EventSchemaPrioritizedSchemaHandlers

	rawTypes map[string]reflect.Type
	types    map[string]reflect.Type
}

// Internal interface for goplugin environment
type IEnvironment interface {
	goext.IEnvironment
	extension.Environment

	RegisterSchemaEventHandler(schemaID string, event string, schemaHandler goext.SchemaHandler, priority int)
	RegisterRawType(name string, typeValue interface{})
	RegisterType(name string, typeValue interface{})

	dispatchSchemaEvent(prioritizedSchemaHandlers PrioritizedSchemaHandlers, sch Schema, event string, context map[string]interface{}) error
	getSchemaHandlers(event string) (SchemaPrioritizedSchemaHandlers, bool)
	getHandlers(event string) (PrioritizedHandlers, bool)
	getRawType(schemaID string) (reflect.Type, bool)
	getType(schemaID string) (reflect.Type, bool)
	getTraceID() string
	getTimeLimit() time.Duration
	getTimeLimits() []*schema.EventTimeLimit
}

// NewEnvironment create new gohan extension rawEnvironment based on context
func NewEnvironment(name string, beforeStartHook func(env *Environment) error, afterStopHook func(env *Environment)) *Environment {
	env := &Environment{
		initFns:         map[string]func(goext.IEnvironment) error{},
		beforeStartHook: beforeStartHook,
		afterStopHook:   afterStopHook,

		name: name,

		rawTypes: make(map[string]reflect.Type),
		types:    make(map[string]reflect.Type),
	}
	return env
}

// Handlers returns a copy of handlers - used for testing
func (env *Environment) Handlers() EventPrioritizedHandlers {
	return env.handlers
}

// SchemaHandlers returns a copy of handlers - used for testing
func (env *Environment) SchemaHandlers() EventSchemaPrioritizedSchemaHandlers {
	return env.schemaHandlers
}

// RawTypes returns raw types
func (env *Environment) RawTypes() map[string]reflect.Type {
	return env.rawTypes
}

// Types returns types
func (env *Environment) Types() map[string]reflect.Type {
	return env.types
}

// Config returns an implementation to Config interface
func (env *Environment) Config() goext.IConfig {
	return &Config{}
}

// Core returns an implementation to Core interface
func (env *Environment) Core() goext.ICore {
	return env.coreImpl
}

// Logger returns an implementation to Logger interface
func (env *Environment) Logger() goext.ILogger {
	return env.loggerImpl
}

// Schemas returns an implementation to Schemas interface
func (env *Environment) Schemas() goext.ISchemas {
	return env.schemasImpl
}

// Sync returns an implementation to Sync interface
func (env *Environment) Sync() goext.ISync {
	return env.syncImpl
}

// Database returns an implementation to IDatabase interface
func (env *Environment) Database() goext.IDatabase {
	return env.databaseImpl
}

// HTTP returns an implementation to IHTTP interface
func (env *Environment) HTTP() goext.IHTTP {
	return &HTTP{}
}

// Auth returns an implementation to IAuth interface
func (env *Environment) Auth() goext.IAuth {
	return &Auth{}
}

func (env *Environment) Util() goext.IUtil {
	return &Util{}
}

// SetDatabase sets and binds database implementation
func (env *Environment) SetDatabase(db gohan_db.DB) {
	env.bindDatabase(db)
}

// SetSync sets and binds sync implementation
func (env *Environment) SetSync(sync gohan_sync.Sync) {
	env.bindSync(sync)
}

func (env *Environment) bindCore() {
	env.coreImpl = NewCore(env)
}

func (env *Environment) bindLogger() {
	env.loggerImpl = NewLogger(env)
}

func (env *Environment) bindSchemas() {
	env.schemasImpl = NewSchemas(env)
}

func (env *Environment) bindSchemasToEnv(envToBind IEnvironment) {
	env.schemasImpl = NewSchemas(envToBind)
}

func (env *Environment) bindSync(sync gohan_sync.Sync) {
	env.syncImpl = NewSync(sync)
}

func (env *Environment) bindDatabase(db gohan_db.DB) {
	env.databaseImpl = NewDatabase(db)
}

// Start starts already loaded environment
func (env *Environment) Start() error {
	var err error

	log.Debug("Starting go environment: %s", env.name)

	// bind
	env.bindCore()
	env.bindLogger()
	env.bindSchemas()

	// Before start init
	if env.beforeStartHook != nil {
		log.Debug("Calling before start for go environment: %s", env.name)

		if err = env.beforeStartHook(env); err != nil {
			log.Error("Failed to call before start for go extension: %s; error: %s", env.name, err)
			return err
		}
	} else {
		log.Debug("Before start init is not set for go environment: %s", env.name)
	}

	env.traceID = newTraceID()

	// init extensions
	log.Debug("Start go extension: %s", env.name)

	for _, initFn := range env.initFns {
		err = initFn(env)

		if err != nil {
			log.Error("Failed to start go extension: %s; error: %s", env.name, err)
			return err
		}
	}

	log.Debug("Go extension started: %s", env.name)

	return nil
}

// Load loads script into the environment
func (env *Environment) Load(fileName string) error {
	log.Debug("Loading go extension: %s", fileName)

	if _, ok := env.initFns[fileName]; ok {
		log.Warning(fmt.Sprintf("Go extension %s already loaded in %s", fileName, env.name))
		return nil
	}

	var err error
	var ok bool

	if filepath.Ext(fileName) != ".so" {
		return fmt.Errorf("go extension must be a *.so file, file: %s", fileName)
	}

	pl, err := plugin.Open(fileName)

	if err != nil {
		return fmt.Errorf("failed to load go extension: %s", err)
	}

	// Init
	initFnRaw, err := pl.Lookup("Init")

	if err != nil {
		return fmt.Errorf("go extension does not export Init: %s", err)
	}

	initFn, ok := initFnRaw.(func(goext.IEnvironment) error)

	if !ok {
		return fmt.Errorf("invalid signature of Init function in go extension: %s", fileName)
	}

	env.initFns[fileName] = initFn

	return nil
}

//LoadExtensionsForPath for returns extensions for specific path
func (env *Environment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
	for _, extension := range extensions {
		if extension.Match(path) {
			if extension.CodeType != "goext" {
				continue
			}
			url := strings.TrimPrefix(extension.URL, "file://")
			if url == "" {
				log.Warning(fmt.Sprintf("ignore go extension '%s' without plugin", extension.ID))
				continue
			}
			if err := env.Load(url); err != nil {
				return fmt.Errorf("failed to load binary: %s", err)
			}
		}
	}
	// setup time limits for matching extensions
	env.timeLimit = timeLimit
	for _, timeLimit := range timeLimits {
		if timeLimit.Match(path) {
			env.timeLimits = append(env.timeLimits, schema.NewEventTimeLimit(timeLimit.EventRegex, timeLimit.TimeDuration))
		}
	}

	if err := env.Start(); err != nil {
		log.Error("failed to start environment: %s", err)
		return err
	}
	return nil
}

func (env *Environment) dispatchSchemaEvent(prioritizedSchemaHandlers PrioritizedSchemaHandlers, sch Schema, event string, context map[string]interface{}) error {
	return dispatchSchemaEventForEnv(env, prioritizedSchemaHandlers, sch, event, context)
}

func dispatchSchemaEventForEnv(env IEnvironment, prioritizedSchemaHandlers PrioritizedSchemaHandlers, sch Schema, event string, context map[string]interface{}) error {
	env.Logger().Debugf("Starting event: %s, schema: %s", event, sch.raw.ID)
	defer env.Logger().Debugf("Finished event: %s, schema: %s", event, sch.raw.ID)
	var resource goext.Resource
	var err error
	if ctxResource, ok := context["resource"]; ok {
		if resource, err = sch.ResourceFromMap(ctxResource.(map[string]interface{})); err != nil {
			env.Logger().Warningf("failed to parse resource from context with schema '%s' for event '%s': %s", sch.ID(), event, err)
			return goext.NewErrorBadRequest(err)
		}
	}
	for _, priority := range sortSchemaHandlers(prioritizedSchemaHandlers) {
		for index, schemaEventHandler := range prioritizedSchemaHandlers[priority] {
			context["go_validation"] = true
			if err := schemaEventHandler(context, resource, env); err != nil {
				env.Logger().Warningf("failed to handle schema '%s' event '%s' at priority '%d' with index '%d': %s", sch.ID(), event, priority, index, err)
				return err
			}
			if resource != nil {
				context["resource"] = env.Util().ResourceToMap(resource)
			}
		}
	}
	return nil
}

func sortSchemaHandlers(schemaHandlers PrioritizedSchemaHandlers) (priorities []int) {
	priorities = []int{}
	for priority := range schemaHandlers {
		priorities = append(priorities, priority)
	}
	sort.Ints(priorities)
	return priorities
}

func sortHandlers(handlers PrioritizedHandlers) (priorities []int) {
	priorities = []int{}
	for priority := range handlers {
		priorities = append(priorities, priority)
	}
	sort.Ints(priorities)
	return priorities
}

func (env *Environment) getSchemaHandlers(event string) (SchemaPrioritizedSchemaHandlers, bool) {
	handler, ok := env.schemaHandlers[event]
	return handler, ok
}

func (env *Environment) getHandlers(event string) (PrioritizedHandlers, bool) {
	handler, ok := env.handlers[event]
	return handler, ok
}

func (env *Environment) getRawType(schemaID string) (reflect.Type, bool) {
	rawType, ok := env.rawTypes[schemaID]
	return rawType, ok
}

func (env *Environment) getType(schemaID string) (reflect.Type, bool) {
	resourceType, ok := env.types[schemaID]
	return resourceType, ok
}

func (env *Environment) getTraceID() string {
	return env.traceID
}

func (env *Environment) getTimeLimit() time.Duration {
	return env.timeLimit
}

func (env *Environment) getTimeLimits() []*schema.EventTimeLimit {
	return env.timeLimits
}

// HandleEvent handles an event
func (env *Environment) HandleEvent(event string, context map[string]interface{}) error {
	err := handleEventForEnv(env, event, context)
	if err != nil {
		env.dumpErrorToLog(err)
	}
	return err
}

func (env *Environment) dumpErrorToLog(err error) {
	switch err.(type) {
	case *goext.Error:
		env.Logger().Warningf("Error:\n%s", err.(*goext.Error).ErrorStack())
	default:
		env.Logger().Warningf("Error: %s", err)
	}
}

func handleEventForEnv(env IEnvironment, event string, requestContext map[string]interface{}) error {
	if !hasCancel(requestContext) {
		done := make(chan bool, 1)
		addCancel(env, event, requestContext, done)

		defer func() {
			done <- true
			delete(requestContext, "context")
		}()
	}

	requestContext["event_type"] = event

	// dispatch to schema handlers
	if schemaPrioritizedSchemaHandlers, ok := env.getSchemaHandlers(event); ok {
		if iSchemaID, ok := requestContext["schema_id"]; ok {
			schemaID := iSchemaID.(string)
			if prioritizedSchemaHandlers, ok := schemaPrioritizedSchemaHandlers[schemaID]; ok {
				if iSchema := env.Schemas().Find(schemaID); iSchema != nil {
					sch := iSchema.(*Schema)
					if err := env.dispatchSchemaEvent(prioritizedSchemaHandlers, *sch, event, requestContext); err != nil {
						return err
					}
				}
			}
		} else {
			// all
			for schemaID, prioritizedSchemaHandlers := range schemaPrioritizedSchemaHandlers {
				if iSchema := env.Schemas().Find(schemaID); iSchema != nil {
					sch := iSchema.(*Schema)
					if err := env.dispatchSchemaEvent(prioritizedSchemaHandlers, *sch, event, requestContext); err != nil {
						return err
					}
				} else {
					return goext.NewErrorInternalServerError(fmt.Errorf("could not find schema: %s", schemaID))
				}
			}
		}
	}

	// dispatch to generic handlers
	if prioritizedEventHandlers, ok := env.getHandlers(event); ok {
		for _, priority := range sortHandlers(prioritizedEventHandlers) {
			for index, eventHandler := range prioritizedEventHandlers[priority] {
				if err := eventHandler(requestContext, env); err != nil {
					env.Logger().Warningf("failed to handle event '%s' at priority '%d' with index '%d': %s", event, priority, index, err)
					return err
				}
			}
		}
	}

	return nil
}

func hasCancel(requestContext map[string]interface{}) bool {
	_, cancelFound := requestContext["context"]
	return cancelFound
}

func addCancel(env IEnvironment, event string, requestContext map[string]interface{}, done <-chan bool) {
	ctx, cancel := buildCancel(env, event)
	cancelOnPeerDisconnect(requestContext, cancel, done)
	requestContext["context"] = ctx
}

func cancelOnPeerDisconnect(requestContext map[string]interface{}, cancel context.CancelFunc, done <-chan bool) {
	closeNotify := getCloseChannel(requestContext)
	go func() {
		select {
		case <-closeNotify:
			cancel()
		case <-done:
			return
		}
	}()
}

func getCloseChannel(requestContext map[string]interface{}) <-chan bool {
	var closeNotifier http.CloseNotifier
	var closeNotify <-chan bool
	if httpResponse, ok := requestContext["http_response"]; ok {
		if closeNotifier, ok = httpResponse.(http.CloseNotifier); ok {
			closeNotify = closeNotifier.CloseNotify()
		}
	}
	return closeNotify
}

func buildCancel(env IEnvironment, event string) (ctx context.Context, cancel context.CancelFunc) {
	selectedTimeLimit := env.getTimeLimit()
	for _, timeLimit := range env.getTimeLimits() {
		if timeLimit.Match(event) {
			selectedTimeLimit = timeLimit.TimeDuration
			break
		}
	}
	if selectedTimeLimit > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), selectedTimeLimit)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	return ctx, cancel
}

// RegisterEventHandler registers an event handler
func (env *Environment) RegisterEventHandler(event string, handler goext.Handler, priority int) {
	if env.handlers == nil {
		env.handlers = EventPrioritizedHandlers{}
	}

	if env.handlers[event] == nil {
		env.handlers[event] = PrioritizedHandlers{}
	}

	if env.handlers[event][priority] == nil {
		env.handlers[event][priority] = Handlers{}
	}

	env.handlers[event][priority] = append(env.handlers[event][priority], handler)
}

// RegisterSchemaEventHandler register an event handler for a schema
func (env *Environment) RegisterSchemaEventHandler(schemaID string, event string, schemaHandler goext.SchemaHandler, priority int) {
	if env.schemaHandlers == nil {
		env.schemaHandlers = EventSchemaPrioritizedSchemaHandlers{}
	}

	if env.schemaHandlers[event] == nil {
		env.schemaHandlers[event] = SchemaPrioritizedSchemaHandlers{}
	}

	if env.schemaHandlers[event][schemaID] == nil {
		env.schemaHandlers[event][schemaID] = PrioritizedSchemaHandlers{}
	}

	if env.schemaHandlers[event][schemaID][priority] == nil {
		env.schemaHandlers[event][schemaID][priority] = SchemaHandlers{}
	}

	env.schemaHandlers[event][schemaID][priority] = append(env.schemaHandlers[event][schemaID][priority], schemaHandler)
}

// RegisterRawType registers a runtime type of raw resource for a given name
func (env *Environment) RegisterRawType(name string, typeValue interface{}) {
	targetType := reflect.TypeOf(typeValue)
	env.rawTypes[name] = targetType
}

// RegisterType registers a runtime type of resource for a given name
func (env *Environment) RegisterType(name string, typeValue interface{}) {
	targetType := reflect.TypeOf(typeValue)
	env.types[name] = targetType
}

// Stop stops the environment to its initial state
func (env *Environment) Stop() {
	log.Info("Stop environment")

	// reset locals
	env.coreImpl = nil
	env.loggerImpl = nil
	env.schemasImpl = nil
	env.syncImpl = nil
	env.databaseImpl = nil

	env.handlers = nil
	env.schemaHandlers = nil

	env.traceID = ""

	env.rawTypes = make(map[string]reflect.Type)
	env.types = make(map[string]reflect.Type)

	// after stop
	if env.afterStopHook != nil {
		log.Debug("Calling after stop hook for go environment: %s", env.name)
		env.afterStopHook(env)
	} else {
		log.Debug("After stop hook is not set for go environment: %s", env.name)
	}
}

// Reset clear the environment to its initial state
func (env *Environment) Reset() {
	log.Info("Reset environment")

	env.Stop()
	env.Start()
}

func newTraceID() string {
	return uuid.NewV4().String()
}

// Clone makes a clone of the rawEnvironment
func (env *Environment) Clone() extension.Environment {
	clone := &Environment{
		initFns:         env.initFns,
		beforeStartHook: env.beforeStartHook,
		afterStopHook:   env.afterStopHook,

		syncImpl:     env.syncImpl.Clone(),
		databaseImpl: env.databaseImpl.Clone(),

		name:    env.name,
		traceID: newTraceID(),

		handlers:       deepcopy.Copy(env.handlers).(EventPrioritizedHandlers),
		schemaHandlers: deepcopy.Copy(env.schemaHandlers).(EventSchemaPrioritizedSchemaHandlers),

		rawTypes: make(map[string]reflect.Type),
		types:    make(map[string]reflect.Type),
	}

	clone.loggerImpl = env.loggerImpl.Clone(clone)
	clone.coreImpl = env.coreImpl.Clone(clone)
	clone.schemasImpl = env.schemasImpl.Clone(clone)

	for k, v := range env.rawTypes {
		clone.rawTypes[k] = v
	}
	for k, v := range env.types {
		clone.types[k] = v
	}
	return clone
}

// IsEventHandled returns whether a given event is handled by this environment
func (env *Environment) IsEventHandled(event string, context map[string]interface{}) bool {
	return true
}
