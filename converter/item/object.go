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

package item

import (
	"fmt"
	"strings"

	"github.com/cloudwan/gohan/converter/crud"
	"github.com/cloudwan/gohan/converter/hash"
	"github.com/cloudwan/gohan/converter/name"
	"github.com/cloudwan/gohan/converter/set"
	"github.com/cloudwan/gohan/converter/util"
)

// Object is an implementation of Item interface
type Object struct {
	objectType string
	properties set.Set
	required   map[string]bool
}

// Empty checks if object has no properties
func (object *Object) Empty() bool {
	return object.properties.Empty()
}

// Copy implementation
func (object *Object) Copy() Item {
	newObject := *object
	return &newObject
}

// ToString implementation
func (object *Object) ToString() string {
	return "#*"
}

// Compress implementation
func (object *Object) Compress(source, destination hash.IHashable) {
	if destinationProperty, ok := destination.(*Property); ok {
		if sourceProperty, ok := source.(*Property); ok {
			if object.properties.Contains(destinationProperty) {
				object.properties.Delete(destinationProperty)
				object.properties.Insert(sourceProperty)
			}
		}
	}
}

// GetChildren implementation
func (object *Object) GetChildren() []hash.IHashable {
	sorted := object.properties.ToArray()
	result := make([]hash.IHashable, len(sorted))
	for i, property := range sorted {
		result[i] = property.(hash.IHashable)
	}
	return result
}

// ChangeName implementation
func (object *Object) ChangeName(mark name.Mark) {
	if mark.Change(&object.objectType) {
		for _, property := range object.properties {
			property.(*Property).ChangeName(mark)
		}
	}
}

// ContainsObject implementation
func (object *Object) ContainsObject() bool {
	return !object.Empty()
}

// IsNull implementation
func (object *Object) IsNull() bool {
	return false
}

// MakeRequired implementation
func (object *Object) MakeRequired() {
}

// Name is a function that allows object to be used as a set element
func (object *Object) Name() string {
	return object.objectType
}

// Default implementation
func (object *Object) Default(suffix string) string {
	if object.properties.Empty() {
		return "map[string]interface{}{}"
	}
	return "Make" + object.getType(suffix) + "()"
}

// Type implementation
func (object *Object) Type(suffix string) string {
	if object.properties.Empty() {
		return "interface{}"
	}
	return "*" + object.getType(suffix)
}

// InterfaceType implementation
func (object *Object) InterfaceType(suffix string) string {
	if object.properties.Empty() {
		return "interface{}"
	}
	return "I" + object.getType(suffix)
}

// AddProperties implementation
func (object *Object) AddProperties(properties set.Set, safe bool) error {
	if properties.Empty() {
		return nil
	}
	if object.properties == nil {
		object.properties = set.New()
	}
	if safe {
		if err := object.properties.SafeInsertAll(properties); err != nil {
			return fmt.Errorf(
				"object %s: multiple properties have the same name",
				object.Name(),
			)
		}
	} else {
		properties.InsertAll(object.properties)
		object.properties = properties
	}
	for _, property := range properties {
		if object.required[property.Name()] {
			newProperty := *property.(*Property)
			if newProperty.MakeRequired() {
				object.properties.Insert(&newProperty)
			}
		}
	}
	return nil
}

// Parse implementation
func (object *Object) Parse(context ParseContext) error {
	defaultValues, _ := context.Defaults.(map[interface{}]interface{})
	level := context.Level
	prefix := context.Prefix
	data := context.Data

	object.objectType = prefix
	object.properties = set.New()
	requiredMap, err := parseRequired(data)
	if err != nil {
		return fmt.Errorf(
			"object %s: %v",
			prefix,
			err,
		)
	}
	if level <= 1 {
		object.required = requiredMap
	}
	properties, ok := data["properties"]
	if !ok {
		return nil
	}
	next, ok := properties.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf(
			"object %s has invalid properties",
			prefix,
		)
	}

	for property, definition := range next {
		strProperty, ok := property.(string)
		if !ok {
			return fmt.Errorf(
				"object %s has property which name is not a string",
				object.Name(),
			)
		}
		newProperty := CreateProperty(strProperty)
		object.properties.Insert(newProperty)
		definitionMap, ok := definition.(map[interface{}]interface{})
		if !ok {
			return fmt.Errorf(
				"object %s has invalid property %s",
				object.Name(),
				strProperty,
			)
		}

		context.Required = requiredMap[strProperty]
		context.Data = definitionMap
		context.Defaults, _ = defaultValues[strProperty]
		if err := newProperty.Parse(context); err != nil {
			return err
		}
	}
	return nil
}

// CollectObjects implementation
func (object *Object) CollectObjects(limit, offset int) (set.Set, error) {
	if limit == 0 {
		return nil, nil
	}
	result := set.New()
	if offset <= 0 {
		result.Insert(object)
	}
	for _, property := range object.properties {
		other, err := property.(*Property).CollectObjects(limit-1, offset-1)
		if err != nil {
			return nil, err
		}
		if err = result.SafeInsertAll(other); err != nil {
			return nil, fmt.Errorf(
				"multiple objects with the same type at object %s",
				object.Name(),
			)
		}
	}
	return result, nil
}

