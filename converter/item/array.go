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

	"github.com/cloudwan/gohan/converter/hash"
	"github.com/cloudwan/gohan/converter/name"
	"github.com/cloudwan/gohan/converter/set"
	"github.com/cloudwan/gohan/converter/util"
)

// Array is an implementation of Item interface
type Array struct {
	arrayItem Item
}

// Copy implementation
func (array *Array) Copy() Item {
	newArray := *array
	return &newArray
}

// ToString implementation
func (array *Array) ToString() string {
	return "#[]"
}

// Compress implementation
func (array *Array) Compress(source, destination hash.IHashable) {
	if sourceItem, ok := source.(Item); array.arrayItem == destination && ok {
		array.arrayItem = sourceItem
	}
}

// GetChildren implementation
func (array *Array) GetChildren() []hash.IHashable {
	return []hash.IHashable{array.arrayItem}
}

// ChangeName implementation
func (array *Array) ChangeName(mark name.Mark) {
	array.arrayItem.ChangeName(mark)
}

// ContainsObject implementation
func (array *Array) ContainsObject() bool {
	return array.arrayItem.ContainsObject()
}

// IsNull implementation
func (array *Array) IsNull() bool {
	return false
}

// MakeRequired implementation
func (array *Array) MakeRequired() {
}

// Default implementation
func (array *Array) Default(suffix string) string {
	return array.Type(suffix) + "{}"
}

// Type implementation
func (array *Array) Type(suffix string) string {
	return "[]" + array.arrayItem.Type(suffix)
}

// InterfaceType implementation
func (array *Array) InterfaceType(suffix string) string {
	return "[]" + array.arrayItem.InterfaceType(suffix)
}

// AddProperties implementation
func (array *Array) AddProperties(set set.Set, safe bool) error {
	return fmt.Errorf("cannot add properties to an array")
}

// Parse implementation
func (array *Array) Parse(context ParseContext) (err error) {
	prefix := context.Prefix
	data := context.Data

	next, ok := data["items"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf(
			"array %s does not have items",
			prefix,
		)
	}
	objectType, ok := next["type"]
	if !ok {
		return fmt.Errorf(
			"items of array %s do not have a type",
			prefix,
		)
	}
	array.arrayItem, err = CreateItem(objectType)
	if err != nil {
		return fmt.Errorf("array %s: %v", prefix, err)
	}

	context.Data = next
	return array.arrayItem.Parse(context)
}

// CollectObjects implementation
func (array *Array) CollectObjects(limit, offset int) (set.Set, error) {
	return array.arrayItem.CollectObjects(limit, offset)
}

// CollectProperties implementation
func (array *Array) CollectProperties(limit, offset int) (set.Set, error) {
	return array.arrayItem.CollectProperties(limit, offset)
}

// GenerateGetter implementation
func (array *Array) GenerateGetter(
	variable,
	argument,
	interfaceSuffix string,
	depth int,
) string {
	indent := util.Indent(depth)
	var resultSuffix string
	if depth == 1 {
		if !array.ContainsObject() {
			return fmt.Sprintf(
				"%sreturn %s",
				indent,
				variable,
			)
		}
		resultSuffix = fmt.Sprintf(
			"\n%sreturn %s",
			indent,
			argument,
		)
	}
	index := arrayIndex(depth)
	return fmt.Sprintf(
		"%s%s make(%s, len(%s))\n%sfor %c := range %s {\n%s\n%s}%s",
		indent,
		util.ResultPrefix(argument, depth, true),
		array.InterfaceType(interfaceSuffix),
		variable,
		indent,
		util.IndexVariable(depth),
		variable,
		array.arrayItem.GenerateGetter(
			variable+index,
			argument+index,
			interfaceSuffix,
			depth+1,
		),
		indent,
		resultSuffix,
	)
}

// GenerateSetter implementation
func (array *Array) GenerateSetter(
	variable,
	argument,
	typeSuffix string,
	depth int,
) string {
	indent := util.Indent(depth)
	if !array.ContainsObject() {
		return fmt.Sprintf(
			"%s%s = %s",
			indent,
			variable,
			argument,
		)
	}
	index := arrayIndex(depth)
	return fmt.Sprintf(
		"%s%s = make(%s, len(%s))\n%sfor %c := range %s {\n%s\n%s}",
		indent,
		variable,
		array.Type(typeSuffix),
		argument,
		indent,
		util.IndexVariable(depth),
		argument,
		array.arrayItem.GenerateSetter(
			variable+index,
			argument+index,
			typeSuffix,
			depth+1,
		),
		indent,
	)
}

func arrayIndex(depth int) string {
	return fmt.Sprintf("[%c]", util.IndexVariable(depth))
}
