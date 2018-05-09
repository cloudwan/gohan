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

import (
	"context"
	"errors"
)

// SchemaID is a type for schema ID
type SchemaID string

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
	Limit  MaybeInt
	Offset uint64
}

// MakeContext creates an empty context
func MakeContext() Context {
	return make(map[string]interface{})
}

// WithSchemaID appends schema ID to given context
func (ctx Context) WithSchemaID(schemaID SchemaID) Context {
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

func GetContext(requestContext Context) context.Context {
	if rawCtx, hasCtx := requestContext["context"]; hasCtx {
		return rawCtx.(context.Context)
	} else {
		return context.Background()
	}
}

// PriorityDefault is a default handler priority
const PriorityDefault = 0

// ErrResourceNotFound represents 'resource not found' error
var ErrResourceNotFound = errors.New("resource not found")

// ISchema is an interface representing a single schema in Gohan
type ISchema interface {
	// ID returns the identifier of this resource
	ID() SchemaID

	// List returns a list of pointers to resources derived from BaseResource
	List(filter Filter, paginator *Paginator, context Context) ([]interface{}, error)

	// ListRaw returns a list of pointers to raw resources, containing db annotations
	ListRaw(filter Filter, paginator *Paginator, context Context) ([]interface{}, error)

	// LockList returns a list of pointers to locked resources derived from BaseResource
	LockList(filter Filter, paginator *Paginator, context Context, lockPolicy LockPolicy) ([]interface{}, error)

	// LockListRaw returns a list of pointers to locked raw resources, containing db annotations
	LockListRaw(filter Filter, paginator *Paginator, context Context, lockPolicy LockPolicy) ([]interface{}, error)

	// Count returns number of resources matching the filter
	Count(filter Filter, context Context) (uint64, error)

	// Fetch returns a pointer to resource derived from BaseResource
	Fetch(id string, context Context) (interface{}, error)

	// FetchRaw returns a pointer to raw resource, containing db annotations
	FetchRaw(id string, context Context) (interface{}, error)

	// FetchFilter returns a pointer to resource derived from BaseResource
	FetchFilter(filter Filter, context Context) (interface{}, error)

	// FetchFilterRaw returns a pointer to raw resource, containing db annotations
	FetchFilterRaw(filter Filter, context Context) (interface{}, error)

	// StateFetchRaw returns a resource state
	StateFetchRaw(id string, requestContext Context) (ResourceState, error)

	// LockFetch returns a pointer to locked resource derived from BaseResource, containing db annotations
	LockFetch(id string, context Context, lockPolicy LockPolicy) (interface{}, error)

	// LockFetchRaw returns a pointer to locked raw resource, containing db annotations
	LockFetchRaw(id string, context Context, lockPolicy LockPolicy) (interface{}, error)

	// LockFetchFilter returns a pointer to locked resource derived from BaseResource, containing db annotations
	LockFetchFilter(filter Filter, context Context, lockPolicy LockPolicy) (interface{}, error)

	// LockFetchFilterRaw returns a pointer to locked raw resource, containing db annotations
	LockFetchFilterRaw(filter Filter, context Context, lockPolicy LockPolicy) (interface{}, error)

	// CreateRaw creates a raw resource, given by a pointer
	CreateRaw(rawResource interface{}, context Context) error

	// DbCreateRaw creates a raw resource, given by a pointer, no events are emitted
	DbCreateRaw(rawResource interface{}, context Context) error

	// UpdateRaw updates a raw resource, given by a pointer
	UpdateRaw(rawResource interface{}, context Context) error

	// DbUpdateRaw updates a raw resource, given by a pointer, no events are emitted
	DbUpdateRaw(rawResource interface{}, context Context) error

	// DbStateUpdateRaw updates state of a raw resource
	DbStateUpdateRaw(rawResource interface{}, context Context, state *ResourceState) error

	// DeleteRaw deletes a raw resource, given by id
	DeleteRaw(id string, context Context) error

	// DbDeleteRaw deletes a raw resource, given by id, no events are emitted
	DbDeleteRaw(id string, context Context) error

	// DeleteFilterRaw deletes a raw resource, given by a filter
	DeleteFilterRaw(filter Filter, context Context) error

	// DbDeleteFilterRaw deletes a raw resource, given by a filter, no events are emitted
	DbDeleteFilterRaw(filter Filter, context Context) error

	// RegisterResourceEventHandler registers an event handler with resource for a named event with given priority
	RegisterResourceEventHandler(event ResourceEvent, schemaHandler SchemaHandler, priority int)

	// RegisterCustomEventHandler registers an event handler without resource for a custom event with given priority
	RegisterCustomEventHandler(event CustomEvent, handler Handler, priority int)

	// RegisterType registers a resource type, derived from IResourceBase
	//
	// Deprecated: use RegisterTypes instead
	RegisterType(resourceType IResourceBase)

	// RegisterRawType registers a raw resource type, containing db annotations
	//
	// Deprecated: use RegisterTypes instead
	RegisterRawType(rawResourceType interface{})

	// RegisterTypes registers both resource types derived from IResourceBase and raw containing db annotations
	RegisterTypes(rawResourceType interface{}, resourceType IResourceBase)

	// ResourceFromMap converts mapped representation to structure representation of the raw resource registered for schema
	ResourceFromMap(context map[string]interface{}) (Resource, error)

	RawSchema() interface{}

	// DerivedSchemas returns list of schemas that extend schema with given id
	DerivedSchemas() []ISchema

	// ColumnNames generates an array that has Gohan style column names
	ColumnNames() []string

	// Properties returns properties of schema
	Properties() []Property

	// Extends return list of schema_ids which given schema extends
	Extends() []SchemaID
}

// Property represents schema property
type Property struct {
	ID       string
	Title    string
	Relation SchemaID
	Type     string
}

// SchemaRelationInfo describes schema relation
type SchemaRelationInfo struct {
	// SchemaID relation to which schema
	SchemaID SchemaID
	// PropertyID ID of property which relation is referenced
	PropertyID string
	// OnDeleteCascade whether cascading delete on related resource delete is enabled
	OnDeleteCascade bool
}

// ISchemas is an interface to schemas manager in Gohan
type ISchemas interface {
	List() []ISchema
	Find(id SchemaID) ISchema

	// Relations returns list of information about schema relations
	Relations(id SchemaID) []SchemaRelationInfo
}
