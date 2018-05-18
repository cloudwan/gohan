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

	// Config returns an implementation of IConfig interface
	Config() IConfig
	// Core returns an implementation of ICore interface
	Core() ICore
	// Logger returns an implementation of ILogger interface
	Logger() ILogger
	// Schemas returns an implementation of ISchemas interface
	Schemas() ISchemas
	// Sync returns an implementation of ISync interface
	Sync() ISync
	// Database returns an implementation of IIDatabase interface
	Database() IDatabase
	// HTTP returns an implementation of IIHTTP interface
	HTTP() IHTTP
	// Auth returns an implementation of IIAuth interface
	Auth() IAuth
	// Util returns an implementation of IIUtil interface
	Util() IUtil

	// state

	// Reset clears the environment to its initial state
	Reset()
}

// ResourceBase is the implementation of base class for all resources
type ResourceBase struct {
	environment IEnvironment
	schema      ISchema
	logger      ILogger
}

// Environment returns an implementation of IEnvironment interface
func (resourceBase *ResourceBase) Environment() IEnvironment {
	return resourceBase.environment
}

// Logger returns an implementation of ILogger interface
func (resourceBase *ResourceBase) Logger() ILogger {
	return resourceBase.logger
}

// Schema returns an implementation of ISchema interface
func (resourceBase *ResourceBase) Schema() ISchema {
	return resourceBase.schema
}

// NewResourceBase allocates a new ResourceBase with the given environment, schema and logger
func NewResourceBase(env IEnvironment, schema ISchema, logger ILogger) *ResourceBase {
	return &ResourceBase{
		environment: env,
		schema:      schema,
		logger:      logger,
	}
}

// IResourceBase is the base class for all resources
type IResourceBase interface {
	// Environment returns an implementation of IEnvironment interface
	Environment() IEnvironment
	// Schema returns an implementation of ISchema interface
	Schema() ISchema
	// Logger returns an implementation of ILogger interface
	Logger() ILogger
}
