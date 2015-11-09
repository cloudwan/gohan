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

package schema

import (
	"encoding/json"
	"fmt"
)

//Tags are additional metadata for resources
type Tags map[string]string // Tags for each resource

//TODO(nati)_define proper error struct.
// Stop using fmt.Errorf

//Resource is a instance of resource
type Resource struct {
	parentID   string
	schema     *Schema
	tags       Tags
	properties map[string]interface{}
}

//ID gets id from resource
func (resource *Resource) ID() string {
	id, _ := resource.properties["id"]
	idString := fmt.Sprint(id)
	return idString
}

//Get gets property from resource
func (resource *Resource) Get(key string) interface{} {
	return resource.properties[key]
}

//ParentID get parent id of the resource
func (resource *Resource) ParentID() string {
	return resource.parentID
}

//SetParentID set parent id of the resource
func (resource *Resource) SetParentID(id string) {
	schema := resource.schema
	if schema.Parent != "" {
		resource.properties[schema.Parent+"_id"] = id
		resource.parentID = id
	}
}

//Path generate path for this resource
func (resource *Resource) Path() string {
	s := resource.Schema()
	key := s.URL + "/" + resource.ID()
	return key
}

//Data gets data from resource
func (resource *Resource) Data() map[string]interface{} {
	return resource.properties
}

//JSONString returns json string of the resource
func (resource *Resource) JSONString() (string, error) {
	bytes, err := json.Marshal(resource.Data())
	return string(bytes), err
}

//Schema gets schema from resource
func (resource *Resource) Schema() *Schema {
	return resource.schema
}

//Values returns list of values
func (resource *Resource) Values() []interface{} {
	var values []interface{}
	schema := resource.schema
	data := resource.properties
	for _, attr := range schema.Properties {
		values = append(values, data[attr.ID])
	}
	return values
}

//NewResource is a constructor for a resource
func NewResource(schema *Schema, properties map[string]interface{}) (*Resource, error) {
	resource := &Resource{
		schema:     schema,
		properties: properties,
	}
	resource.tags = make(Tags)
	if schema.Parent != "" {
		parentID, ok := properties[schema.Parent+"_id"]
		if ok {
			parentIDStr, _ := parentID.(string)
			resource.SetParentID(parentIDStr)
		}
	}
	return resource, nil
}

//String return string form representation
func (resource *Resource) String() string {
	return fmt.Sprintf("%v", resource.properties)
}

//Update resource data
func (resource *Resource) Update(updateData map[string]interface{}) error {
	data := resource.properties
	err := resource.schema.ValidateOnUpdate(updateData)
	if err != nil {
		return err
	}
	for _, property := range resource.schema.Properties {
		id := property.ID
		if val, ok := updateData[id]; ok {
			data[id] = val
		}
	}
	return nil
}

//PopulateDefaults Populates not provided data with defaults
func (resource *Resource) PopulateDefaults() error {
	for _, property := range resource.Schema().Properties {
		if _, ok := resource.properties[property.ID]; !ok && property.Default != nil {
			resource.properties[property.ID] = property.Default
		}
	}
	return nil
}
