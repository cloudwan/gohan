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

package util

import (
	"fmt"
	"strings"

	"github.com/serenize/snaker"
)

var typeMapping = map[string]string{
	"integer":  "int",
	"number":   "float64",
	"boolean":  "bool",
	"abstract": "object",
}

// TryToAddName creates a snake case name from prefix and suffix
// if prefix is empty empty string is returned
func TryToAddName(prefix, suffix string) string {
	if prefix == "" {
		return ""
	}
	return AddName(prefix, suffix)
}

// AddName creates a snake case name from prefix and suffix
func AddName(prefix, suffix string) string {
	if prefix == "" {
		return suffix
	}
	return prefix + "_" + suffix
}

// ToGoName creates a camel case name from prefix and suffix
func ToGoName(prefix, suffix string) string {
	name := strings.Replace(AddName(prefix, suffix), "-", "_", -1)
	return snaker.SnakeToCamel(name)
}

func mapType(typeName string) string {
	if mappedName, ok := typeMapping[typeName]; ok {
		return mappedName
	}
	return typeName
}

// ParseType converts an interface to a name of a go type
func ParseType(itemType interface{}) (string, bool, error) {
	switch goType := itemType.(type) {
	case string:
		return mapType(goType), false, nil
	case []interface{}:
		var (
			result string
			null   bool
		)
		for _, singleType := range goType {
			if strType, ok := singleType.(string); ok {
				if strType == "null" {
					null = true
				} else if result == "" {
					result = mapType(strType)
				}
			}
		}
		if result != "" {
			return result, null, nil
		}

	}
	return "", false, fmt.Errorf("unsupported type: %T", itemType)
}
