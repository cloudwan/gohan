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
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

//Schema type for defining data type
type Schema struct {
	ID, Plural, Title, Description string
	ParentSchema                   *Schema
	Parent                         string
	NamespaceID                    string
	Namespace                      *Namespace
	Tags                           Tags
	Metadata                       map[string]interface{}
	Policy                         []interface{}
	Prefix                         string
	Properties                     []Property
	Required                       []string
	JSONSchema                     map[string]interface{}
	JSONSchemaOnCreate             map[string]interface{}
	JSONSchemaOnUpdate             map[string]interface{}
	Actions                        []Action
	Singular                       string
	URL                            string
	URLWithParents                 string
	RawData                        interface{}
}

//Schemas is a list of schema
//This struct is needed for json decode
type Schemas struct {
	Schemas []*Schema
}

//Map is a map of schema
type Map map[string]*Schema

type typeAssertionError struct {
	field string
}

func (e *typeAssertionError) Error() string {
	return fmt.Sprintf("Type Assertion Error: invalid schema %v field", e.field)
}

//NewSchema is a constructor for a schema
func NewSchema(id, plural, title, description, singular string) *Schema {
	schema := &Schema{
		ID:          id,
		Title:       title,
		Plural:      plural,
		Description: description,
		Singular:    singular,
	}
	schema.Tags = make(Tags)
	schema.Policy = make([]interface{}, 0)
	schema.Properties = make([]Property, 0)
	schema.Required = make([]string, 0)
	return schema
}

//NewSchemaFromObj is a constructor for a schema by obj
func NewSchemaFromObj(rawTypeData interface{}) (*Schema, error) {
	typeData := rawTypeData.(map[string]interface{})

	metaschema, ok := GetManager().Schema("schema")
	if ok {
		err := metaschema.Validate(metaschema.JSONSchema, typeData)
		if err != nil {
			return nil, err
		}
	}

	id, ok := typeData["id"].(string)
	if !ok {
		return nil, &typeAssertionError{"id"}
	}
	plural, ok := typeData["plural"].(string)
	if !ok {
		return nil, &typeAssertionError{"plural"}
	}
	title, ok := typeData["title"].(string)
	if !ok {
		return nil, &typeAssertionError{"title"}
	}
	prefix, _ := typeData["prefix"].(string)
	url, _ := typeData["url"].(string)
	description, ok := typeData["description"].(string)
	if !ok {
		return nil, &typeAssertionError{"description"}
	}
	parent, _ := typeData["parent"].(string)
	namespaceID, _ := typeData["namespace"].(string)
	jsonSchema, ok := typeData["schema"].(map[string]interface{})
	if !ok {
		return nil, &typeAssertionError{"schema"}
	}

	actions, ok := typeData["actions"].(map[string]interface{})
	actionList := []Action{}
	for actionID, actionBody := range actions {
		action, err := NewActionFromObject(actionID, actionBody)
		if err != nil {
			return nil, err
		}
		actionList = append(actionList, action)
	}

	required, _ := jsonSchema["required"]
	if required == nil {
		required = []interface{}{}
	}
	if parent != "" && jsonSchema["properties"].(map[string]interface{})[FormatParentID(parent)] == nil {
		jsonSchema["properties"].(map[string]interface{})[FormatParentID(parent)] = getParentPropertyObj(parent, parent)
		jsonSchema["propertiesOrder"] = append(jsonSchema["propertiesOrder"].([]interface{}), FormatParentID(parent))
		required = append(required.([]interface{}), FormatParentID(parent))
	}

	jsonSchema["required"] = required

	requiredStrings := []string{}
	for _, req := range required.([]interface{}) {
		requiredStrings = append(requiredStrings, req.(string))
	}

	metadata, _ := typeData["metadata"].(map[string]interface{})
	properties, _ := jsonSchema["properties"].(map[string]interface{})

	policy, _ := typeData["policy"].([]interface{})
	singular, ok := typeData["singular"].(string)
	if !ok {
		return nil, &typeAssertionError{"singular"}
	}

	schema := &Schema{
		ID:                 id,
		Title:              title,
		Parent:             parent,
		NamespaceID:        namespaceID,
		Prefix:             prefix,
		Plural:             plural,
		Policy:             policy,
		Description:        description,
		JSONSchema:         jsonSchema,
		JSONSchemaOnCreate: filterSchemaByPermission(jsonSchema, "create"),
		JSONSchemaOnUpdate: filterSchemaByPermission(jsonSchema, "update"),
		Actions:            actionList,
		Metadata:           metadata,
		RawData:            rawTypeData,
		Singular:           singular,
		URL:                url,
		Required:           requiredStrings,
	}
	//TODO(nati) load tags
	schema.Tags = make(Tags)
	schema.Properties = make([]Property, 0)
	for id, property := range properties {
		required := false
		for _, req := range schema.Required {
			if req == id {
				required = true
				break
			}
		}
		propertyObj, err := NewPropertyFromObj(id, property, required)
		if err != nil {
			return nil, fmt.Errorf("Invalid schema: Properties is missing %v", err)
		}
		schema.Properties = append(schema.Properties, *propertyObj)
	}
	return schema, nil
}

