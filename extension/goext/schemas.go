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

import "errors"

// LockPolicy indicates lock policy
type LockPolicy int

const (
	// LockRelatedResources indicates that related resources are also locked
	LockRelatedResources LockPolicy = iota

	// SkipRelatedResources indicates that related resources are not locked
	SkipRelatedResources

	// NoLock indicates that no lock is acquired at all
	NoLock
)

// Resource represents a resource
type Resource interface{}

// Resources is a list of resources
type Resources []Resource

// Context represents a context of a handler
type Context map[string]interface{}

// Filter represents filtering options for fetching functions
type Filter map[string]interface{}

// Paginator represents a paginator
type Paginator struct {
	Key    string
	Order  string
	Limit  uint64
	Offset uint64
}

// MakeContext creates an empty context
func MakeContext() Context {
	return make(map[string]interface{})
}

// WithSchemaID appends schema ID to given context
func (ctx Context) WithSchemaID(schemaID string) Context {
	ctx["schema_id"] = schemaID
	return ctx
}

// WithISchema appends ISchema to given context
func (ctx Context) WithISchema(schema ISchema) Context {
	ctx["schema"] = schema
	return ctx
}

// WithResource appends resource to given context
func (ctx Context) WithResource(resource Resource) Context {
	ctx["resource"] = resource
	return ctx
}

// WithResourceID appends resource ID to given context
func (ctx Context) WithResourceID(resourceID string) Context {
	ctx["id"] = resourceID
	return ctx
}

// WithTransaction appends transaction to given context
func (ctx Context) WithTransaction(tx ITransaction) Context {
	ctx["transaction"] = tx
	return ctx
}

// Clone returns copy of context
func (ctx Context) Clone() Context {
	contextCopy := MakeContext()
	for k, v := range ctx {
		contextCopy[k] = v
	}
	return contextCopy
}

// PriorityDefault is a default handler priority
const PriorityDefault = 0

// ErrResourceNotFound represents 'resource not found' error
var ErrResourceNotFound = errors.New("resource not found")

// ISchema is an interface representing a single schema in Gohan
type ISchema interface {
	// ID returns the identifier of this resource
	ID() string

	// List returns a list of pointers to resources derived from BaseResource
	List(filter Filter, paginator *Paginator, context Context) ([]interface{}, error)

	// ListRaw returns a list of pointers to raw resources, containing db annotations
	ListRaw(filter Filter, paginator *Paginator, context Context) ([]interface{}, error)

	// LockList returns a list of pointers to locked resources derived from BaseResource
	LockList(filter Filter, paginator *Paginator, context Context, lockPolicy LockPolicy) ([]interface{}, error)

	// LockListRaw returns a list of pointers to locked raw resources, containing db annotations
	LockListRaw(filter Filter, paginator *Paginator, context Context, lockPolicy LockPolicy) ([]interface{}, error)

	// Fetch returns a pointer to resource derived from BaseResource
	Fetch(id string, context Context) (interface{}, error)

	// ListRaw returns a pointer to raw resource, containing db annotations
	FetchRaw(id string, context Context) (interface{}, error)

	// LockFetch returns a pointer to locked resource derived from BaseResource, containing db annotations
	LockFetch(id string, context Context, lockPolicy LockPolicy) (interface{}, error)

	// LockFetchRaw returns a pointer to locked raw resource, containing db annotations
	LockFetchRaw(id string, context Context, lockPolicy LockPolicy) (interface{}, error)

	// CreateRaw creates a raw resource, given by a pointer
	CreateRaw(rawResource interface{}, context Context) error

	// DbCreateRaw creates a raw resource, given by a pointer, no events are emitted
	DbCreateRaw(rawResource interface{}, context Context) error

	// UpdateRaw updates a raw resource, given by a pointer
	UpdateRaw(rawResource interface{}, context Context) error

	// DbUpdateRaw updates a raw resource, given by a pointer, no events are emitted
	DbUpdateRaw(rawResource interface{}, context Context) error

	// DeleteRaw deletes a raw resource, given by a pointer
	DeleteRaw(filter Filter, context Context) error

	// DbDeleteRaw deletes a raw resource, given by a pointer, no events are emitted
	DbDeleteRaw(filter Filter, context Context) error

	// RegisterEventHandler registers an event handler for a named event with given priority
	RegisterEventHandler(event string, handler func(context Context, resource Resource, environment IEnvironment) error, priority int)

	// RegisterType registers a resource type, derived from BaseResource
	RegisterType(resourceType interface{})

	// RegisterRawType registers a raw resource type, containing db annotations
	RegisterRawType(rawResourceType interface{})
}

// ISchemas is an interface to schemas manager in Gohan
type ISchemas interface {
	List() []ISchema
	Find(id string) ISchema
}
