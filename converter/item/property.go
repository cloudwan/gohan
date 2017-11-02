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

	"github.com/cloudwan/gohan/converter/hash"
	"github.com/cloudwan/gohan/converter/name"
	"github.com/cloudwan/gohan/converter/set"
	"github.com/cloudwan/gohan/converter/util"
)

// Property is a type for an item with name
type Property struct {
	hasDefault bool
	name       string
	item       Item
	kind       Kind
	mark       name.Mark
}

// CreateProperty is a constructor
func CreateProperty(name string) *Property {
	return &Property{name: name}
}

// ToString implementation
func (property *Property) ToString() string {
	return property.name
}

// Compress implementation
func (property *Property) Compress(source, destination hash.IHashable) {
	if sourceItem, ok := source.(Item); property.item == destination && ok {
		property.item = sourceItem
		property.item.ChangeName(property.mark)
	}
}

// GetChildren implementation
func (property *Property) GetChildren() []hash.IHashable {
	return []hash.IHashable{property.item}
}

// CompressObjects removes duplicate objects
// from an object tree rooted at a property
func (property *Property) CompressObjects() {
	hash.Run(property, 2)
}

// ChangeName should change name of items of a property
func (property *Property) ChangeName(mark name.Mark) {
	property.mark.Update(mark)
	property.item.ChangeName(mark)
}

// Name gets a name of a property
func (property *Property) Name() string {
	return property.name
}

// MakeRequired makes an item in property required
// returns true if property was changed
func (property *Property) MakeRequired() bool {
	if property.item.IsNull() {
		property.item = property.item.Copy()
		property.item.MakeRequired()
		return true
	}
	return false
}

// IsObject checks if an item in property is an object
func (property *Property) IsObject() bool {
	_, ok := property.item.(*Object)
	return ok
}

// Parse creates property from given map, prefix and level
// prefix is used to determine a go type of an item
// level is used to determine a kind of a property
// args:
//   context ParseContext - a context used for parsing
// return:
//   1. error during execution
func (property *Property) Parse(context ParseContext) (err error) {
	level := context.Level
	prefix := context.Prefix
	data := context.Data

	property.getKindFromLevel(level)
	property.mark = name.CreateMark(util.AddName(prefix, ""))

	objectType, ok := data["type"]
	if !ok {
		return fmt.Errorf(
			"property %s does not have a type",
			util.AddName(prefix, property.name),
		)
	}
	property.item, err = CreateItem(objectType)
	if err != nil {
		return fmt.Errorf(
			"property %s: %v",
			util.AddName(prefix, property.name),
			err,
		)
	}

	if property.goName() == "ID" {
		context.Required = true
	}

	if value, ok := data["default"]; ok {
		context.Defaults = value
	}
	property.hasDefault = context.Defaults != nil

	context.Prefix = util.AddName(prefix, property.name)
	context.Level++
	return property.item.Parse(context)
}

// AddProperties adds properties to items of given property
// args:
//   set set.Set [Property] - a set of properties
//   safe bool - flag; if in the set exists a property with the same type
//               as one of the items properties, then if flag is set
//               an error should be returned,
//               otherwise that property should be ignored
// return:
//   1. error during execution
func (property *Property) AddProperties(set set.Set, safe bool) error {
	return property.item.AddProperties(set, safe)
}

// CollectObjects should return a set of objects contained within a property
// args:
//   1. int - limit; how deep to search for an object; starting from 0;
//            if limit is negative this parameter is ignored.
//   2. int - offset; from which level gathering objects should begin;
// return:
//   1. set of collected objects
//   2. error during execution
func (property *Property) CollectObjects(limit, offset int) (set.Set, error) {
	return property.item.CollectObjects(limit, offset)
}

// CollectProperties should return a set properties contained within a property
// args:
//   1. int - limit; how deep to search for a property; starting from 0;
//            if limit is negative this parameter is ignored.
//   2. int - offset; from which level gathering properties should begin;
// return:
//   1. set of collected properties
//   2. error during execution
func (property *Property) CollectProperties(limit, offset int) (set.Set, error) {
	if limit == 0 {
		return nil, nil
	}
	result, err := property.item.CollectProperties(limit-1, offset-1)
	if err != nil {
		return nil, err
	}
	if offset <= 0 {
		if result == nil {
			result = set.New()
		}
		err = result.SafeInsert(property)
		if err != nil {
			return nil, fmt.Errorf(
				"multiple properties with the same name: %s",
				property.name,
			)
		}
	}
	return result, nil
}

// GenerateConstructor creates a constructor for a property
func (property *Property) GenerateConstructor(suffix string) string {
	if !property.hasDefault {
		if property.item != nil && property.item.IsNull() {
			return fmt.Sprintf(
				"%s: goext.MakeNull%s()",
				util.ToGoName(property.name, ""),
				util.ToGoName(strings.TrimSuffix(property.item.Type(suffix), "64"), ""),
			)
		}
		return ""
	}
	return fmt.Sprintf(
		"%s: %s",
		util.ToGoName(property.name, ""),
		property.kind.Default(suffix, property.item),
	)
}

// GenerateProperty creates a property of a go struct from given property
// with suffix added to type name
func (property *Property) GenerateProperty(suffix string) string {
	return fmt.Sprintf(
		"\t%s %s %s\n",
		util.ToGoName(property.name, ""),
		property.kind.Type(suffix, property.item),
		property.kind.Annotation(property.name, property.item),
	)
}

// GetterHeader returns a header of a getter for a property
func (property *Property) GetterHeader(suffix string) string {
	return fmt.Sprintf(
		"%s() %s",
		util.ToGoName("get", property.name),
		property.kind.InterfaceType(suffix, property.item),
	)
}

// SetterHeader returns a header of a setter for a property
func (property *Property) SetterHeader(suffix string, argument bool) string {
	var arg string
	if argument {
		arg = util.VariableName(property.Name()) + " "
	}
	return fmt.Sprintf(
		"%s(%s%s)",
		util.ToGoName("set", property.name),
		arg,
		property.kind.InterfaceType(suffix, property.item),
	)
}

// GenerateGetter returns a getter for a property
func (property *Property) GenerateGetter(
	variable,
	suffix string,
) string {
	return fmt.Sprintf(
		"%s {\n%s\n}",
		property.GetterHeader(suffix),
		property.item.GenerateGetter(
			variable+"."+property.goName(),
			"result",
			suffix,
			1,
		),
	)
}

// GenerateSetter returns a setter for a property
func (property *Property) GenerateSetter(
	variable,
	interfaceSuffix,
	typeSuffix string,
) string {
	return fmt.Sprintf(
		"%s {\n%s\n}",
		property.SetterHeader(interfaceSuffix, true),
		property.item.GenerateSetter(
			variable+"."+property.goName(),
			util.VariableName(property.Name()),
			typeSuffix,
			1,
		),
	)
}

func (property *Property) goName() string {
	return util.ToGoName(property.name, "")
}

func (property *Property) getKindFromLevel(level int) {
	if level <= 1 {
		property.kind = &DBKind{}
	} else {
		property.kind = &JSONKind{}
	}
}
