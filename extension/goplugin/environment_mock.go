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
	"reflect"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/schema"
	"github.com/golang/mock/gomock"
)

type MockIEnvironment struct {
	env          *Environment
	mockModules  goext.MockModules
	testReporter gomock.TestReporter
	ctrl         *gomock.Controller

	core     goext.ICore
	logger   goext.ILogger
	schemas  goext.ISchemas
	sync     goext.ISync
	database goext.IDatabase
	http     goext.IHTTP
	auth     goext.IAuth
	config   goext.IConfig
	util     goext.IUtil
}

func (mockEnv *MockIEnvironment) setModules() {
	mockEnv.mockModules = goext.MockModules{}
	mockEnv.core = mockEnv.env.Core()
	mockEnv.logger = mockEnv.env.Logger()
	mockEnv.schemas = mockEnv.env.Schemas()
	mockEnv.sync = mockEnv.env.Sync()
	mockEnv.database = mockEnv.env.Database()
	mockEnv.http = mockEnv.env.HTTP()
	mockEnv.auth = mockEnv.env.Auth()
	mockEnv.config = mockEnv.env.Config()
	mockEnv.util = mockEnv.env.Util()
}

func (mockEnv *MockIEnvironment) GetController() *gomock.Controller {
	return mockEnv.ctrl
}

func (mockEnv *MockIEnvironment) SetMockModules(modules goext.MockModules) {
	mockEnv.mockModules = modules

	if mockEnv.mockModules.Core {
		mockEnv.core = goext.NewMockICore(mockEnv.ctrl)
	}

	if mockEnv.mockModules.Logger {
		mockEnv.logger = goext.NewMockILogger(mockEnv.ctrl)
	}

	if mockEnv.mockModules.Schemas {
		mockEnv.schemas = goext.NewMockISchemas(mockEnv.ctrl)
	}

	if mockEnv.mockModules.Sync {
		mockEnv.sync = goext.NewMockISync(mockEnv.ctrl)
	}

	if mockEnv.mockModules.Database {
		mockEnv.database = goext.NewMockIDatabase(mockEnv.ctrl)
		if mockEnv.mockModules.DefaultDatabase {
			setupDefaultMockDatabase(mockEnv.database.(*goext.MockIDatabase))
		}
	}

	if mockEnv.mockModules.Http {
		mockEnv.http = goext.NewMockIHTTP(mockEnv.ctrl)
	}

	if mockEnv.mockModules.Auth {
		mockEnv.auth = goext.NewMockIAuth(mockEnv.ctrl)
	}

	if mockEnv.mockModules.Config {
		mockEnv.config = goext.NewMockIConfig(mockEnv.ctrl)
	}

	if mockEnv.mockModules.Util {
		mockEnv.util = goext.NewMockIUtil(mockEnv.ctrl)
	}
}

func (mockEnv *MockIEnvironment) Core() goext.ICore {
	return mockEnv.core
}

func (mockEnv *MockIEnvironment) Logger() goext.ILogger {
	return mockEnv.logger
}

func (mockEnv *MockIEnvironment) Schemas() goext.ISchemas {
	return mockEnv.schemas
}

func (mockEnv *MockIEnvironment) Sync() goext.ISync {
	return mockEnv.sync
}

func (mockEnv *MockIEnvironment) Database() goext.IDatabase {
	return mockEnv.database
}

func (mockEnv *MockIEnvironment) HTTP() goext.IHTTP {
	return mockEnv.http
}

func (mockEnv *MockIEnvironment) Auth() goext.IAuth {
	return mockEnv.auth
}

func (mockEnv *MockIEnvironment) Config() goext.IConfig {
	return mockEnv.config
}

func (mockEnv *MockIEnvironment) Util() goext.IUtil {
	return mockEnv.util
}

func (mockEnv *MockIEnvironment) MockCore() *goext.MockICore {
	return mockEnv.core.(*goext.MockICore)
}

func (mockEnv *MockIEnvironment) MockLogger() *goext.MockILogger {
	return mockEnv.logger.(*goext.MockILogger)
}

func (mockEnv *MockIEnvironment) MockSchemas() *goext.MockISchemas {
	return mockEnv.schemas.(*goext.MockISchemas)
}

func (mockEnv *MockIEnvironment) MockSync() *goext.MockISync {
	return mockEnv.sync.(*goext.MockISync)
}

func (mockEnv *MockIEnvironment) MockDatabase() *goext.MockIDatabase {
	return mockEnv.database.(*goext.MockIDatabase)
}

