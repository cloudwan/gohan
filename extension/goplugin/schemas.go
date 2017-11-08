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
	"reflect"

	"github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	gohan_schema "github.com/cloudwan/gohan/schema"
	"github.com/jmoiron/sqlx/reflectx"
)

var (
	// ErrNotPointer indicates that a resource was not passed by a pointer
	ErrNotPointer = fmt.Errorf("raw resource must be passed by a pointer")
)

func makeErrMissingType(missingType string) error {
	return fmt.Errorf("resource type '%s' not registered", missingType)
}

func isPointer(resource interface{}) bool {
	rv := reflect.ValueOf(resource)
	return rv.Kind() == reflect.Ptr
}

// Schemas in an implementation of ISchemas
type Schemas struct {
	env IEnvironment
}

// List returns a list of loaded schemas
func (schemas *Schemas) List() []goext.ISchema {
	manager := gohan_schema.GetManager()
	result := []goext.ISchema{}
	for _, raw := range manager.OrderedSchemas() {
		result = append(result, NewSchema(schemas.env, raw))
	}
	return result
}

func (schemas *Schemas) Relations(id string) []goext.SchemaRelationInfo {
	manager := gohan_schema.GetManager()
	relations := map[string][]goext.SchemaRelationInfo{}

	for _, schema := range manager.OrderedSchemas() {
		for _, property := range schema.Properties {
			if property.Relation != "" {
				if _, ok := relations[property.Relation]; !ok {
					relations[property.Relation] = []goext.SchemaRelationInfo{}
				}

				onDeleteCascade := property.OnDeleteCascade
				if schema.Parent != "" && schema.Parent == property.Relation {
					onDeleteCascade = schema.OnParentDeleteCascade
				}

				relations[property.Relation] = append(relations[property.Relation], goext.SchemaRelationInfo{
					SchemaID:        schema.ID,
					PropertyID:      property.ID,
					OnDeleteCascade: onDeleteCascade,
				})
			}
		}
	}

	return relations[id]
}

// Find returns a schema by id or nil if not found
func (schemas *Schemas) Find(id string) goext.ISchema {
	manager := gohan_schema.GetManager()
	sch, ok := manager.Schema(id)

	if !ok {
		log.Warning(fmt.Sprintf("cannot find schema: %s", id))
		return nil
	}

	return NewSchema(schemas.env, sch)
}

// NewSchemas allocates a new Schemas
func NewSchemas(env IEnvironment) *Schemas {
	return &Schemas{env: env}
}

// Clone allocates a clone of Schemas; object may be nil
func (schemas *Schemas) Clone(env *Environment) *Schemas {
	if schemas == nil {
		return nil
	}
	return &Schemas{
		env: env,
	}
}

// Schema is an implementation of ISchema
type Schema struct {
	env IEnvironment
	raw *gohan_schema.Schema
}

// ID returns ID of schema
func (schema *Schema) ID() string {
	return schema.raw.ID
}

// ResourceFromMap converts mapped representation to structure representation of the resource registered for schema
func (schema *Schema) ResourceFromMap(context map[string]interface{}) (goext.Resource, error) {
	rawType, ok := schema.env.getRawType(schema.ID())

	if !ok {
		schema.env.Logger().Warningf("Raw resource type not registered for %s", schema.ID())
		return nil, makeErrMissingType(schema.ID())
	}

	resource := reflect.New(rawType)
	if err := resourceFromMap(context, resource); err != nil {
		return nil, err
	}
	return resource.Interface(), nil
}

func (schema *Schema) structToResource(resource interface{}) (*gohan_schema.Resource, error) {
	fieldsMap := schema.env.Util().ResourceToMap(resource)
	return gohan_schema.NewResource(schema.raw, fieldsMap)
}

// ListRaw lists schema raw resources
func (schema *Schema) ListRaw(filter goext.Filter, paginator *goext.Paginator, requestContext goext.Context) ([]interface{}, error) {
	return schema.listImpl(requestContext, func(ctx context.Context, tx goext.ITransaction) ([]map[string]interface{}, uint64, error) {
		return tx.List(ctx, schema, filter, nil, paginator)
	})
}

type listFunc func(ctx context.Context, tx goext.ITransaction) ([]map[string]interface{}, uint64, error)

