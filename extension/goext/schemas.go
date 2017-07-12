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

type Resource interface{}
type Resources []Resource

type Context map[string]interface{}

func MakeContext() (ctx Context) {
	return make(map[string]interface{})
}

func (ctx Context) WithSchemaID(schemaId string) Context {
	ctx["schema_id"] = schemaId
	return ctx
}

func (ctx Context) WithResource(resource Resource) Context {
	ctx["resource"] = resource
	return ctx
}

func (ctx Context) WithResourceID(resourceID string) Context {
	ctx["id"] = resourceID
	return ctx
}

type Priority int

const PriorityDefault Priority = 0

// ISchema is an interface representing a single schema in Gohan
type ISchema interface {
	IEnvironmentSupport

	// properties
	ID() string

	// database
	List(resources interface{}) error
	Fetch(id string, resource interface{}) error
	FetchRelated(resource interface{}, relatedResource interface{}) error
	Create(resource interface{}) error
	Update(resource interface{}) error
	Delete(resourceID string) error

	// events
	RegisterEventHandler(event string, handler func(context Context, resource Resource, environment IEnvironment) error, priority Priority)
	RegisterResourceType(typeValue interface{})
}

// ISchemas is an interface to schemas manager in Gohan
type ISchemas interface {
	IEnvironmentSupport

	List() []ISchema
	Find(id string) ISchema
}
