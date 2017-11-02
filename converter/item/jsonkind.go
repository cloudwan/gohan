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

import "fmt"

// JSONKind is an implementation of Kind interface
type JSONKind struct {
}

// Type implementation
func (jsonKind *JSONKind) Type(suffix string, item Item) string {
	if item.IsNull() {
		return "goext." + getNullType(suffix, item)
	}
	return item.Type(suffix)
}

// InterfaceType implementation
func (jsonKind *JSONKind) InterfaceType(suffix string, item Item) string {
	if item.IsNull() {
		return jsonKind.Type(suffix, item)
	}
	return item.InterfaceType(suffix)
}

func jsonAnnotation(name string, item Item) string {
	var annotation string
	if item.IsNull() {
		annotation = ",omitempty"
	}
	return fmt.Sprintf(
		"json:\"%s%s\"",
		name,
		annotation,
	)
}

// Annotation implementation
func (jsonKind *JSONKind) Annotation(name string, item Item) string {
	return fmt.Sprintf("`%s`", jsonAnnotation(name, item))
}

// Default implementation
func (jsonKind *JSONKind) Default(suffix string, item Item) string {
	if item.IsNull() {
		return fmt.Sprintf(
			"goext.Make%s(%s)",
			getNullType(suffix, item),
			item.Default(suffix),
		)
	}
	return item.Default(suffix)
}
