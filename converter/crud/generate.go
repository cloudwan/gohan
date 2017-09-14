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

package crud

import "fmt"

// GenerateList generates a list function
func GenerateList(
	goextPackage,
	name,
	typeName string,
	lock,
	raw bool,
) string {
	var (
		prefix,
		suffix,
		arg,
		argType string
	)
	if raw {
		suffix = "Raw"
	}
	if lock {
		prefix = "Lock"
		arg = ", policy"
		argType = " " + goextPackage + ".LockPolicy"
	}

	return fmt.Sprintf(
		`func %sList%s%s(`+
			`schema %s.ISchema, `+
			`filter %s.Filter, `+
			`paginator *%s.Paginator, `+
			`context %s.Context%s%s) ([]%s, error) {
	list, err := schema.%sList%s(filter, paginator, context%s)
	if err != nil {
		return nil, err
	}
	result := make([]%s, len(list))
	for i, object := range list {
		result[i] = object.(%s)
	}
	return result, nil
}
`,
		prefix,
		suffix,
		name,
		goextPackage,
		goextPackage,
		goextPackage,
		goextPackage,
		arg,
		argType,
		typeName,
		prefix,
		suffix,
		arg,
		typeName,
		typeName,
	)
}

// GenerateFetch generates a fetch function
func GenerateFetch(
	goextPackage,
	name,
	typeName string,
	lock,
	raw bool,
) string {
	var (
		prefix,
		suffix,
		arg,
		argType string
	)
	if raw {
		suffix = "Raw"
	}
	if lock {
		prefix = "Lock"
		arg = ", policy"
		argType = " " + goextPackage + ".LockPolicy"
	}

	return fmt.Sprintf(
		`func %sFetch%s%s(`+
			`schema %s.ISchema, `+
			`id string, `+
			`context %s.Context%s%s) (%s, error) {
	result, err := schema.%sFetch%s(id, context%s)
	if err != nil {
		return nil, err
	}
	return result.(%s), nil
}
`,
		prefix,
		suffix,
		name,
		goextPackage,
		goextPackage,
		arg,
		argType,
		typeName,
		prefix,
		suffix,
		arg,
		typeName,
	)
}
