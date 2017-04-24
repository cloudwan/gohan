// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

type Context map[string]interface{}
type Resource interface{}
type Resources []Resource
type Priority int

const PRIORITY_DEFAULT Priority = 0

type Schema interface {
	EnvContainer

	// properties
	ID() string

	// database
	List(resources interface{}) error
	Fetch(id string, resource interface{}) error
	FetchRelated(resource interface{}, relatedResource interface{}) error
	Create(resource interface{}) error
	Update(resource interface{}) error
	Delete(id string) error

	RegisterEventHandler(eventName string, handler func(context Context, resource Resource, environment *Environment) error, priority Priority)
	RegisterResourceType(typeValue interface{})
}

type SchemasInterface interface {
	EnvContainer

	List() []Schema
	Find(id string) Schema
}
