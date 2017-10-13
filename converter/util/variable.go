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

var keywords = []string{
	"break",
	"case",
	"chan",
	"const",
	"continue",
	"default",
	"defer",
	"else",
	"fallthrough",
	"for",
	"func",
	"go",
	"goto",
	"if",
	"import",
	"interface",
	"map",
	"package",
	"range",
	"return",
	"select",
	"struct",
	"switch",
	"type",
	"var",
}

// ResultPrefix returns a prefix for getter function
func ResultPrefix(argument string, depth int, create bool) string {
	if depth > 1 {
		return fmt.Sprintf("%s =", argument)
	}
	if create {
		return fmt.Sprintf("%s :=", argument)
	}
	return "return"
}

// VariableName gets a variable name from its type
func VariableName(name string) string {
	result := snaker.SnakeToCamelLower(strings.Replace(name, "-", "_", -1))
	for _, keyword := range keywords {
		if result == keyword {
			return result + "Object"
		}
	}
	return result
}

// IndexVariable returns a name of variable used in for loop
func IndexVariable(depth int) rune {
	return rune('i' + depth - 1)
}

// Indent returns an indent with given width
func Indent(width int) string {
	return strings.Repeat("\t", width)
}
