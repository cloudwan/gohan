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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/schema"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/twinj/uuid"
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
	environment *Environment
}

// List returns a list of loaded schemas
func (thisSchemas *Schemas) List() []goext.ISchema {
	manager := schema.GetManager()
	result := []goext.ISchema{}
	for _, rawSchema := range manager.OrderedSchemas() {
		result = append(result, NewSchema(thisSchemas.environment, rawSchema))
	}
	return result
}

// Find returns a schema by id or nil if not found
func (thisSchemas *Schemas) Find(id string) goext.ISchema {
	manager := schema.GetManager()
	sch, ok := manager.Schema(id)

	if !ok {
		log.Warning("cannot find schema '%s'", id)
		return nil
	}

	return NewSchema(thisSchemas.environment, sch)
}

// Environment returns the parent environment
func (thisSchemas *Schemas) Environment() goext.IEnvironment {
	return thisSchemas.environment
}

// NewSchemas allocates a new Schemas
func NewSchemas(environment *Environment) goext.ISchemas {
	return &Schemas{environment: environment}
}

// Schema is an implementation of ISchema
type Schema struct {
	environment *Environment
	rawSchema   *schema.Schema
}

// ID returns ID of schema
func (thisSchema *Schema) ID() string {
	return thisSchema.rawSchema.ID
}

func (thisSchema *Schema) structToMap(resource interface{}) map[string]interface{} {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("db")

	structMap := mapper.TypeMap(reflect.TypeOf(resource))
	resourceValue := reflect.ValueOf(resource)

	for _, property := range thisSchema.rawSchema.Properties {
		field := property.ID

		fi, ok := structMap.Names[property.ID]
		if !ok {
			panic(fmt.Sprintf("property %s not found in %+v", property.ID, resource))
		}

		v := reflectx.FieldByIndexesReadOnly(resourceValue, fi.Index)
		val := v.Interface()
		if field == "id" && v.String() == "" {
			id := uuid.NewV4().String()
			fieldsMap[field] = id
			v.SetString(id)
		} else if strings.Contains(v.Type().String(), "goext.Null") {
			valid := v.FieldByName("Valid").Bool()
			if valid {
				fieldsMap[field] = v.FieldByName("Value").Interface()
			} else {
				fieldsMap[field] = nil
			}
		} else if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				fieldsMap[field] = nil
			} else {
				fieldsMap[field] = val
			}
		} else {
			fieldsMap[field] = val
		}
	}

	return fieldsMap
}

func (thisSchema *Schema) structToResource(resource interface{}) (*schema.Resource, error) {
	fieldsMap := thisSchema.structToMap(resource)
	return schema.NewResource(thisSchema.rawSchema, fieldsMap)
}

func (thisSchema *Schema) assignField(name string, field reflect.Value, value interface{}) error {
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
func (thisSchema *Schema) ListRaw(filter goext.Filter, paginator *goext.Paginator, context goext.Context) ([]interface{}, error) {
	resourceType, ok := GlobRawTypes[thisSchema.ID()]
	if !ok {
		return nil, ErrMissingType
	}

	if context == nil {
		context = goext.MakeContext()
	}

	tx, hasOpenTransaction := contextGetTransaction(context)
	if !hasOpenTransaction {
		var err error
		tx, err = thisSchema.environment.Database().Begin()

		if err != nil {
			return nil, err
		}

		defer tx.Close()
	}

	data, _, err := tx.List(thisSchema, filter, nil, paginator)

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
			if err := thisSchema.assignField(name, field, value); err != nil {
				return nil, err
			}
		}
		res[i] = resource.Interface()
	}

	return res, nil
}

// LockListRaw locks and returns raw resources
func (thisSchema *Schema) LockListRaw(filter goext.Filter, paginator *goext.Paginator, context goext.Context, policy goext.LockPolicy) ([]interface{}, error) {
	//TODO: implement proper locking
	return thisSchema.ListRaw(filter, paginator, context)
}

