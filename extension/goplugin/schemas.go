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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	gohan_schema "github.com/cloudwan/gohan/schema"
	"github.com/jmoiron/sqlx/reflectx"
)

var (
	// ErrNotPointer indicates that a resource was not passed by a pointer
	ErrNotPointer = fmt.Errorf("raw resource must be passed by a pointer")

	// ErrMissingType indicates that a runtime type was not registered for a resource
	ErrMissingType = fmt.Errorf("resource type not registered")
)

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
		return nil, fmt.Errorf("no raw type registered for schema: %s", schema.ID())
	}

	return resourceFromMap(context, rawType)
}

func (schema *Schema) structToResource(resource interface{}) (*gohan_schema.Resource, error) {
	fieldsMap := schema.env.Util().ResourceToMap(resource)
	return gohan_schema.NewResource(schema.raw, fieldsMap)
}

func (schema *Schema) assignField(name string, field reflect.Value, value interface{}) error {
	if field.Kind() == reflect.Struct || field.Kind() == reflect.Slice || field.Kind() == reflect.Ptr {
		mapJSON, err := json.Marshal(value)
		if err != nil {
			return err
		}
		newField := reflect.New(field.Type())
		fieldJSON := string(mapJSON)
		fieldInterface := newField.Interface()
		err = json.Unmarshal([]byte(fieldJSON), &fieldInterface)
		if err != nil {
			return err
		}
		field.Set(newField.Elem())
	} else {
		setValue(field, reflect.ValueOf(value))
	}
	return nil
}

// ListRaw lists schema raw resources
func (schema *Schema) ListRaw(filter goext.Filter, paginator *goext.Paginator, requestContext goext.Context) ([]interface{}, error) {
	return schema.listImpl(requestContext, func(ctx context.Context, tx goext.ITransaction) ([]map[string]interface{}, uint64, error) {
		return tx.List(ctx, schema, filter, nil, paginator)
	})
}

type listFunc func(ctx context.Context, tx goext.ITransaction) ([]map[string]interface{}, uint64, error)

func (schema *Schema) listImpl(requestContext goext.Context, list listFunc) ([]interface{}, error) {
	resourceType, ok := schema.env.getRawType(schema.ID())
	if !ok {
		log.Warning(fmt.Sprintf("cannot find raw type for: %s", schema.ID()))
		return nil, ErrMissingType
	}

	if requestContext == nil {
		requestContext = goext.MakeContext()
	}

	tx, hasOpenTransaction := contextGetTransaction(requestContext)
	if !hasOpenTransaction {
		var err error
		tx, err = schema.env.Database().Begin()

		if err != nil {
			return nil, err
		}

		defer tx.Close()
	}

	data, _, err := list(goext.GetContext(requestContext), tx)

	if err != nil {
		return nil, err
	}

	mapper := reflectx.NewMapper("db")
	res := make([]interface{}, len(data), len(data))

	for i := 0; i < len(data); i++ {
		resource := reflect.New(resourceType)
		mapped := mapper.FieldMap(resource)

		for name, field := range mapped {
			value := data[i][name]
			if err := schema.assignField(name, field, value); err != nil {
				return nil, err
			}
		}
		res[i] = resource.Interface()
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
	return schema.rawListToResourceList(fetched), nil
}

// LockList locks and returns list of resources.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (schema *Schema) LockList(filter goext.Filter, paginator *goext.Paginator, context goext.Context, policy goext.LockPolicy) ([]interface{}, error) {
	fetched, err := schema.LockListRaw(filter, paginator, context, policy)
	if err != nil {
		return nil, err
	}
	return schema.rawListToResourceList(fetched), nil
}

func (schema *Schema) rawListToResourceList(rawList []interface{}) []interface{} {
	if len(rawList) == 0 {
		return rawList
	}
	xRaw := reflect.ValueOf(rawList)
	resourceType, _ := schema.env.getType(schema.ID())
	resources := reflect.MakeSlice(reflect.SliceOf(resourceType), xRaw.Len(), xRaw.Len())
	x := reflect.New(resources.Type())
	x.Elem().Set(resources)
	x = x.Elem()

	res := make([]interface{}, xRaw.Len(), xRaw.Len())
	for i := 0; i < xRaw.Len(); i++ {
		rawResource := xRaw.Index(i)
		res[i] = schema.rawToResource(rawResource.Elem())
	}
	return res
}