func (schema *Schema) listImpl(requestContext goext.Context, list listFunc) ([]interface{}, error) {
	tx := mustGetOpenTransactionFromContext(requestContext)

	data, _, err := list(goext.GetContext(requestContext), tx)

	if err != nil {
		return nil, err
	}

	res := make([]interface{}, len(data), len(data))

	for i := 0; i < len(data); i++ {
		resource, err := schema.ResourceFromMap(data[i])
		if err != nil {
			return nil, err
		}
		res[i] = resource
	}

	return res, nil
}

// LockListRaw locks and returns raw resources
func (schema *Schema) LockListRaw(filter goext.Filter, paginator *goext.Paginator, requestContext goext.Context, policy goext.LockPolicy) ([]interface{}, error) {
	return schema.listImpl(requestContext, func(ctx context.Context, tx goext.ITransaction) ([]map[string]interface{}, uint64, error) {
		return tx.LockList(ctx, schema, filter, nil, paginator, policy)
	})
}

// List returns list of resources.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (schema *Schema) List(filter goext.Filter, paginator *goext.Paginator, context goext.Context) ([]interface{}, error) {
	fetched, err := schema.ListRaw(filter, paginator, context)
	if err != nil {
		return nil, err
	}
	return schema.rawListToResourceList(fetched)
}

// LockList locks and returns list of resources.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (schema *Schema) LockList(filter goext.Filter, paginator *goext.Paginator, context goext.Context, policy goext.LockPolicy) ([]interface{}, error) {
	fetched, err := schema.LockListRaw(filter, paginator, context, policy)
	if err != nil {
		return nil, err
	}
	return schema.rawListToResourceList(fetched)
}

func (schema *Schema) rawListToResourceList(rawList []interface{}) ([]interface{}, error) {
	if len(rawList) == 0 {
		return rawList, nil
	}
	xRaw := reflect.ValueOf(rawList)
	resourceType, ok := schema.env.getType(schema.ID())
	if !ok {
		schema.env.Logger().Warningf("Full resource type not registered for %s", schema.ID())
		return nil, makeErrMissingType(schema.ID())
	}
	resources := reflect.MakeSlice(reflect.SliceOf(resourceType), xRaw.Len(), xRaw.Len())
	x := reflect.New(resources.Type())
	x.Elem().Set(resources)
	x = x.Elem()

	var err error
	res := make([]interface{}, xRaw.Len(), xRaw.Len())
	for i := 0; i < xRaw.Len(); i++ {
		rawResource := xRaw.Index(i)
		if res[i], err = schema.rawToResource(rawResource.Elem()); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (schema *Schema) rawToResource(xRaw reflect.Value) (interface{}, error) {
	xRaw = xRaw.Elem()
	resourceType, ok := schema.env.getType(schema.ID())
	if !ok {
		schema.env.Logger().Warningf("Full resource type not registered for %s", schema.ID())
		return nil, makeErrMissingType(schema.ID())
	}
	resource := reflect.New(resourceType).Elem()
	setValue(resource.FieldByName(xRaw.Type().Name()), xRaw.Addr())
	resourceBase := goext.NewResourceBase(schema.env, schema, NewLogger(schema.env))
	setValue(resource.FieldByName("ResourceBase"), reflect.ValueOf(resourceBase))
	return resource.Addr().Interface(), nil
}

// FetchRaw fetches a raw resource by ID
func (schema *Schema) FetchRaw(id string, requestContext goext.Context) (interface{}, error) {
	return schema.fetchImpl(id, requestContext, func(ctx context.Context, tx goext.ITransaction, filter goext.Filter) (map[string]interface{}, error) {
		return tx.Fetch(ctx, schema, filter)
	})
}

// LockFetchRaw locks and fetches resource by ID
func (schema *Schema) LockFetchRaw(id string, requestContext goext.Context, policy goext.LockPolicy) (interface{}, error) {
	return schema.fetchImpl(id, requestContext, func(ctx context.Context, tx goext.ITransaction, filter goext.Filter) (map[string]interface{}, error) {
		return tx.LockFetch(ctx, schema, filter, policy)
	})
}

// StateFetchRaw returns a resource state
func (schema *Schema) StateFetchRaw(id string, requestContext goext.Context) (goext.ResourceState, error) {
	tx := mustGetOpenTransactionFromContext(requestContext)
	return tx.StateFetch(goext.GetContext(requestContext), schema, goext.Filter{"id": id})
}

type fetchFunc func(ctx context.Context, tx goext.ITransaction, filter goext.Filter) (map[string]interface{}, error)

func (schema *Schema) fetchImpl(id string, requestContext goext.Context, fetch fetchFunc) (interface{}, error) {
	tx := mustGetOpenTransactionFromContext(requestContext)

	filter := goext.Filter{"id": id}

	data, err := fetch(goext.GetContext(requestContext), tx, filter)

	if err != nil {
		if err == transaction.ErrResourceNotFound {
			return nil, goext.ErrResourceNotFound
		}
		return nil, err
	}

	return schema.ResourceFromMap(data)
}

// Fetch fetches a resource by id.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (schema *Schema) Fetch(id string, context goext.Context) (interface{}, error) {
	fetched, err := schema.FetchRaw(id, context)
	if err != nil {
		return nil, err
	}
	xRaw := reflect.ValueOf(fetched)
	return schema.rawToResource(xRaw)
}

// LockFetch fetches a resource by id.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (schema *Schema) LockFetch(id string, context goext.Context, lockPolicy goext.LockPolicy) (interface{}, error) {
	fetched, err := schema.LockFetchRaw(id, context, lockPolicy)
	if err != nil {
		return nil, err
	}
	xRaw := reflect.ValueOf(fetched)
	return schema.rawToResource(xRaw)
}

