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

func bindSchemas(rawEnvironment *Environment) goext.SchemasInterface {
	return &schemasBinder{rawEnvironment: rawEnvironment}
}

func (self *schemasBinder) List() []goext.Schema {
	manager := schema.GetManager()
	result := []goext.Schema{}
	for _, rawSchema := range manager.OrderedSchemas() {
		result = append(result, bindSchema(self.rawEnvironment, rawSchema))
	}
	return result
}

func (self *schemasBinder) Find(id string) goext.Schema {
	manager := schema.GetManager()
	schema, ok := manager.Schema(id)

	if !ok {
		return nil
	}

	return bindSchema(self.rawEnvironment, schema)
}

func (self *schemasBinder) Env() *goext.Environment {
	return &self.rawEnvironment.extEnvironment
}

type schemaBinder struct {
	rawEnvironment *Environment
	rawSchema      *schema.Schema
}

func bindSchema(rawEnvironment *Environment, rawSchema *schema.Schema) goext.Schema {
	return &schemaBinder{rawEnvironment: rawEnvironment, rawSchema: rawSchema}
}

func (self *schemaBinder) ID() string {
	return self.rawSchema.ID
}

func (self *schemaBinder) Env() *goext.Environment {
	return &self.rawEnvironment.extEnvironment
}

func (self *schemaBinder) structToResource(resource interface{}) (*schema.Resource, error) {
	fieldsMap := map[string]interface{}{}

	mapper := reflectx.NewMapper("db")

	for _, property := range self.rawSchema.Properties {
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

	return schema.NewResource(self.rawSchema, fieldsMap)
}

//List - lists resources
func (self *schemaBinder) List(resources interface{}) error {
	slicePtrValue := reflect.ValueOf(resources)
	slicePtrType := reflect.TypeOf(resources)
	sliceValue := slicePtrValue.Elem()
	sliceType := slicePtrType.Elem()
	elemType := sliceType.Elem()

	sliceValue.SetLen(0)

	tx, err := self.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	data, _, err := tx.List(self.rawSchema, transaction.Filter{}, nil, nil)

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

func (self *schemaBinder) FetchRelated(resource interface{}, relatedResource interface{}) error {
	for _, property := range self.rawSchema.Properties {
		if property.Relation != "" {
			relatedSchema, ok := schema.GetManager().Schema(property.Relation)

			if !ok {
				return fmt.Errorf("Could not get related schema: %s for: %s", property.Relation, self.rawSchema.ID)
			}

			mapper := reflectx.NewMapper("db")
			id := mapper.FieldByName(reflect.ValueOf(resource), property.ID).String()

			bindSchema(self.rawEnvironment, relatedSchema).Fetch(id, relatedResource)

			return nil
		}
	}

	return nil
}

//Fetch - retrieves resource by ID
func (self *schemaBinder) Fetch(id string, res interface{}) error {
	tx, err := self.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	filter := transaction.Filter{"id": id}

	data, err := tx.Fetch(self.rawSchema, filter)

	if err != nil {
		return err
	}
	resourceTypePtr, ok := self.rawEnvironment.resourceTypes[self.rawSchema.ID]
	if !ok {
		return fmt.Errorf("No type registered for schema title: %s", self.rawSchema.ID)
	}
	resourceType := resourceTypePtr.Elem()
	resource := reflect.ValueOf(res)

	for i := 0; i < resourceType.NumField(); i++ {
		field := resource.Elem().Field(i)

		fieldType := resourceType.Field(i)
		propertyName := fieldType.Tag.Get("db")
		property, err := self.rawSchema.GetPropertyByID(propertyName)
		if err != nil {
			return err
		}
		if fieldType.Type.Kind() == reflect.Struct {
			mapJson, err := json.Marshal(data.Get(property.ID))
			if err != nil {
				return err
			}
			newField := reflect.New(field.Type())
			fieldJson := string(mapJson)
			fieldIface := newField.Interface()
			err = json.Unmarshal([]byte(fieldJson), &fieldIface)
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
func (self *schemaBinder) Create(resource interface{}) error { //resource should be already created
	tx, err := self.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	resourceData, err := self.structToResource(resource)

	if err != nil {
		return err
	}

	err = tx.Create(resourceData)

	if err != nil {
		return err
	}

	return tx.Commit()
}

//Update - updates resource
func (self *schemaBinder) Update(resource interface{}) error {
	tx, err := self.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	resourceData, err := self.structToResource(resource)

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
func (self *schemaBinder) Delete(id string) error {
	tx, err := self.rawEnvironment.DataStore.Begin()

	if err != nil {
		return err
	}

	defer tx.Close()

	err = tx.Delete(self.rawSchema, id)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (self *schemaBinder) RegisterEventHandler(eventName string, handler func(context goext.Context, resource goext.Resource, environment *goext.Environment) error, priority goext.Priority) {
	self.rawEnvironment.RegisterSchemaEventHandler(self.rawSchema.ID, eventName, handler, priority)
}

func (self *schemaBinder) RegisterResourceType(typeValue interface{}) {
	self.rawEnvironment.RegisterResourceType(self.rawSchema.ID, typeValue)
}