func (schema *Schema) rawToResource(xRaw reflect.Value) interface{} {
	xRaw = xRaw.Elem()
	resourceType, _ := schema.env.getType(schema.ID())
	resource := reflect.New(resourceType).Elem()
	setValue(resource.FieldByName(xRaw.Type().Name()), xRaw.Addr())
	setValue(resource.FieldByName("Schema"), reflect.ValueOf(schema))
	setValue(resource.FieldByName("Logger"), reflect.ValueOf(NewLogger(schema.env)))
	setValue(resource.FieldByName("Environment"), reflect.ValueOf(schema.env))
	return resource.Addr().Interface()
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

type fetchFunc func(ctx context.Context, tx goext.ITransaction, filter goext.Filter) (map[string]interface{}, error)

func (schema *Schema) fetchImpl(id string, requestContext goext.Context, fetch fetchFunc) (interface{}, error) {
	if requestContext == nil {
		requestContext = goext.MakeContext()
	}
	tx, hasOpenTransaction := contextGetTransaction(requestContext)
	if !hasOpenTransaction {
		var err error
		tx, err = schema.env.Database().Begin()

		if err != nil {
			return nil, err
		}

		defer tx.Close()

		contextSetTransaction(requestContext, tx)
	}

	filter := goext.Filter{"id": id}

	data, err := fetch(goext.GetContext(requestContext), tx, filter)

	if err != nil {
		if err == transaction.ErrResourceNotFound {
			return nil, goext.ErrResourceNotFound
		}
		return nil, err
	}
	resourceType, ok := schema.env.getRawType(schema.raw.ID)
	if !ok {
		return nil, fmt.Errorf("No type registered for schema: %s", schema.raw.ID)
	}
	rawResources, _ := schema.env.getRawType(schema.ID())
	resource := reflect.New(rawResources)

	for i := 0; i < resourceType.NumField(); i++ {
		field := resource.Elem().Field(i)

		fieldType := resourceType.Field(i)
		propertyName := fieldType.Tag.Get("db")
		property, err := schema.raw.GetPropertyByID(propertyName)
		if err != nil {
			return nil, err
		}
		value := data[property.ID]
		schema.assignField(propertyName, field, value)
	}

	return resource.Interface(), nil
}

// Fetch fetches a resource by id.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (schema *Schema) Fetch(id string, context goext.Context) (interface{}, error) {
	fetched, err := schema.FetchRaw(id, context)
	if err != nil {
		return nil, err
	}
	xRaw := reflect.ValueOf(fetched)
	return schema.rawToResource(xRaw), nil
}

// LockFetch fetches a resource by id.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (schema *Schema) LockFetch(id string, context goext.Context, lockPolicy goext.LockPolicy) (interface{}, error) {
	fetched, err := schema.LockFetchRaw(id, context, lockPolicy)
	if err != nil {
		return nil, err
	}
	xRaw := reflect.ValueOf(fetched)
	return schema.rawToResource(xRaw), nil
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

	if requestContext == nil {
		requestContext = goext.MakeContext()
	}
	ctx := schema.env.Util().ResourceToMap(rawResource)
	tx, hasOpenTransaction := contextGetTransaction(requestContext)
	if hasOpenTransaction {
		contextCopy := goext.MakeContext().
			WithSchemaID(schema.ID()).
			WithResource(ctx)
		contextSetTransaction(contextCopy, tx)
		return schema.createInTransaction(rawResource, contextCopy, tx, triggerEvents)
	}

	requestContext.WithSchemaID(schema.ID()).
		WithResource(ctx)

	if triggerEvents {
		if err := schema.env.HandleEvent(goext.PreCreate, requestContext); err != nil {
			return err
		}
	}

	tx, err := schema.env.Database().Begin()
	if err != nil {
		return err
	}
	defer tx.Close()
	contextSetTransaction(requestContext, tx)

	if triggerEvents {
		if err = schema.env.HandleEvent(goext.PreCreateTx, requestContext); err != nil {
			return err
		}
	}

	ctxResource := requestContext["resource"].(map[string]interface{})
	if err = tx.Create(goext.GetContext(requestContext), schema, ctxResource); err != nil {
		return err
	}

	v := reflect.ValueOf(rawResource).Elem()
	res, err := resourceFromMap(ctxResource, v.Type())
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(res).Elem())

	if triggerEvents {
		if err = schema.env.HandleEvent(goext.PostCreateTx, requestContext); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if err = tx.Close(); err != nil {
		return err
	}

	if !triggerEvents {
		return nil
	}
	return schema.env.HandleEvent(goext.PostCreate, requestContext)
}