func setValue(field, value reflect.Value) {
	if value.IsValid() {
		if value.Type() != field.Type() && field.Kind() == reflect.Slice { // empty slice has type []interface{}
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		} else {
			field.Set(value)
		}
	}
}

// CreateRaw creates a resource
func (schema *Schema) CreateRaw(rawResource interface{}, context goext.Context) error {
	return schema.create(rawResource, context, true)
}

// DbCreateRaw creates a resource without triggering events
func (schema *Schema) DbCreateRaw(rawResource interface{}, context goext.Context) error {
	return schema.create(rawResource, context, false)
}

func (schema *Schema) create(rawResource interface{}, requestContext goext.Context, triggerEvents bool) error {
	if !isPointer(rawResource) {
		return ErrNotPointer
	}

	tx := mustGetOpenTransactionFromContext(requestContext)
	mapFromResource := schema.env.Util().ResourceToMap(rawResource)
	contextCopy := requestContext.Clone().
		WithResource(mapFromResource).
		WithResourceID(mapFromResource["id"].(string)).
		WithSchemaID(schema.ID())

	if triggerEvents {
		if err := schema.env.HandleEvent(goext.PreCreateTx, contextCopy); err != nil {
			return err
		}
	}

	mapFromContext := contextGetMapResource(contextCopy)
	if err := tx.Create(goext.GetContext(contextCopy), schema, mapFromContext); err != nil {
		return err
	}

	v := reflect.ValueOf(rawResource).Elem()
	if err := resourceFromMap(mapFromContext, v); err != nil {
		return err
	}

	if triggerEvents {
		if err := schema.env.HandleEvent(goext.PostCreateTx, contextCopy); err != nil {
			return err
		}
	}

	return nil
}

// UpdateRaw updates a resource and triggers handlers
func (schema *Schema) UpdateRaw(rawResource interface{}, context goext.Context) error {
	return schema.update(rawResource, context, true)
}

// DbUpdateRaw updates a raw resource without triggering events
func (schema *Schema) DbUpdateRaw(rawResource interface{}, context goext.Context) error {
	return schema.update(rawResource, context, false)
}

func (schema *Schema) update(rawResource interface{}, requestContext goext.Context, triggerEvents bool) error {
	if !isPointer(rawResource) {
		return ErrNotPointer
	}
	var resourceData *gohan_schema.Resource
	var err error

	if resourceData, err = schema.structToResource(rawResource); err != nil {
		return err
	}

	tx := mustGetOpenTransactionFromContext(requestContext)

	mapFromResource := schema.env.Util().ResourceToMap(rawResource)
	contextCopy := requestContext.Clone().
		WithResource(mapFromResource).
		WithResourceID(resourceData.ID()).
		WithSchemaID(schema.ID())

	if triggerEvents {
		if err = schema.env.HandleEvent(goext.PreUpdateTx, contextCopy); err != nil {
			return err
		}
	}

	mapFromContext := contextGetMapResource(contextCopy)
	if err = tx.Update(goext.GetContext(requestContext), schema, mapFromContext); err != nil {
		return err
	}

	v := reflect.ValueOf(rawResource).Elem()
	if err := resourceFromMap(mapFromContext, v); err != nil {
		return err
	}

	if triggerEvents {
		if err = schema.env.HandleEvent(goext.PostUpdateTx, contextCopy); err != nil {
			return err
		}
	}

	return nil
}

