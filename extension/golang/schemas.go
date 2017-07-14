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

package golang

import (
	"fmt"
	"reflect"

	"encoding/json"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/schema"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/twinj/uuid"
)

type Schemas struct {
	environment *Environment
}

func (thisSchemasBinder *Schemas) List() []goext.ISchema {
	manager := schema.GetManager()
	result := []goext.ISchema{}
	for _, rawSchema := range manager.OrderedSchemas() {
		result = append(result, NewSchema(thisSchemasBinder.environment, rawSchema))
	}
	return result
}

func (thisSchemasBinder *Schemas) Find(id string) goext.ISchema {
	manager := schema.GetManager()
	schema, ok := manager.Schema(id)

	if !ok {
		return nil
	}

	return NewSchema(thisSchemasBinder.environment, schema)
}

func (thisSchemasBinder *Schemas) Environment() goext.IEnvironment {
	return thisSchemasBinder.environment
}

type Schema struct {
	environment *Environment
	rawSchema   *schema.Schema
}

func NewSchema(environment *Environment, rawSchema *schema.Schema) goext.ISchema {
	return &Schema{environment: environment, rawSchema: rawSchema}
}

func (thisSchema *Schema) ID() string {
	return thisSchema.rawSchema.ID
}

func (thisSchema *Schema) Environment() goext.IEnvironment {
	return thisSchema.environment
}

