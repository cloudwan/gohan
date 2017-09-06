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

// ICore is an interface to core parts of Gohan: event triggering and registering
type ICore interface {
	NewUUID() string

	TriggerEvent(event string, context Context) error
	HandleEvent(event string, context Context) error

	RegisterEventHandler(eventName string, handler func(context Context, environment IEnvironment) error, priority Priority)
	RegisterSchemaEventHandler(schemaID string, eventName string, handler func(context Context, resource Resource, environment IEnvironment) error, priority Priority)
}
