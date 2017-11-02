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

	"github.com/cloudwan/gohan/converter/util"
)

// DBKind is an implementation of Kind interface
type DBKind struct {
}

// Type implementation
func (dbKind *DBKind) Type(suffix string, item Item) string {
	if item.IsNull() {
		return "goext." + getNullType(suffix, item)
	}
	return item.Type(suffix)
}

// InterfaceType implementation
func (dbKind *DBKind) InterfaceType(suffix string, item Item) string {
	if item.IsNull() {
		return dbKind.Type(suffix, item)
	}
	return item.InterfaceType(suffix)
}

func dbAnnotation(name string, item Item) string {
	return fmt.Sprintf("db:\"%s\"", name)
}

// Annotation implementation
func (dbKind *DBKind) Annotation(name string, item Item) string {
	return fmt.Sprintf(
		"`%s %s`",
		dbAnnotation(name, item),
		jsonAnnotation(name, item),
	)
}

// Default implementation
func (dbKind *DBKind) Default(suffix string, item Item) string {
	itemDefault := item.Default(suffix)

	if item.IsNull() {
		return fmt.Sprintf(
			"goext.Make%s(%s)",
			util.ToGoName(item.Type(suffix), ""),
			itemDefault,
		)
	}
	return itemDefault
}

func getNullType(suffix string, item Item) string {
	return util.ToGoName(
		"Maybe",
		strings.TrimSuffix(item.Type(suffix), "64"),
	)
}
