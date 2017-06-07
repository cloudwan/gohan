package schema

import (
	"fmt"
	"strings"
)

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

type IndexType string

const (
	Unique   IndexType = "UNIQUE"
	FullText           = "FULLTEXT"
	Spatial            = "SPATIAL"
	None               = ""
)

//Property is a definition of each Property
type Index struct {
	Name    string
	Columns []string
	Type    IndexType
}

//NewProperty is a constructor for Property type
func NewIndex(name string, columns []string, indexType IndexType) Index {
	return Index{
		Name:    name,
		Columns: columns,
		Type:    indexType,
	}
}

func mapIndexType(rawIndexType string) (IndexType, error) {
	indexType := IndexType(strings.ToUpper(rawIndexType))
	switch indexType {
	case Unique, FullText, Spatial, None:
		return indexType, nil
	default:
		return "", fmt.Errorf("Unknown index type: %s", rawIndexType)
	}
}

//NewPropertyFromObj make Index  from obj
func NewIndexFromObj(name string, rawTypeData interface{}) (*Index, error) {
	typeData := rawTypeData.(map[string]interface{})
	rawIndexType, ok := typeData["type"]
	if !ok {
		rawIndexType = ""
	}
	stringIndexType := rawIndexType.(string)
	indexType, err := mapIndexType(stringIndexType)
	if err != nil {
		return nil, err
	}

	columns := []string{}
	for _, rawColumn := range typeData["columns"].([]interface{}) {
		columnName := rawColumn.(string)
		columns = append(columns, columnName)
	}
	index := NewIndex(name, columns, indexType)
	return &index, nil
}
