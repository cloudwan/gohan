// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwan/gohan/schema"
	"github.com/olekukonko/tablewriter"
)

var errorKey = "error"

func (gohanClientCLI *GohanClientCLI) formatOutput(s *schema.Schema, rawResult interface{}) string {
	if rawResult == nil {
		return ""
	}
	switch gohanClientCLI.opts.outputFormat {
	case outputFormatTable:
		return gohanClientCLI.formatOutputTable(s, rawResult)
	default:
		result, _ := json.MarshalIndent(rawResult, "", "\t")
		return fmt.Sprintf("%s", result)
	}
}

func (gohanClientCLI *GohanClientCLI) formatOutputTable(s *schema.Schema, rawResult interface{}) string {
	buffer := bytes.NewBufferString("")
	for k, v := range rawResult.(map[string]interface{}) {
		if k == errorKey {
			return fmt.Sprintf("%v", v)
		}
		switch v.(type) {
		case []interface{}:
			gohanClientCLI.createResourcesTable(s, buffer, v.([]interface{}))
		case map[string]interface{}:
			gohanClientCLI.createSingleResourceTable(s, buffer, v.(map[string]interface{}))
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return buffer.String()
}

func (gohanClientCLI *GohanClientCLI) createResourcesTable(s *schema.Schema, buffer *bytes.Buffer, resources []interface{}) {
	table := tablewriter.NewWriter(buffer)
	if len(resources) == 0 {
		return
	}

	include := gohanClientCLI.fieldFilter(s)
	titles := make([]string, 0, len(s.Properties))
	for _, property := range s.Properties {
		if include != nil && !include[normField(property.ID, s.ID)] {
			continue
		}

		titles = append(titles, property.Title)
	}
	table.SetHeader(titles)

	for _, rawResource := range resources {
		resourceSlice := []string{}
		resource := rawResource.(map[string]interface{})
		for _, property := range s.Properties {
			if include != nil && !include[normField(property.ID, s.ID)] {
				continue
			}

			v := ""
			if val, ok := resource[property.ID]; ok && val != nil {
				switch property.Type {
				case "string":
					v = fmt.Sprint(val)
					if property.RelationProperty != "" {
						relatedResource := resource[property.RelationProperty].(map[string]interface{})
						v = relatedResource["name"].(string)
					}
				default:
					v = fmt.Sprint(val)
				}
			}
			resourceSlice = append(resourceSlice, v)
		}
		table.Append(resourceSlice)
	}
	table.Render()
}

func (gohanClientCLI *GohanClientCLI) createSingleResourceTable(s *schema.Schema, buffer *bytes.Buffer, resource map[string]interface{}) {
	include := gohanClientCLI.fieldFilter(s)
	table := tablewriter.NewWriter(buffer)
	table.SetHeader([]string{"Property", "Value"})
	for _, property := range s.Properties {
		if include != nil && !include[normField(property.ID, s.ID)] {
			continue
		}

		table.Append([]string{property.Title, fmt.Sprint(resource[property.ID])})
	}
	table.Render()
}

//fieldFilter returns opts filters as string to boolean map with normalised keys.
func (gohanClientCLI *GohanClientCLI) fieldFilter(s *schema.Schema) map[string]bool {
	var include map[string]bool
	if gohanClientCLI.opts.fields != nil {
		include = make(map[string]bool)
		for _, f := range gohanClientCLI.opts.fields {
			include[normField(f, s.ID)] = true
		}
	}

	return include
}

//normField returns field prefixed with schema ID.
func normField(field, schemaID string) string {
	if strings.Contains(field, ".") {
		return field
	}

	return fmt.Sprintf("%s.%s", schemaID, field)
}