func (thisSchema *Schema) structToMap(resource interface{}) map[string]interface{} {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("db")

	for _, property := range thisSchema.rawSchema.Properties {
		field := property.ID
		v := mapper.FieldByName(reflect.ValueOf(resource), property.ID)
		val := v.Interface()
		if field == "id" && v.String() == "" {
			id := uuid.NewV4().String()
			fieldsMap[field] = id
			v.SetString(id)
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

//List - lists resources
func (thisSchema *Schema) List(resources interface{}) error {
	slicePtrValue := reflect.ValueOf(resources)
	slicePtrType := reflect.TypeOf(resources)
	sliceValue := slicePtrValue.Elem()
	sliceType := slicePtrType.Elem()
	elemType := sliceType.Elem()

	sliceValue.SetLen(0)

	tx, err := thisSchema.environment.dataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	data, _, err := tx.List(thisSchema.rawSchema, transaction.Filter{}, nil, nil)

	if err != nil {
		return err
	}

	mapper := reflectx.NewMapper("db")

	for i := 0; i < len(data); i++ {
		resource := reflect.New(elemType)
		mapped := mapper.FieldMap(resource)

		for name, field := range mapped {
			field.Set(reflect.ValueOf(data[i].Get(name)))
		}

		sliceValue.Set(reflect.Append(sliceValue, resource.Elem()))
	}

	return nil
}

func (thisSchema *Schema) FetchRelated(resource interface{}, relatedResource interface{}) error {
	for _, property := range thisSchema.rawSchema.Properties {
		if property.Relation != "" {
			relatedSchema, ok := schema.GetManager().Schema(property.Relation)

			if !ok {
				return fmt.Errorf("Could not get related schema: %s for: %s", property.Relation, thisSchema.rawSchema.ID)
			}

			mapper := reflectx.NewMapper("db")
			id := mapper.FieldByName(reflect.ValueOf(resource), property.ID).String()

			NewSchema(thisSchema.environment, relatedSchema).Fetch(id, relatedResource)

			return nil
		}
	}

	return nil
}

//Fetch - retrieves resource by ID
func (thisSchema *Schema) Fetch(id string, res interface{}) error {
	tx, err := thisSchema.environment.dataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	filter := transaction.Filter{"id": id}

	data, err := tx.Fetch(thisSchema.rawSchema, filter)

	if err != nil {
		return err
	}
	resourceType, ok := GlobResourceTypes[thisSchema.rawSchema.ID]
	if !ok {
		return fmt.Errorf("No type registered for schema title: %s", thisSchema.rawSchema.ID)
	}
	resource := reflect.ValueOf(res)

	for i := 0; i < resourceType.NumField(); i++ {
		field := resource.Elem().Field(i)

		fieldType := resourceType.Field(i)
		propertyName := fieldType.Tag.Get("db")
		property, err := thisSchema.rawSchema.GetPropertyByID(propertyName)
		if err != nil {
			return err
		}
		if fieldType.Type.Kind() == reflect.Struct {
			mapJSON, err := json.Marshal(data.Get(property.ID))
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
			value := reflect.ValueOf(data.Get(property.ID))
			if value.IsValid() {
				field.Set(value)
			}
		}
	}

	return nil
}

// Create creates a resource
func (thisSchema *Schema) Create(resource interface{}) error {
	var tx transaction.Transaction
	var resourceData *schema.Resource
	var err error

	context := goext.MakeContext().
		WithResource(thisSchema.structToMap(resource)).
		WithSchemaID(thisSchema.ID())

	if err = thisSchema.environment.HandleEvent(goext.PreCreate, context); err != nil {
		return err
	}

	if tx, err = thisSchema.environment.dataStore.Begin(); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PreCreateTx, context); err != nil {
		return err
	}

	defer tx.Close()

	if resourceData, err = schema.NewResource(thisSchema.rawSchema, context["resource"].(map[string]interface{})); err != nil {
		return err
	}

	if err = tx.Create(resourceData); err != nil {
		return err
	}

	if err = thisSchema.environment.updateResourceFromContext(resource, context); err != nil {
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

// Update updates a resource
func (thisSchema *Schema) Update(resource interface{}) error {
	var tx transaction.Transaction
	var resourceData *schema.Resource
	var err error

	context := goext.MakeContext().
		WithResource(thisSchema.structToMap(resource)).
		WithResourceID(resourceData.ID()).
		WithSchemaID(thisSchema.ID())

	if err = thisSchema.environment.HandleEvent(goext.PreUpdate, context); err != nil {
		return err
	}

	if tx, err = thisSchema.environment.dataStore.Begin(); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PreUpdateTx, context); err != nil {
		return err
	}

	defer tx.Close()

	if resourceData, err = schema.NewResource(thisSchema.rawSchema, context["resource"].(map[string]interface{})); err != nil {
		return err
	}

	if err = tx.Update(resourceData); err != nil {
		return err
	}

	if err = thisSchema.environment.updateResourceFromContext(resource, context); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PostUpdateTx, context); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if err = tx.Close(); err != nil {
		return err
	}

	return thisSchema.environment.HandleEvent(goext.PostUpdate, context)
}

// Delete deletes resource by ID
func (thisSchema *Schema) Delete(resourceID string) error {
	var tx transaction.Transaction
	var err error

	context := goext.MakeContext().
		WithResourceID(resourceID).
		WithSchemaID(thisSchema.ID())

	if err = thisSchema.environment.HandleEvent(goext.PreDelete, context); err != nil {
		return err
	}

	if tx, err = thisSchema.environment.dataStore.Begin(); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PreDeleteTx, context); err != nil {
		return err
	}

	defer tx.Close()

	if err = tx.Delete(thisSchema.rawSchema, resourceID); err != nil {
		return err
	}

	if err = thisSchema.environment.HandleEvent(goext.PostDeleteTx, context); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if err = tx.Close(); err != nil {
		return err
	}

	return thisSchema.environment.HandleEvent(goext.PostDelete, context)
}

func (thisSchema *Schema) RegisterEventHandler(event string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority goext.Priority) {
	thisSchema.environment.RegisterSchemaEventHandler(thisSchema.rawSchema.ID, event, handler, priority)
}

func (thisSchema *Schema) RegisterResourceType(typeValue interface{}) {
	thisSchema.environment.RegisterResourceType(thisSchema.rawSchema.ID, typeValue)
}

// NewSchemas returns a new implementation of schemas interface
func NewSchemas(environment *Environment) goext.ISchemas {
	return &Schemas{environment: environment}
}
