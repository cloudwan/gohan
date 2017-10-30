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

	"github.com/cloudwan/gohan/converter/defaults"
	"github.com/cloudwan/gohan/converter/hash"
	"github.com/cloudwan/gohan/converter/name"
	"github.com/cloudwan/gohan/converter/set"
	"github.com/cloudwan/gohan/converter/util"
)

// PlainItem is an implementation of Item interface
type PlainItem struct {
	required     bool
	null         bool
	defaultValue defaults.PlainDefaults
	itemType     string
}

// Copy implementation
func (plainItem *PlainItem) Copy() Item {
	newItem := *plainItem
	return &newItem
}

// ToString implementation
func (plainItem *PlainItem) ToString() string {
	defaultValue := ""
	if plainItem.defaultValue != nil {
		defaultValue = plainItem.defaultValue.Write()
	}
	return fmt.Sprintf(
		"#%s,%v,%s",
		plainItem.itemType,
		plainItem.IsNull(),
		defaultValue,
	)
}

// Compress implementation
func (plainItem *PlainItem) Compress(hash.IHashable, hash.IHashable) {
}

// GetChildren implementation
func (plainItem *PlainItem) GetChildren() []hash.IHashable {
	return nil
}

// ChangeName implementation
func (plainItem *PlainItem) ChangeName(mark name.Mark) {
}

// ContainsObject implementation
func (plainItem *PlainItem) ContainsObject() bool {
	return false
}

// IsNull implementation
func (plainItem *PlainItem) IsNull() bool {
	return plainItem.null || !plainItem.required
}

// MakeRequired implementation
func (plainItem *PlainItem) MakeRequired() {
	plainItem.required = true
}

// Default implementation
func (plainItem *PlainItem) Default(suffix string) string {
	return plainItem.defaultValue.Write()
}

// Type implementation
func (plainItem *PlainItem) Type(suffix string) string {
	return plainItem.itemType
}

// InterfaceType implementation
func (plainItem *PlainItem) InterfaceType(suffix string) string {
	return plainItem.itemType
}

// AddProperties implementation
func (plainItem *PlainItem) AddProperties(set set.Set, safe bool) error {
	return fmt.Errorf("cannot add properties to a plain item")
}

// Parse implementation
func (plainItem *PlainItem) Parse(context ParseContext) (err error) {
	defaultValue := context.Defaults
	required := context.Required
	prefix := context.Prefix
	data := context.Data

	objectType, ok := data["type"]
	if !ok {
		return fmt.Errorf(
			"item %s does not have a type",
			prefix,
		)
	}
	plainItem.itemType, plainItem.null, err = util.ParseType(objectType)
	if err != nil {
		err = fmt.Errorf(
			"item %s: %v",
			prefix,
			err,
		)
	}

	if _, ok = data["default"]; ok || required {
		plainItem.required = true
	}

	plainItem.defaultValue = defaults.CreatePlainDefaults(defaultValue)

	return
}

// CollectObjects implementation
func (plainItem *PlainItem) CollectObjects(limit, offset int) (set.Set, error) {
	return nil, nil
}

// CollectProperties implementation
func (plainItem *PlainItem) CollectProperties(limit, offset int) (set.Set, error) {
	return nil, nil
}

// GenerateGetter implementation
func (plainItem *PlainItem) GenerateGetter(
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
func (plainItem *PlainItem) GenerateSetter(
	variable,
	argument,
	typeSuffix string,
	depth int,
) string {
	return fmt.Sprintf(
		"%s%s = %s",
		util.Indent(depth),
		variable,
		argument,
	)
}