func (schema *Schema) createInTransaction(rawResource interface{}, requestContext goext.Context, tx goext.ITransaction, triggerEvents bool) error {
	var err error

	if triggerEvents {
		if err = schema.env.HandleEvent(goext.PreCreate, requestContext); err != nil {
			return err
		}

		if err = schema.env.HandleEvent(goext.PreCreateTx, requestContext); err != nil {
			return err
		}
	}

	ctxResource := requestContext["resource"].(map[string]interface{})
	if err = tx.Create(goext.GetContext(requestContext), schema, ctxResource); err != nil {
		return err
	}

	v := reflect.ValueOf(rawResource).Elem()
	res, err := resourceFromMap(ctxResource, v.Type())
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(res).Elem())

	if !triggerEvents {
		return nil
	}
	if err = schema.env.HandleEvent(goext.PostCreateTx, requestContext); err != nil {
		return err
	}

	return schema.env.HandleEvent(goext.PostCreate, requestContext)
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
	var tx goext.ITransaction
	var resourceData *gohan_schema.Resource
	var err error

	if resourceData, err = schema.structToResource(rawResource); err != nil {
		return err
	}

	if requestContext == nil {
		requestContext = goext.MakeContext()
	}

	contextCopy := goext.MakeContext()
	for k, v := range requestContext {
		contextCopy[k] = v
	}
	ctx := schema.env.Util().ResourceToMap(rawResource)
	contextCopy.WithResource(ctx).
		WithResourceID(resourceData.ID()).
		WithSchemaID(schema.ID())

	if triggerEvents {
		if err = schema.env.HandleEvent(goext.PreUpdate, contextCopy); err != nil {
			return err
		}
	}

	tx, hasOpenTransaction := contextGetTransaction(contextCopy)
	if !hasOpenTransaction {
		if tx, err = schema.env.Database().Begin(); err != nil {
			return err
		}

		defer tx.Close()
		contextSetTransaction(contextCopy, tx)
		contextSetTransaction(requestContext, tx)
	}

	if triggerEvents {
		if err = schema.env.HandleEvent(goext.PreUpdateTx, contextCopy); err != nil {
			return err
		}
	}

	ctxResource := contextCopy["resource"].(map[string]interface{})
	if err = tx.Update(goext.GetContext(requestContext), schema, ctxResource); err != nil {
		return err
	}

	v := reflect.ValueOf(rawResource).Elem()
	res, err := resourceFromMap(ctxResource, v.Type())
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(res).Elem())

	if triggerEvents {
		if err = schema.env.HandleEvent(goext.PostUpdateTx, contextCopy); err != nil {
			return err
		}
	}

	if !hasOpenTransaction {
		if err = tx.Commit(); err != nil {
			return err
		}
	}

	if !triggerEvents {
		return nil
	}
	return schema.env.HandleEvent(goext.PostUpdate, contextCopy)
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
	var tx goext.ITransaction
	var err error
	if requestContext == nil {
		requestContext = goext.MakeContext()
	}
	tx, hasOpenTransaction := contextGetTransaction(requestContext)
	if !hasOpenTransaction {
		if tx, err = schema.env.Database().Begin(); err != nil {
			return err
		}

		defer tx.Close()

		contextSetTransaction(requestContext, tx)
	}
	contextTx := goext.MakeContext()
	contextSetTransaction(contextTx, tx)

	fetched, err := schema.LockListRaw(filter, nil, contextTx, goext.LockRelatedResources)
	if err != nil {
		return err
	}

	mapper := reflectx.NewMapper("db")
	for i := 0; i < len(fetched); i++ {
		resource := reflect.ValueOf(fetched[i])
		resourceID := mapper.FieldByName(resource, "id").Interface()

		ctx := schema.env.Util().ResourceToMap(resource.Interface())
		contextTx = contextTx.WithResource(ctx).
			WithSchemaID(schema.ID())

		if triggerEvents {
			if err = schema.env.HandleEvent(goext.PreDelete, contextTx); err != nil {
				return err
			}

			if err = schema.env.HandleEvent(goext.PreDeleteTx, contextTx); err != nil {
				return err
			}
		}

		if err = tx.Delete(goext.GetContext(requestContext), schema, resourceID); err != nil {
			return err
		}

		if triggerEvents {
			if err = schema.env.HandleEvent(goext.PostDeleteTx, contextTx); err != nil {
				return err
			}

			if err = schema.env.HandleEvent(goext.PostDelete, contextTx); err != nil {
				return err
			}
		}
	}

	if !hasOpenTransaction {
		tx.Commit()
	}

	return nil
}

// RegisterEventHandler registers a schema handler
func (schema *Schema) RegisterEventHandler(event string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority int) {
	schema.env.RegisterSchemaEventHandler(schema.raw.ID, event, handler, priority)
}

// RegisterRawType registers a runtime type for a raw resource
func (schema *Schema) RegisterRawType(typeValue interface{}) {
	schema.env.RegisterRawType(schema.raw.ID, typeValue)
}

// RegisterType registers a runtime type for a resource
func (schema *Schema) RegisterType(typeValue interface{}) {
	schema.env.RegisterType(schema.raw.ID, typeValue)
}

// NewSchema allocates a new Schema
func NewSchema(env IEnvironment, raw *gohan_schema.Schema) goext.ISchema {
	return &Schema{env: env, raw: raw}
}

func contextSetTransaction(ctx goext.Context, tx goext.ITransaction) goext.Context {
	ctx["transaction"] = tx.RawTransaction()
	return ctx
}
