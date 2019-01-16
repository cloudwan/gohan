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

	"github.com/cloudwan/gohan/util"
)

//Property is a definition of each Property
type Property struct {
	ID, Title, Description string
	Type, Format           string
	Properties             []Property
	Items                  *Property
	Relation               string
	RelationColumn         string
	RelationProperty       string
	Unique                 bool
	Nullable               bool
	SQLType                string
	OnDeleteCascade        bool
	Default                interface{}
	DefaultMask            interface{}
	Indexed                bool
	Enum                   []interface{}
}

const ItemPropertyID = "[]"

type PropertyBuilder struct {
	property Property
}

func NewPropertyBuilder(id, title, description, typeID string) *PropertyBuilder {
	return &PropertyBuilder{
		property: Property{
			ID:          id,
			Title:       title,
			Description: description,
			Type:        typeID,
		},
	}
}

func (pb *PropertyBuilder) WithFormat(format string) *PropertyBuilder {
	pb.property.Format = format
	return pb
}

func (pb *PropertyBuilder) WithProperties(properties []Property) *PropertyBuilder {
	pb.property.Properties = properties
	return pb
}

func (pb *PropertyBuilder) WithItems(items *Property) *PropertyBuilder {
	pb.property.Items = items
	return pb
}

func (pb *PropertyBuilder) WithEnum(enum []interface{}) *PropertyBuilder {
	pb.property.Enum = enum
	return pb
}

func (pb *PropertyBuilder) WithRelation(relation, relationColumn, relationProperty string) *PropertyBuilder {
	pb.property.Relation = relation
	pb.property.RelationColumn = relationColumn
	pb.property.RelationProperty = relationProperty
	return pb
}

func (pb *PropertyBuilder) WithUnique(unique bool) *PropertyBuilder {
	pb.property.Unique = unique
	return pb
}

func (pb *PropertyBuilder) WithNullable(nullable bool) *PropertyBuilder {
	pb.property.Nullable = nullable
	return pb
}

func (pb *PropertyBuilder) WithOnDeleteCascade(onDeleteCascade bool) *PropertyBuilder {
	pb.property.OnDeleteCascade = onDeleteCascade
	return pb
}

func (pb *PropertyBuilder) WithIndexed(indexed bool) *PropertyBuilder {
	pb.property.Indexed = indexed
	return pb
}

func (pb *PropertyBuilder) WithSQLType(sqlType string) *PropertyBuilder {
	pb.property.SQLType = sqlType
	return pb
}

func (pb *PropertyBuilder) WithDefaultValue(defaultValue interface{}) *PropertyBuilder {
	pb.property.Default = defaultValue
	return pb
}

func (pb *PropertyBuilder) Build() *Property {
	retProp := &Property{}
	*retProp = pb.property
	retProp.generateDefaultMask()
	return retProp
}

//NewPropertyFromObj make Property  from obj
func NewPropertyFromObj(id string, rawTypeData interface{}, required bool) *Property {
	typeData := rawTypeData.(map[string]interface{})
	title, _ := typeData["title"].(string)
	description, _ := typeData["description"].(string)
	var typeID string
	nullable := false
	switch typeData["type"].(type) {
	case string:
		typeID = typeData["type"].(string)
	case []interface{}:
		for _, typeInt := range typeData["type"].([]interface{}) {
			// type can be either string or list of string. we allow for any type and optional null
			// in order to retrieve right type, we need to skip null
			if typeInt.(string) != "null" {
				typeID = typeInt.(string)
			} else {
				nullable = true
			}
		}
	}

	pb := NewPropertyBuilder(id, title, description, typeID)
	pb.WithNullable(nullable)

	if format, ok := typeData["format"].(string); ok {
		pb.WithFormat(format)
	}

	relation, _ := typeData["relation"].(string)
	relationColumn, _ := typeData["relation_column"].(string)
	relationProperty, _ := typeData["relation_property"].(string)
	pb.WithRelation(relation, relationColumn, relationProperty)

	if unique, ok := typeData["unique"].(bool); ok {
		pb.WithUnique(unique)
	}
	if cascade, ok := typeData["on_delete_cascade"].(bool); ok {
		pb.WithOnDeleteCascade(cascade)
	}

	defaultValue, hasDefaultValue := typeData["default"]
	if hasDefaultValue {
		pb.WithDefaultValue(defaultValue)
	}
	if !required && defaultValue == nil {
		pb.WithNullable(true)
	}

	if sqlType, ok := typeData["sql"].(string); ok {
		pb.WithSQLType(sqlType)
	}
	if indexed, ok := typeData["indexed"].(bool); ok {
		pb.WithIndexed(indexed)
	}

	if itemsRaw, hasItems := typeData["items"]; hasItems && typeID == "array" {
		pb.WithItems(parseItems(itemsRaw))
	}

	if enumRaw, hasEnum := typeData["enum"]; hasEnum {
		pb.WithEnum(parseEnum(enumRaw))
	}

	if typeID == "object" {
		pb.WithProperties(parseSubproperties(typeData))
	}

	return pb.Build()
}

func parseItems(itemsRaw interface{}) *Property {
	switch typedItems := itemsRaw.(type) {
	case map[string]interface{}:
		return NewPropertyFromObj(ItemPropertyID, typedItems, true)
	case []interface{}:
		// In the case of metaschema, this can be an array: it means that elements of the array
		// can be of more than one type. The metaschema does not define a regular resource,
		// so this information would not be used and can be ignored.
		return nil
	default:
		panic(fmt.Sprintf(
			"Invalid \"items\" type in property: should be a map or an array, but is %T",
			itemsRaw,
		))
	}
}

func parseEnum(enumRaw interface{}) []interface{} {
	return enumRaw.([]interface{})
}

func parseSubproperties(typeData map[string]interface{}) []Property {
	required, _ := typeData["required"].([]string)
	properties, _ := typeData["properties"].(map[string]interface{})

	parsedProperties := []Property{}

	for innerPropertyID, innerPropertyDesc := range properties {
		isRequired := util.ContainsString(required, innerPropertyID)
		parsedProperty := NewPropertyFromObj(innerPropertyID, innerPropertyDesc, isRequired)
		parsedProperties = append(parsedProperties, *parsedProperty)
	}

	return parsedProperties
}

func (p *Property) generateDefaultMask() {
	if p.Default != nil {
		p.DefaultMask = p.Default
		return
	}
	if p.Type != "object" {
		p.DefaultMask = nil
		return
	}

	var defaultMask map[string]interface{}
	for _, prop := range p.Properties {
		prop.generateDefaultMask()
		if prop.DefaultMask != nil {
			if defaultMask == nil {
				defaultMask = map[string]interface{}{}
			}
			defaultMask[prop.ID] = prop.DefaultMask
		}
	}

	p.DefaultMask = defaultMask
}
