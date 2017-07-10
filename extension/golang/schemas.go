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

package golang

import (
	"fmt"
	"reflect"

	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/schema"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/twinj/uuid"
	"encoding/json"
)

type schemasBinder struct {
	rawEnvironment *Environment
}

func bindSchemas(rawEnvironment *Environment) goext.ISchemas {
	return &schemasBinder{rawEnvironment: rawEnvironment}
}

func (thisSchemasBinder *schemasBinder) List() []goext.ISchema {
	manager := schema.GetManager()
	result := []goext.ISchema{}
	for _, rawSchema := range manager.OrderedSchemas() {
		result = append(result, bindSchema(thisSchemasBinder.rawEnvironment, rawSchema))
	}
	return result
}

func (thisSchemasBinder *schemasBinder) Find(id string) goext.ISchema {
	manager := schema.GetManager()
	schema, ok := manager.Schema(id)

	if !ok {
		return nil
	}

	return bindSchema(thisSchemasBinder.rawEnvironment, schema)
}

func (thisSchemasBinder *schemasBinder) Env() *goext.Environment {
	return &thisSchemasBinder.rawEnvironment.extEnv
}

type schemaBinder struct {
	rawEnvironment *Environment
	rawSchema      *schema.Schema
}

func bindSchema(rawEnvironment *Environment, rawSchema *schema.Schema) goext.ISchema {
	return &schemaBinder{rawEnvironment: rawEnvironment, rawSchema: rawSchema}
}

func (thisSchemasBinder *schemaBinder) ID() string {
	return thisSchemasBinder.rawSchema.ID
}

func (thisSchemasBinder *schemaBinder) Env() *goext.Environment {
	return &thisSchemasBinder.rawEnvironment.extEnv
}

func (thisSchemasBinder *schemaBinder) structToMap(resource interface{}) map[string]interface{} {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("db")

	for _, property := range thisSchemasBinder.rawSchema.Properties {
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

func (thisSchemasBinder *schemaBinder) structToResource(resource interface{}) (*schema.Resource, error) {
	fieldsMap := thisSchemasBinder.structToMap(resource)
	return schema.NewResource(thisSchemasBinder.rawSchema, fieldsMap)
}

//List - lists resources
func (thisSchemasBinder *schemaBinder) List(resources interface{}) error {
	slicePtrValue := reflect.ValueOf(resources)
	slicePtrType := reflect.TypeOf(resources)
	sliceValue := slicePtrValue.Elem()
	sliceType := slicePtrType.Elem()
	elemType := sliceType.Elem()

	sliceValue.SetLen(0)

	tx, err := thisSchemasBinder.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	data, _, err := tx.List(thisSchemasBinder.rawSchema, transaction.Filter{}, nil, nil)

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

func (thisSchemasBinder *schemaBinder) FetchRelated(resource interface{}, relatedResource interface{}) error {
	for _, property := range thisSchemasBinder.rawSchema.Properties {
		if property.Relation != "" {
			relatedSchema, ok := schema.GetManager().Schema(property.Relation)

			if !ok {
				return fmt.Errorf("Could not get related schema: %s for: %s", property.Relation, thisSchemasBinder.rawSchema.ID)
			}

			mapper := reflectx.NewMapper("db")
			id := mapper.FieldByName(reflect.ValueOf(resource), property.ID).String()

			bindSchema(thisSchemasBinder.rawEnvironment, relatedSchema).Fetch(id, relatedResource)

			return nil
		}
	}

	return nil
}

//Fetch - retrieves resource by ID
func (thisSchemasBinder *schemaBinder) Fetch(id string, res interface{}) error {
	tx, err := thisSchemasBinder.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	filter := transaction.Filter{"id": id}

	data, err := tx.Fetch(thisSchemasBinder.rawSchema, filter)

	if err != nil {
		return err
	}
	resourceType, ok := thisSchemasBinder.rawEnvironment.resourceTypes[thisSchemasBinder.rawSchema.ID]
	if !ok {
		return fmt.Errorf("No type registered for schema title: %s", thisSchemasBinder.rawSchema.ID)
	}
	resource := reflect.ValueOf(res)

	for i := 0; i < resourceType.NumField(); i++ {
		field := resource.Elem().Field(i)

		fieldType := resourceType.Field(i)
		propertyName := fieldType.Tag.Get("db")
		property, err := thisSchemasBinder.rawSchema.GetPropertyByID(propertyName)
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
func (thisSchemasBinder *schemaBinder) Create(resource interface{}) error { //resource should be already created
	context := make(map[string]interface{})
	context["resource"] = thisSchemasBinder.structToMap(resource)
	thisSchemasBinder.rawEnvironment.HandleEvent(goext.PreCreateTx, context)
	tx, err := thisSchemasBinder.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	resourceData, err := thisSchemasBinder.structToResource(resource)

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
	return thisSchemasBinder.rawEnvironment.HandleEvent(goext.PostCreateTx, context)
}

//Update - updates resource
func (thisSchemasBinder *schemaBinder) Update(resource interface{}) error {
	tx, err := thisSchemasBinder.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	resourceData, err := thisSchemasBinder.structToResource(resource)

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
func (thisSchemasBinder *schemaBinder) Delete(id string) error {
	tx, err := thisSchemasBinder.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	err = tx.Delete(thisSchemasBinder.rawSchema, id)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (thisSchemasBinder *schemaBinder) RegisterEventHandler(eventName string, handler func(context goext.Context, resource goext.Resource, environment *goext.Environment) error, priority goext.Priority) {
	thisSchemasBinder.rawEnvironment.RegisterSchemaEventHandler(thisSchemasBinder.rawSchema.ID, eventName, handler, priority)
}

func (thisSchemasBinder *schemaBinder) RegisterResourceType(typeValue interface{}) {
	thisSchemasBinder.rawEnvironment.RegisterResourceType(thisSchemasBinder.rawSchema.ID, typeValue)
}
