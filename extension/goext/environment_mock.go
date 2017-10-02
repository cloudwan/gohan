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

// MockModules indicates modules which should be mocked.
// By default none of the modules are mocked so that MockIEnvironment behaves exactly the same as IEnvironment.
type MockModules struct {
	Core, Logger, Schemas, Sync, Database, Http, Auth, Util, Config bool
}

// MockIEnvironment is the only scope of Gohan available for a go unit tests extensions;
// it should not be used outside unit tests
type MockIEnvironment interface {
	IEnvironment
	SetMockModules(modules MockModules)
	MockCore() *MockICore
	MockLogger() *MockILogger
	MockSchemas() *MockISchemas
	MockSync() *MockISync
	MockDatabase() *MockIDatabase
	MockHttp() *MockIHTTP
	MockAuth() *MockIAuth
	MockConfig() *MockIConfig
	MockUtil() *MockIUtil
}