// ParentID returns parent property ID
func (schema *Schema) ParentID() string {
	if schema.Parent == "" {
		return ""
	}
	return FormatParentID(schema.Parent)
}

// GetSingleURL returns a URL for access to a single schema object
func (schema *Schema) GetSingleURL() string {
	return fmt.Sprintf("%s/:id", schema.URL)
}

// GetActionURL returns a URL for access to resources actions
func (schema *Schema) GetActionURL(path string) string {
	return schema.URL + path
}

// GetPluralURL returns a URL for access to all schema objects
func (schema *Schema) GetPluralURL() string {
	return schema.URL
}

// GetSingleURLWithParents returns a URL for access to a single schema object
func (schema *Schema) GetSingleURLWithParents() string {
	return fmt.Sprintf("%s/:id", schema.URLWithParents)
}

// GetPluralURLWithParents returns a URL for access to all schema objects
func (schema *Schema) GetPluralURLWithParents() string {
	return schema.URLWithParents
}

// GetDbTableName returns a name of DB table used for storing schema instances
func (schema *Schema) GetDbTableName() string {
	return schema.ID + "s"
}

// GetParentURL returns Parent URL
func (schema *Schema) GetParentURL() string {
	if schema.Parent == "" {
		return ""
	}

	return schema.ParentSchema.GetParentURL() + "/" + schema.ParentSchema.Plural + "/:" + schema.Parent
}

func filterSchemaByPermission(schema map[string]interface{}, permission string) map[string]interface{} {
	filteredSchema := map[string]interface{}{"type": "object"}
	filteredProperties := map[string]map[string]interface{}{}
	for id, property := range schema["properties"].(map[string]interface{}) {
		propertyMap, ok := property.(map[string]interface{})
		if ok == false {
			continue
		}
		allowedList, ok := propertyMap["permission"]
		if ok == false {
			continue
		}
		allowedStringList, ok := allowedList.([]interface{})
		if ok == false {
			continue
		}
		for _, allowed := range allowedStringList {
			if allowed == permission {
				filteredProperties[id] = propertyMap
			}
		}
	}
	filteredSchema["properties"] = filteredProperties
	if required, ok := schema["required"]; ok {
		filteredSchema["required"] = required
	} else {
		filteredSchema["required"] = []string{}
	}
	if permission != "create" {
		// in case of updates or deletes, we don't expect all required attributes
		filteredSchema["required"] = []string{}
	}
	filteredSchema["additionalProperties"] = false
	return filteredSchema
}

func getParentPropertyObj(title, parent string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"relation":    parent,
		"title":       title,
		"description": "parent object",
		"unique":      false,
		"permission":  []interface{}{"create"},
	}
}

//ValidateOnCreate validates json object using jsoncschema on object creation
func (schema *Schema) ValidateOnCreate(object interface{}) error {
	return schema.Validate(schema.JSONSchemaOnCreate, object)
}

//ValidateOnUpdate validates json object using jsoncschema on object update
func (schema *Schema) ValidateOnUpdate(object interface{}) error {
	return schema.Validate(schema.JSONSchemaOnUpdate, object)
}

//Validate validates json object using jsoncschema
func (schema *Schema) Validate(jsonSchema interface{}, object interface{}) error {
	schemaLoader := gojsonschema.NewGoLoader(jsonSchema)
	documentLoader := gojsonschema.NewGoLoader(object)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}
	if result.Valid() {
		return nil
	}
	errDescription := "Json validation error:"
	for _, err := range result.Errors() {
		errDescription += fmt.Sprintf("\n\t%v,", err)
	}
	return fmt.Errorf(errDescription)
}

//SetParentSchema sets parent schema
func (schema *Schema) SetParentSchema(parentSchema *Schema) {
	schema.ParentSchema = parentSchema
}

// SetNamespace sets namespace
func (schema *Schema) SetNamespace(namespace *Namespace) {
	schema.Namespace = namespace
}

//ParentSchemaPropertyID get property id for parent relation
func (schema *Schema) ParentSchemaPropertyID() string {
	if schema.Parent == "" {
		return ""
	}
	return FormatParentID(schema.Parent)
}

//GetPropertyByID get a property object using ID
func (schema *Schema) GetPropertyByID(id string) (*Property, error) {
	for _, p := range schema.Properties {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("Property with ID %s not found", id)
}

//StateVersioning whether resources created from this schema should track state and config versions
func (schema *Schema) StateVersioning() bool {
	statefulRaw, ok := schema.Metadata["state_versioning"]
	if !ok {
		return false
	}
	stateful, ok := statefulRaw.(bool)
	if !ok {
		return false
	}
	return stateful
}

// FormatParentID ...
func FormatParentID(parent string) string {
	return fmt.Sprintf("%s_id", parent)
}