// List returns list of resources.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (thisSchema *Schema) List(filter goext.Filter, paginator *goext.Paginator, context goext.Context) ([]interface{}, error) {
	fetched, err := thisSchema.ListRaw(filter, paginator, context)
	if err != nil {
		return nil, err
	}
	xRaw := reflect.ValueOf(fetched)
	return thisSchema.rawListToResourceList(xRaw), nil
}

// LockList locks and returns list of resources.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (thisSchema *Schema) LockList(filter goext.Filter, paginator *goext.Paginator, context goext.Context, policy goext.LockPolicy) ([]interface{}, error) {
	fetched, err := thisSchema.LockListRaw(filter, paginator, context, policy)
	if err != nil {
		return nil, err
	}
	xRaw := reflect.ValueOf(fetched)
	return thisSchema.rawListToResourceList(xRaw), nil
}

func (thisSchema *Schema) rawListToResourceList(xRaw reflect.Value) []interface{} {
	resources := reflect.MakeSlice(reflect.SliceOf(GlobTypes[thisSchema.ID()]), xRaw.Len(), xRaw.Len())
	x := reflect.New(resources.Type())
	x.Elem().Set(resources)
	x = x.Elem()

	res := make([]interface{}, xRaw.Len(), xRaw.Len())
	for i := 0; i < xRaw.Len(); i++ {
		rawResource := xRaw.Index(i)
		res[i] = thisSchema.rawToResource(rawResource.Elem())
	}
	return res
}

func (thisSchema *Schema) rawToResource(xRaw reflect.Value) interface{} {
	xRaw = xRaw.Elem()
	resource := reflect.New(GlobTypes[thisSchema.ID()]).Elem()
	setValue(resource.FieldByName(xRaw.Type().Name()), xRaw.Addr())
	setValue(resource.FieldByName("Schema"), reflect.ValueOf(thisSchema))
	setValue(resource.FieldByName("Logger"), reflect.ValueOf(NewLogger(thisSchema.environment)))
	setValue(resource.FieldByName("Environment"), reflect.ValueOf(thisSchema.environment))
	return resource.Addr().Interface()
}

// FetchRaw fetches a raw resource by ID
func (thisSchema *Schema) FetchRaw(id string, context goext.Context) (interface{}, error) {
	if context == nil {
		context = goext.MakeContext()
	}
	tx, hasOpenTransaction := contextGetTransaction(context)
	if !hasOpenTransaction {
		var err error
		tx, err = thisSchema.environment.Database().Begin()

		if err != nil {
			return nil, err
		}

		defer tx.Close()

		contextSetTransaction(context, tx)
	}

	filter := goext.Filter{"id": id}

	data, err := tx.Fetch(thisSchema, filter)

	if err != nil {
		return nil, err
	}
	resourceType, ok := GlobRawTypes[thisSchema.rawSchema.ID]
	if !ok {
		return nil, fmt.Errorf("No type registered for schema: %s", thisSchema.rawSchema.ID)
	}
	rawResources := GlobRawTypes[thisSchema.ID()]
	resource := reflect.New(rawResources)

	for i := 0; i < resourceType.NumField(); i++ {
		field := resource.Elem().Field(i)

		fieldType := resourceType.Field(i)
		propertyName := fieldType.Tag.Get("db")
		property, err := thisSchema.rawSchema.GetPropertyByID(propertyName)
		if err != nil {
			return nil, err
		}
		value := data[property.ID]
		thisSchema.assignField(propertyName, field, value)
	}

	return resource.Interface(), nil
}

