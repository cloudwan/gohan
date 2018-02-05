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

package schema

import (
	"github.com/cloudwan/gohan/converter/crud"
	"github.com/cloudwan/gohan/converter/item"
	"github.com/cloudwan/gohan/converter/set"
)

// Convert converts given maps describing schemas to go structs
// args:
//   other []map[interface{}]interface{} - maps describing schemas than
//                                         should not be converted to go structs
//   toConvert []map[interface{}]interface{} - maps describing schemas that
//                                             should be converted to go structs
//   annotationDB string - annotation added to each field in schemas
//   annotationObject string - annotation added to each field in objects
//   suffix string - suffix added to each type name
// return:
//   1. list of go interfaces as strings
//   2. list of go structs as strings
//   3. list of implementations of interfaces as strings
//   4. error during execution
func Convert(
	other,
	toConvert []map[interface{}]interface{},
	rawSuffix,
	interfaceSuffix,
	packageName string,
) (*Generated, error) {
	otherSet, err := parseAll(other)
	if err != nil {
		return nil, err
	}

	toConvertSet, err := parseAll(toConvert)
	if err != nil {
		return nil, err
	}

	if err = collectSchemas(toConvertSet, otherSet); err != nil {
		return nil, err
	}

	dbObjects := set.New()
	jsonObjects := set.New()
	for _, toConvertSchema := range toConvertSet {
		objectFromSchema, _ := toConvertSchema.(*Schema).collectObjects(1, 0)
		dbObjects.InsertAll(objectFromSchema)
		var object set.Set
		object, err = toConvertSchema.(*Schema).collectObjects(-1, 1)
		if err != nil {
			return nil, err
		}
		jsonObjects.InsertAll(object)
	}

	result := &Generated{}
	generatedCrud := map[bool][]string{}
	for _, rawObject := range dbObjects.ToArray() {
		boolean := []bool{false, true}
		object := rawObject.(*item.Object)

		result.Names = append(result.Names, object.GenerateSchemaName(packageName, "SchemaID"))

		if !object.Empty() {
			for _, raw := range boolean {
				for _, lock := range boolean {
					for _, filter := range boolean {
						generatedCrud[raw] = append(generatedCrud[raw], object.GenerateFetch(
							packageName,
							rawSuffix,
							crud.Params{Raw: raw, Lock: lock, Filter: filter},
						))
					}
					generatedCrud[raw] = append(generatedCrud[raw], object.GenerateList(
						packageName,
						rawSuffix,
						crud.Params{Raw: raw, Lock: lock},
					))
				}
			}
		}
	}
	result.Crud = generatedCrud[false]
	result.RawCrud = generatedCrud[true]

	dbObjects.InsertAll(jsonObjects)
	for _, rawObject := range dbObjects.ToArray() {
		object := rawObject.(*item.Object)
		if !object.Empty() {
			result.RawInterfaces = append(
				result.RawInterfaces,
				object.GenerateInterface(interfaceSuffix),
			)
			result.Interfaces = append(
				result.Interfaces,
				object.GenerateMutableInterface(interfaceSuffix, rawSuffix),
			)
			result.Structs = append(
				result.Structs,
				object.GenerateStruct(rawSuffix),
			)
			result.Implementations = append(
				result.Implementations,
				object.GenerateImplementation(interfaceSuffix, rawSuffix),
			)
			result.Constructors = append(
				result.Constructors,
				object.GenerateConstructor(rawSuffix),
			)
		}
	}

	return result, nil
}