// CollectProperties implementation
func (object *Object) CollectProperties(limit, offset int) (set.Set, error) {
	result := set.New()
	for _, property := range object.properties {
		other, err := property.(*Property).CollectProperties(limit, offset)
		if err != nil {
			return nil, err
		}
		err = result.SafeInsertAll(other)
		if err != nil {
			return nil, fmt.Errorf(
				"multiple properties with the same name at object %s",
				object.Name(),
			)
		}
	}
	return result, nil
}

// GenerateGetter implementation
func (object *Object) GenerateGetter(
	variable,
	argument,
	interfaceSuffix string,
	depth int,
) string {
	return fmt.Sprintf(
		"%s%s %s",
		util.Indent(depth),
		util.ResultPrefix(argument, depth, false),
		variable,
	)
}

// GenerateSetter implementation
func (object *Object) GenerateSetter(
	variable,
	argument,
	typeSuffix string,
	depth int,
) string {
	if object.Empty() {
		return fmt.Sprintf(
			"%s%s = %s",
			util.Indent(depth),
			variable,
			argument,
		)
	}
	return fmt.Sprintf(
		"%s%s, _ = %s.(%s)",
		util.Indent(depth),
		variable,
		argument,
		object.Type(typeSuffix),
	)
}

// GenerateConstructor creates a constructor for an object
func (object *Object) GenerateConstructor(suffix string) string {
	objectType := object.getType(suffix)
	code := fmt.Sprintf(
		"func Make%s() %s {\n\treturn &%s{\n",
		objectType,
		object.Type(suffix),
		objectType,
	)

	properties := object.properties.ToArray()
	for _, property := range properties {
		propertyCode := property.(*Property).GenerateConstructor(suffix)
		if propertyCode != "" {
			code += fmt.Sprintf(
				"\t\t%s,\n",
				propertyCode,
			)
		}
	}
	return code + "\t}\n}\n"
}

// GenerateStruct creates a struct of an object
// with suffix added to type name of each field
func (object *Object) GenerateStruct(suffix string) string {
	code := "type " + object.getType(suffix) + " struct {\n"

	properties := object.properties.ToArray()
	for _, property := range properties {
		code += property.(*Property).GenerateProperty(suffix)
	}
	return code + "}\n"
}

// GenerateMutableInterface creates an interface of an object
// with suffix added to objects type
// this interface can be edited
func (object *Object) GenerateMutableInterface(
	interfaceSuffix,
	typeSuffix string,
) string {
	return fmt.Sprintf(
		"type %s interface {\n\t%s\n}\n",
		object.InterfaceType(typeSuffix),
		object.InterfaceType(interfaceSuffix),
	)
}

// GenerateInterface creates an interface of an object
// with suffix added to objects type
func (object *Object) GenerateInterface(suffix string) string {
	code := "type " + object.InterfaceType(suffix) + " interface {\n"
	properties := object.properties.ToArray()
	for _, property := range properties {
		code += fmt.Sprintf(
			"\t%s\n\t%s\n",
			property.(*Property).GetterHeader(suffix),
			property.(*Property).SetterHeader(suffix, false),
		)
	}
	return code + "}\n"
}

// GenerateImplementation creates an implementation of an objects
// getter and setter methods
func (object *Object) GenerateImplementation(interfaceSuffix, typeSuffix string) string {
	variable := util.VariableName(util.AddName(object.objectType, typeSuffix))
	prefix := fmt.Sprintf(
		"func (%s %s) ",
		variable,
		object.Type(typeSuffix),
	)
	properties := object.properties.ToArray()
	code := ""
	for _, property := range properties {
		code += fmt.Sprintf(
			"%s%s\n\n%s%s\n\n",
			prefix,
			property.(*Property).GenerateGetter(variable, interfaceSuffix),
			prefix,
			property.(*Property).GenerateSetter(variable, interfaceSuffix, typeSuffix),
		)
	}
	return strings.TrimSuffix(code, "\n")
}

//GenerateFetch generates a fetch function for an object
func (object *Object) GenerateFetch(
	packageName,
	suffix string,
	lock,
	raw bool,
) string {
	return crud.GenerateFetch(
		packageName,
		object.getType(suffix),
		object.getTypeName(raw, suffix),
		lock,
		raw,
	)
}

//GenerateList generates a list function for an object
func (object *Object) GenerateList(
	packageName,
	suffix string,
	lock,
	raw bool,
) string {
	return crud.GenerateList(
		packageName,
		object.getType(suffix),
		object.getTypeName(raw, suffix),
		lock,
		raw,
	)
}

func (object *Object) getTypeName(raw bool, suffix string) string {
	if raw {
		return object.Type(suffix)
	}
	return object.InterfaceType(suffix)
}

func parseRequired(data map[interface{}]interface{}) (map[string]bool, error) {
	required, ok := data["required"]
	if !ok {
		return nil, nil
	}
	list, ok := required.([]interface{})
	if !ok {
		return nil, fmt.Errorf("required should be a list of strings")
	}
	result := map[string]bool{}
	for _, element := range list {
		elementString, ok := element.(string)
		if !ok {
			return nil, fmt.Errorf("required should be a list of strings")
		}
		result[elementString] = true
	}
	return result, nil
}

func (object *Object) getType(suffix string) string {
	return util.ToGoName(object.Name(), suffix)
}
