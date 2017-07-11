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

func (thisSchemas *Schema) ID() string {
	return thisSchemas.rawSchema.ID
}

func (thisSchemas *Schema) Environment() goext.IEnvironment {
	return thisSchemas.environment
}

func (thisSchemas *Schema) structToMap(resource interface{}) map[string]interface{} {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("db")

	for _, property := range thisSchemas.rawSchema.Properties {
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

func (thisSchemas *Schema) structToResource(resource interface{}) (*schema.Resource, error) {
	fieldsMap := thisSchemas.structToMap(resource)
	return schema.NewResource(thisSchemas.rawSchema, fieldsMap)
}

//List - lists resources
func (thisSchemas *Schema) List(resources interface{}) error {
	slicePtrValue := reflect.ValueOf(resources)
	slicePtrType := reflect.TypeOf(resources)
	sliceValue := slicePtrValue.Elem()
	sliceType := slicePtrType.Elem()
	elemType := sliceType.Elem()

	sliceValue.SetLen(0)

	tx, err := thisSchemas.environment.dataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	data, _, err := tx.List(thisSchemas.rawSchema, transaction.Filter{}, nil, nil)

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

func (thisSchemas *Schema) FetchRelated(resource interface{}, relatedResource interface{}) error {
	for _, property := range thisSchemas.rawSchema.Properties {
		if property.Relation != "" {
			relatedSchema, ok := schema.GetManager().Schema(property.Relation)

			if !ok {
				return fmt.Errorf("Could not get related schema: %s for: %s", property.Relation, thisSchemas.rawSchema.ID)
			}

			mapper := reflectx.NewMapper("db")
			id := mapper.FieldByName(reflect.ValueOf(resource), property.ID).String()

			NewSchema(thisSchemas.environment, relatedSchema).Fetch(id, relatedResource)

			return nil
		}
	}

	return nil
}

//Fetch - retrieves resource by ID
func (thisSchemas *Schema) Fetch(id string, res interface{}) error {
	tx, err := thisSchemas.environment.dataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	filter := transaction.Filter{"id": id}

	data, err := tx.Fetch(thisSchemas.rawSchema, filter)

	if err != nil {
		return err
	}
	resourceType, ok := thisSchemas.environment.resourceTypes[thisSchemas.rawSchema.ID]
	if !ok {
		return fmt.Errorf("No type registered for schema title: %s", thisSchemas.rawSchema.ID)
	}
	resource := reflect.ValueOf(res)

	for i := 0; i < resourceType.NumField(); i++ {
		field := resource.Elem().Field(i)

		fieldType := resourceType.Field(i)
		propertyName := fieldType.Tag.Get("db")
		property, err := thisSchemas.rawSchema.GetPropertyByID(propertyName)
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

//Create - creates resource
func (thisSchemas *Schema) Create(resource interface{}) error { //resource should be already created
	context := make(map[string]interface{})
	context["resource"] = thisSchemas.structToMap(resource)
	thisSchemas.environment.HandleEvent(goext.PreCreateTx, context)
	tx, err := thisSchemas.environment.dataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	resourceData, err := thisSchemas.structToResource(resource)

	if err != nil {
		return err
	}

	err = tx.Create(resourceData)

	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return thisSchemas.environment.HandleEvent(goext.PostCreateTx, context)
}

//Update - updates resource
func (thisSchemas *Schema) Update(resource interface{}) error {
	tx, err := thisSchemas.environment.dataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	resourceData, err := thisSchemas.structToResource(resource)

	if err != nil {
		return err
	}

	err = tx.Update(resourceData)

	if err != nil {
		return err
	}

	return tx.Commit()
}

//Delete - deletes resource by ID
func (thisSchemas *Schema) Delete(id string) error {
	tx, err := thisSchemas.environment.dataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	err = tx.Delete(thisSchemas.rawSchema, id)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (thisSchemas *Schema) RegisterEventHandler(eventName string, handler func(context goext.Context, resource goext.Resource, environment goext.IEnvironment) error, priority goext.Priority) {
	thisSchemas.environment.RegisterSchemaEventHandler(thisSchemas.rawSchema.ID, eventName, handler, priority)
}

func (thisSchemas *Schema) RegisterResourceType(typeValue interface{}) {
	thisSchemas.environment.RegisterResourceType(thisSchemas.rawSchema.ID, typeValue)
}

// NewSchemas returns a new implementation of schemas interface
func NewSchemas(environment *Environment) goext.ISchemas {
	return &Schemas{environment: environment}
}