// Fetch fetches a resource by id.
// Schema, Logger, Environment and pointer to raw resource are required fields in the resource
func (thisSchema *Schema) Fetch(id string, context goext.Context) (interface{}, error) {
	fetched, err := thisSchema.FetchRaw(id, context)
	if err != nil {
		return nil, err
	}
	xRaw := reflect.ValueOf(fetched)
	return thisSchema.rawToResource(xRaw), nil
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

// LockFetchRaw locks and fetches resource by ID
func (thisSchema *Schema) LockFetchRaw(id string, context goext.Context, policy goext.LockPolicy) (interface{}, error) {
	return thisSchema.FetchRaw(id, context)
}

// CreateRaw creates a resource
func (thisSchema *Schema) CreateRaw(rawResource interface{}, context goext.Context) error {
	if !isPointer(rawResource) {
		return ErrNotPointer
	}

	if context == nil {
		context = goext.MakeContext()
	}
	tx, hasOpenTransaction := contextGetTransaction(context)
	if hasOpenTransaction {
		contextCopy := goext.MakeContext().
			WithSchemaID(thisSchema.ID()).
			WithResource(thisSchema.structToMap(rawResource))
		contextSetTransaction(contextCopy, tx)
		return thisSchema.createInTransaction(rawResource, contextCopy, tx)
	}

	context.WithSchemaID(thisSchema.ID()).
		WithResource(thisSchema.structToMap(rawResource))

	if err := thisSchema.environment.HandleEvent(goext.PreCreate, context); err != nil {
		return err
	}

	tx, err := thisSchema.environment.Database().Begin()
	if err != nil {
		return err
	}
	defer tx.Close()
	contextSetTransaction(context, tx)

	if err = thisSchema.environment.HandleEvent(goext.PreCreateTx, context); err != nil {
		return err
	}

	if err = tx.Create(thisSchema, context["resource"].(map[string]interface{})); err != nil {
		return err
	}

	if err = thisSchema.environment.updateResourceFromContext(rawResource, context); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PostCreateTx, context); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if err = tx.Close(); err != nil {
		return err
	}

	return thisSchema.environment.HandleEvent(goext.PostCreate, context)
}

func (thisSchema *Schema) createInTransaction(resource interface{}, context goext.Context, tx goext.ITransaction) error {
	var err error

	if err = thisSchema.environment.HandleEvent(goext.PreCreate, context); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PreCreateTx, context); err != nil {
		return err
	}

	if err = tx.Create(thisSchema, context["resource"].(map[string]interface{})); err != nil {
		return err
	}

	if err = thisSchema.environment.updateResourceFromContext(resource, context); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PostCreateTx, context); err != nil {
		return err
	}

	return thisSchema.environment.HandleEvent(goext.PostCreate, context)
}

// UpdateRaw updates a resource and triggers handlers
func (thisSchema *Schema) UpdateRaw(rawResource interface{}, context goext.Context) error {
	if !isPointer(rawResource) {
		return ErrNotPointer
	}
	var tx goext.ITransaction
	var resourceData *schema.Resource
	var err error

	if resourceData, err = thisSchema.structToResource(rawResource); err != nil {
		return err
	}

	if context == nil {
		context = goext.MakeContext()
	}

	contextCopy := goext.MakeContext()
	for k, v := range context {
		contextCopy[k] = v
	}
	contextCopy.WithResource(thisSchema.structToMap(rawResource)).
		WithResourceID(resourceData.ID()).
		WithSchemaID(thisSchema.ID())

	if err = thisSchema.environment.HandleEvent(goext.PreUpdate, contextCopy); err != nil {
		return err
	}

	tx, hasOpenTransaction := contextGetTransaction(contextCopy)
	if !hasOpenTransaction {
		if tx, err = thisSchema.environment.Database().Begin(); err != nil {
			return err
		}

		defer tx.Close()
		contextSetTransaction(contextCopy, tx)
		contextSetTransaction(context, tx)
	}

	if err = thisSchema.environment.HandleEvent(goext.PreUpdateTx, contextCopy); err != nil {
		return err
	}

	if err = tx.Update(thisSchema, contextCopy["resource"].(map[string]interface{})); err != nil {
		return err
	}

	if err = thisSchema.environment.updateResourceFromContext(rawResource, contextCopy); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PostUpdateTx, contextCopy); err != nil {
		return err
	}

	if !hasOpenTransaction {
		if err = tx.Commit(); err != nil {
			return err
		}
	}

	return thisSchema.environment.HandleEvent(goext.PostUpdate, contextCopy)
}

