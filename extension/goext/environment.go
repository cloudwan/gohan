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

// IEnvironment is the only scope of Gohan available for a go extensions;
// other packages must not be imported nor used
type IEnvironment interface {
	// modules
	Config() IConfig
	Core() ICore
	Logger() ILogger
	Schemas() ISchemas
	Sync() ISync
	Database() IDatabase
	HTTP() IHTTP
	Auth() IAuth
	Util() IUtil

	// state
	Reset()
}

// ResourceBase is the implementation of base class for all resources
type ResourceBase struct {
	environment IEnvironment
	schema      ISchema
	logger      ILogger
}

func (resourceBase *ResourceBase) Environment() IEnvironment {
	return resourceBase.environment
}

func (resourceBase *ResourceBase) Logger() ILogger {
	return resourceBase.logger
}

func (resourceBase *ResourceBase) Schema() ISchema {
	return resourceBase.schema
}

func NewResourceBase(env IEnvironment, schema ISchema, logger ILogger) *ResourceBase {
	return &ResourceBase{
		environment: env,
		schema:      schema,
		logger:      logger,
	}
}

// IResourceBase is the base class for all resources
type IResourceBase interface {
	Environment() IEnvironment
	Schema() ISchema
	Logger() ILogger
}
