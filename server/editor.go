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

package server

import (
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/mohae/deepcopy"
)

//TrimmedResource returns the schema filtered and trimmed for a specific user or nil when the user shouldn't see it at all
func TrimmedResource(s *schema.Schema, authorization schema.Authorization) (result *schema.Resource) {
	manager := schema.GetManager()
	metaschema, _ := manager.Schema("schema")
	policy, _ := manager.PolicyValidate("read", s.GetPluralURL(), authorization)
	if policy == nil {
		return
	}
	if s.IsAbstract() {
		return
	}
	rawSchema := s.JSON()
	filteredSchema := util.ExtendMap(nil, s.JSONSchema)
	rawSchema["schema"] = filteredSchema
	schemaProperties, schemaPropertiesOrder, schemaRequired := policy.FilterSchema(
		util.MaybeMap(deepcopy.Copy(s.JSONSchema["properties"])),
		util.MaybeStringList(s.JSONSchema["propertiesOrder"]),
		util.MaybeStringList(s.JSONSchema["required"]))

	filteredSchema["properties"] = schemaProperties
	filteredSchema["propertiesOrder"] = schemaPropertiesOrder
	filteredSchema["required"] = schemaRequired

	permission := []string{}
	defaultFilter := schema.CreateExcludeAllFilter()
	for _, action := range schema.AllActions {
		if p, _ := manager.PolicyValidate(action, s.GetPluralURL(), authorization); p != nil {
			permission = append(permission, action)
			filterPermission(action, schemaProperties, p.GetPropertyFilter())
		} else {
			filterPermission(action, schemaProperties, defaultFilter)
		}
	}
	filteredSchema["permission"] = permission

	result = schema.NewResource(metaschema, rawSchema)
	return
}

func filterPermission(action string, schemaProperties map[string]interface{}, filter *schema.Filter) {
	for property, rawPermissions := range schemaProperties {
		permissions := rawPermissions.(map[string]interface{})
		if filter.IsForbidden(property) {
			if _, ok := permissions["permission"]; !ok {
				continue
			}
			permissions["permission"] = removePermission(action, permissions["permission"].([]interface{}))
		}
	}
}

func removePermission(action string, permissions []interface{}) []interface{} {
	permissionIdx := -1

	for idx, permission := range permissions {
		if action == permission.(string) {
			permissionIdx = idx
			break
		}
	}

	if permissionIdx != -1 {
		permissions = append(permissions[:permissionIdx], permissions[permissionIdx+1:]...)
	}

	return permissions
}