// DbUpdateRaw updates a raw resource without triggering events
func (thisSchema *Schema) DbUpdateRaw(rawResource interface{}, context goext.Context) error {
	if !isPointer(rawResource) {
		return ErrNotPointer
	}
	resourceData, err := thisSchema.structToResource(rawResource)
	if err != nil {
		return err
	}

	if context == nil {
		context = goext.MakeContext()
	}

	context.WithResource(thisSchema.structToMap(rawResource)).
		WithResourceID(resourceData.ID()).
		WithSchemaID(thisSchema.ID())

	tx, hasOpenTransaction := contextGetTransaction(context)
	if !hasOpenTransaction {
		if tx, err = thisSchema.environment.Database().Begin(); err != nil {
			return err
		}

		defer tx.Close()
		contextSetTransaction(context, tx)
	}

	if err = tx.Update(thisSchema, context["resource"].(map[string]interface{})); err != nil {
		return err
	}

	if err = thisSchema.environment.updateResourceFromContext(rawResource, context); err != nil {
		return err
	}

	if !hasOpenTransaction {
		if err = tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

// DeleteRaw deletes resource by ID
func (thisSchema *Schema) DeleteRaw(filter goext.Filter, context goext.Context) error {
	var tx goext.ITransaction
	var err error
	if context == nil {
		context = goext.MakeContext()
	}
	tx, hasOpenTransaction := contextGetTransaction(context)
	if !hasOpenTransaction {
		if tx, err = thisSchema.environment.Database().Begin(); err != nil {
			return err
		}

		defer tx.Close()

		contextSetTransaction(context, tx)
	}
	contextTx := goext.MakeContext()
	contextSetTransaction(contextTx, tx)

	fetched, err := thisSchema.LockListRaw(filter, nil, contextTx, goext.LockRelatedResources)
	if err != nil {
		return err
	}
	fetchedLen := len(fetched)
	if fetchedLen == 0 {
		return fmt.Errorf("resource not found")
	}

	mapper := reflectx.NewMapper("db")
	for i := 0; i < fetchedLen; i++ {
		resource := reflect.ValueOf(fetched[i])
		resourceID := mapper.FieldByName(resource, "id").Interface()

		contextTx = contextTx.WithResource(thisSchema.structToMap(resource.Interface())).
			WithSchemaID(thisSchema.ID())

		if err = thisSchema.environment.HandleEvent(goext.PreDelete, contextTx); err != nil {
			return err
		}

		if err = thisSchema.environment.HandleEvent(goext.PreDeleteTx, contextTx); err != nil {
			return err
		}

		if err = tx.Delete(thisSchema, resourceID); err != nil {
			return err
		}

		if err = thisSchema.environment.HandleEvent(goext.PostDeleteTx, contextTx); err != nil {
			return err
		}

		if err = thisSchema.environment.HandleEvent(goext.PostDelete, contextTx); err != nil {
			return err
		}
	}

	if !hasOpenTransaction {
		tx.Commit()
	}

	return nil
}

// RegisterEventHandler registers a schema handler
func (thisSchema *Schema) RegisterEventHandler(event string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority goext.Priority) {
	thisSchema.environment.RegisterSchemaEventHandler(thisSchema.rawSchema.ID, event, handler, priority)
}

// RegisterRawType registers a runtime type for a raw resource
func (thisSchema *Schema) RegisterRawType(typeValue interface{}) {
	thisSchema.environment.RegisterRawType(thisSchema.rawSchema.ID, typeValue)
}

// RegisterType registers a runtime type for a resource
func (thisSchema *Schema) RegisterType(typeValue interface{}) {
	thisSchema.environment.RegisterType(thisSchema.rawSchema.ID, typeValue)
}

// Environment returns the parent environment
func (thisSchema *Schema) Environment() goext.IEnvironment {
	return thisSchema.environment
}

// NewSchema allocates a new Schema
func NewSchema(environment *Environment, rawSchema *schema.Schema) goext.ISchema {
	return &Schema{environment: environment, rawSchema: rawSchema}
}

func contextSetTransaction(ctx goext.Context, tx goext.ITransaction) goext.Context {
	ctx["transaction"] = tx
	return ctx
}

func contextGetTransaction(ctx goext.Context) (goext.ITransaction, bool) {
	ctxTx := ctx["transaction"]
	if ctxTx == nil {
		return nil, false
	}
	return ctxTx.(goext.ITransaction), true
}
