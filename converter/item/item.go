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
	"github.com/cloudwan/gohan/converter/hash"
	"github.com/cloudwan/gohan/converter/name"
	"github.com/cloudwan/gohan/converter/set"
	"github.com/cloudwan/gohan/converter/util"
)

// Item is an interface for a type of a variable
type Item interface {
	hash.IHashable

	// Copy should make a copy of an item
	Copy() Item

	// ChangeName should change the name of an item recursively
	// args:
	//   1. name.Mark - mark that changes the items name
	ChangeName(name.Mark)

	// IsNull checks if an item can be null
	// return:
	//   true iff. item can be null
	IsNull() bool

	// MakeRequired should not allow item to be null
	MakeRequired()

	// ContainsObject checks if an item contains an object
	// return:
	//   true iff. item contains an object
	ContainsObject() bool

	// Default should return a default value for an item
	// args:
	//   1. string - a suffix added to a type
	// return:
	//   default value of an item
	Default(string) string

	// Type should return a go type of item
	// args:
	//   1. string - a suffix added to a type
	// return:
	//   type of item with suffix appended
	Type(string) string

	// InterfaceType should return an interface type of item
	// args:
	//   1. string - a suffix added to a type
	// return:
	//   interface type of item with suffix appended
	InterfaceType(string) string

	// AddProperties should add properties to an item
	// args:
	//   1. set.Set [Property] - a set of properties
	//   2. bool - flag; if in the set exists a property with the same type
	//             as one of the items properties, then if flag is set
	//             an error should be returned,
	//             otherwise that property should be ignored
	// return:
	//   1. error during execution
	AddProperties(set.Set, bool) error

	// Parse should create an item from given map
	// args:
	//   1. context - ParseContext; context used for parsing
	// return:
	//   1. error during execution
	Parse(ParseContext) error

	// CollectObjects should return a set of objects contained within an item
	// args:
	//   1. int - limit; how deep to search for an object; starting from 1;
	//            if limit is negative this parameter is ignored.
	//   2. int - offset; from which level gathering objects should begin;
	//            starting from 0;
	// return:
	//   1. set of collected objects
	//   2. error during execution
	// example:
	//   let objects be denoted by o and other items by i
	//   suppose we have the following tree:
	//             o1
	//            / \
	//           o2  o3
	//          /  \   \
	//        o4   o5   o6
	//        / \   \    \
	//       o7  i1  i2   i3
	//
	// CollectObjects(3, 1) should return a set of o2, o3, o4, o5, o6
	// CollectObjects(2, 2) should return an empty set
	// CollectObjects(-1, 4) should return a set of o7
	CollectObjects(int, int) (set.Set, error)

	// CollectProperties should return a set properties contained within an item
	// args:
	//   1. int - limit; how deep to search for a property; starting from 1;
	//            if limit is negative this parameter is ignored.
	//   2. int - offset; from which level gathering properties should begin;
	//            starting from 0;
	// return:
	//   1. set of collected properties
	//   2. error during execution
	CollectProperties(int, int) (set.Set, error)

	// GenerateGetter should return a body of a getter function for given item
	// args:
	//   1. string - variable; a name of a variable to get
	//   2. string - argument; a name of a result
	//   3. string - suffix; a suffix added to items type
	//   4. int - depth; a width of an indent
	// return:
	//   string representing a body of a getter function
	GenerateGetter(string, string, string, int) string

	// GenerateSetter should return a body of a setter function for given item
	// args:
	//   1. string - variable; a name of a variable to set
	//   2. string - argument; a name of an argument of the function
	//   3. string - suffix; a suffix added to items type
	//   4. int - depth; a width of an indent
	// return:
	//   string representing a body of a setter function
	GenerateSetter(string, string, string, int) string
}

// CreateItem is a factory for items
func CreateItem(itemType interface{}) (Item, error) {
	strType, _, err := util.ParseType(itemType)
	if err != nil {
		return nil, err
	}
	return createItemFromString(strType), nil
}

func createItemFromString(itemType string) Item {
	switch itemType {
	case "array":
		return &Array{}
	case "object":
		return &Object{}
	default:
		return &PlainItem{}
	}
}