func (mockEnv *MockIEnvironment) MockHttp() *goext.MockIHTTP {
	return mockEnv.http.(*goext.MockIHTTP)
}

func (mockEnv *MockIEnvironment) MockAuth() *goext.MockIAuth {
	return mockEnv.auth.(*goext.MockIAuth)
}

func (mockEnv *MockIEnvironment) MockConfig() *goext.MockIConfig {
	return mockEnv.config.(*goext.MockIConfig)
}

func (mockEnv *MockIEnvironment) MockUtil() *goext.MockIUtil {
	return mockEnv.util.(*goext.MockIUtil)
}

func (mockEnv *MockIEnvironment) Reset() {
	mockEnv.setModules()
	mockEnv.env.Reset()
	mockEnv.env.bindSchemasToEnv(mockEnv)
	mockEnv.ctrl = NewController(mockEnv.testReporter)
}

func (mockEnv *MockIEnvironment) Clone() extension.Environment {
	return mockEnv
}

func (mockEnv *MockIEnvironment) HandleEvent(event string, context map[string]interface{}) error {
	return handleEventForEnv(mockEnv, event, context)
}

func (mockEnv *MockIEnvironment) getSchemaHandlers(event string) (SchemaPrioritizedSchemaHandlers, bool) {
	return mockEnv.env.getSchemaHandlers(event)
}

func (mockEnv *MockIEnvironment) getHandlers(event string) (PrioritizedHandlers, bool) {
	return mockEnv.env.getHandlers(event)
}

func (mockEnv *MockIEnvironment) dispatchSchemaEvent(prioritizedSchemaHandlers PrioritizedSchemaHandlers, sch Schema, event string, context map[string]interface{}) error {
	return dispatchSchemaEventForEnv(mockEnv, prioritizedSchemaHandlers, sch, event, context)
}

func (mockEnv *MockIEnvironment) RegisterRawType(name goext.SchemaID, typeValue interface{}) {
	mockEnv.env.RegisterRawType(name, typeValue)
}

func (mockEnv *MockIEnvironment) RegisterType(name goext.SchemaID, typeValue interface{}) {
	mockEnv.env.RegisterType(name, typeValue)
}

func (mockEnv *MockIEnvironment) RegisterSchemaEventHandler(schemaID goext.SchemaID, event string, schemaHandler goext.SchemaHandler, priority int) {
	mockEnv.env.RegisterSchemaEventHandler(schemaID, event, schemaHandler, priority)
}

func (mockEnv *MockIEnvironment) getRawType(schemaID goext.SchemaID) (reflect.Type, bool) {
	return mockEnv.env.getRawType(schemaID)
}

func (mockEnv *MockIEnvironment) getType(schemaID goext.SchemaID) (reflect.Type, bool) {
	return mockEnv.env.getType(schemaID)
}

func (mockEnv *MockIEnvironment) getTraceID() string {
	return mockEnv.env.getTraceID()
}

func (mockEnv *MockIEnvironment) getTimeLimit() time.Duration {
	return mockEnv.env.timeLimit
}

func (mockEnv *MockIEnvironment) getTimeLimits() []*schema.EventTimeLimit {
	return mockEnv.env.timeLimits
}

func (mockEnv *MockIEnvironment) IsEventHandled(event string, context map[string]interface{}) bool {
	return mockEnv.env.IsEventHandled(event, context)
}

func (mockEnv *MockIEnvironment) LoadExtensionsForPath(extensions []*schema.Extension, timeLimit time.Duration, timeLimits []*schema.PathEventTimeLimit, path string) error {
	return mockEnv.env.LoadExtensionsForPath(extensions, timeLimit, timeLimits, path)
}

func NewMockIEnvironment(env *Environment, testReporter gomock.TestReporter) *MockIEnvironment {
	mockIEnvironment := &MockIEnvironment{env: env, testReporter: testReporter, ctrl: NewController(testReporter)}
	return mockIEnvironment
}

func setupDefaultMockDatabase(m *goext.MockIDatabase) {
	m.EXPECT().Within(gomock.Any(), gomock.Any()).DoAndReturn(
		func(
			context goext.Context,
			fn func(tx goext.ITransaction) error,
		) error {
			return within(m.Options(), context, fn, func() (db.ITransaction, error) {
				return m.Begin()
			})
		}).AnyTimes()

	m.EXPECT().WithinTx(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(
			context goext.Context,
			options *goext.TxOptions,
			fn func(tx goext.ITransaction) error,
		) error {
			return within(m.Options(), context, fn, func() (db.ITransaction, error) {
				return m.BeginTx(context, options)
			})
		}).AnyTimes()
}