// DbStateUpdateRaw updates states of a raw resource
func (schema *Schema) DbStateUpdateRaw(rawResource interface{}, requestContext goext.Context, state *goext.ResourceState) error {
	mapFromResource := schema.env.Util().ResourceToMap(rawResource)
	tx := mustGetOpenTransactionFromContext(requestContext)
	return tx.StateUpdate(goext.GetContext(requestContext), schema, mapFromResource, state)
}

// DeleteRaw deletes resource by ID
func (schema *Schema) DeleteRaw(filter goext.Filter, context goext.Context) error {
	return schema.delete(filter, context, true)
}

// DbDeleteRaw deletes resource by ID without triggering events
func (schema *Schema) DbDeleteRaw(filter goext.Filter, context goext.Context) error {
	return schema.delete(filter, context, false)
}

func (schema *Schema) delete(filter goext.Filter, requestContext goext.Context, triggerEvents bool) error {
	tx := mustGetOpenTransactionFromContext(requestContext)
	contextCopy := requestContext.Clone()

	fetched, err := schema.ListRaw(filter, nil, contextCopy)
	if err != nil {
		return err
	}

	mapper := reflectx.NewMapper("db")
	for i := 0; i < len(fetched); i++ {
		resource := reflect.ValueOf(fetched[i])
		resourceID := mapper.FieldByName(resource, "id").Interface().(string)

		mapFromResource := schema.env.Util().ResourceToMap(resource.Interface())
		contextCopy = contextCopy.WithResource(mapFromResource).
			WithSchemaID(schema.ID()).
			WithResourceID(resourceID)

		if triggerEvents {
			if err = schema.env.HandleEvent(goext.PreDeleteTx, contextCopy); err != nil {
				return err
			}
		}

		if err = tx.Delete(goext.GetContext(requestContext), schema, resourceID); err != nil {
			return err
		}

		if triggerEvents {
			if err = schema.env.HandleEvent(goext.PostDeleteTx, contextCopy); err != nil {
				return err
			}
		}
	}

	return nil
}

// RegisterEventHandler registers a schema handler
func (schema *Schema) RegisterEventHandler(event string, schemaHandler goext.SchemaHandler, priority int) {
	schema.env.RegisterSchemaEventHandler(schema.raw.ID, event, schemaHandler, priority)
}

// RegisterRawType registers a runtime type for a raw resource
func (schema *Schema) RegisterRawType(typeValue interface{}) {
	schema.env.RegisterRawType(schema.raw.ID, typeValue)
}

// RegisterType registers a runtime type for a resource
func (schema *Schema) RegisterType(resourceType goext.IResourceBase) {
	schema.env.RegisterType(schema.raw.ID, resourceType)
}

func (schema *Schema) RawSchema() interface{} {
	return schema.raw
}

// DerivedSchemas returns list of schemas that extend schema with given id
func (schema *Schema) DerivedSchemas() []goext.ISchema {
	manager := gohan_schema.GetManager()
	derived := []goext.ISchema{}
	for _, raw := range manager.OrderedSchemas() {
		for _, parent := range raw.Extends {
			if parent == schema.ID() {
				derived = append(derived, NewSchema(schema.env, raw))
				break
			}
		}
	}
	return derived
}

// ColumnNames generates an array that has Gohan style column names
func (schema *Schema) ColumnNames() []string {
	return sql.MakeColumns(schema.raw, schema.raw.GetDbTableName(), nil, false)
}

// Properties returns properties of schema
func (schema *Schema) Properties() []goext.Property {
	rawProperties := schema.raw.Properties
	properties := make([]goext.Property, len(rawProperties))
	for i, property := range rawProperties {
		properties[i] = goext.Property{
			ID:       property.ID,
			Title:    property.Title,
			Relation: property.Relation,
		}
	}
	return properties
}

// NewSchema allocates a new Schema
func NewSchema(env IEnvironment, raw *gohan_schema.Schema) goext.ISchema {
	return &Schema{env: env, raw: raw}
}

func contextSetTransaction(ctx goext.Context, tx goext.ITransaction) goext.Context {
	ctx["transaction"] = tx.RawTransaction()
	return ctx
}

func contextGetMapResource(ctx goext.Context) map[string]interface{} {
	return ctx["resource"].(map[string]interface{})
}

func mustGetOpenTransactionFromContext(context goext.Context) goext.ITransaction {
	if context == nil {
		panic("Database function called without open transaction")
	}
	tx, hasTransaction := contextGetTransaction(context)
	if !hasTransaction || tx.Closed() {
		panic("Database function called without open transaction")
	}
	return tx
}
