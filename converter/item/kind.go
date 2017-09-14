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

// Kind is an interface for a type of property
// it can be either db or json
type Kind interface {
	// Type should return a go type of a property
	// args:
	//   1. string - a suffix added to a type
	//   2. item - an item of a property
	// return:
	//   go type of a property
	Type(string, Item) string

	// Type should return an interface type of a property
	// args:
	//   1. string - a suffix added to a type
	//   2. item - an item of a property
	// return:
	//   interface type of a property
	InterfaceType(string, Item) string

	// Annotation should return an annotation for a property is a go struct
	// args:
	//   1. string - a name of a property
	//   2. item - an item of a property
	// return:
	//   go annotation of a property
	Annotation(string, Item) string

	// Default should return a default value for a property
	// args:
	//   1. string - a name of a property
	//   2. item - an item of a property
	// return:
	//   default value of a property as string
	Default(string, Item) string
}
