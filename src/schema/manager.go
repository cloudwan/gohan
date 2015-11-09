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

package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	"github.com/cloudwan/gohan/util"
)

//manager singleton schema manager
var manager *Manager

//Manager manages handling of schemas
//Manager manages routing with external data
//and gohan resource representation
//This is a singleton class
type Manager struct {
	schemas     Map
	schemaOrder []string
	policies    []*Policy
	Extensions  []*Extension
	namespaces  map[string]*Namespace
}

func (manager *Manager) String() string {
	var str string
	for key, schema := range manager.schemas {
		str += fmt.Sprintf("%s\t[%s]: %s\n", key, schema.Plural, schema.Description)
		str += fmt.Sprintf("Parent: %s\n", schema.Parent)
		str += "--------------------------------------\n"
		//TODO(nati) show properties
		for _, prop := range schema.Properties {
			str += fmt.Sprintf("\t%s\t(%s)\t%s\n", prop.Title, prop.Type, prop.Description)
		}
		str += "\n"
	}
	return str
}

//RegisterSchema registers new schema for schema manager
func (manager *Manager) RegisterSchema(schema *Schema) error {
	if _, ok := manager.schemas[schema.ID]; ok {
		log.Warning("Overwriting schema %s", schema.ID)
	}
	manager.schemas[schema.ID] = schema
	manager.schemaOrder = append(manager.schemaOrder, schema.ID)
	baseURL := "/"
	if schema.Parent != "" {
		parentSchema, ok := manager.Schema(schema.Parent)
		if !ok {
			return fmt.Errorf("Parent schema %s of %s not found", schema.Parent, schema.ID)
		}
		schema.SetParentSchema(parentSchema)
	}
	if schema.NamespaceID != "" {
		namespace, ok := manager.Namespace(schema.NamespaceID)
		if !ok {
			return fmt.Errorf("Namespace schema %s of %s not found", schema.NamespaceID, schema.ID)
		}
		schema.SetNamespace(namespace)
		baseURL = schema.Namespace.GetFullPrefix() + "/"
	}
	if schema.Prefix != "" {
		baseURL = baseURL + strings.TrimPrefix(schema.Prefix, "/") + "/"
	}
	schema.URL = baseURL + schema.Plural
	if schema.Parent != "" {
		schema.URLWithParents = baseURL + strings.TrimPrefix(schema.GetParentURL(), "/") + "/" + schema.Plural
	} else {
		schema.URLWithParents = schema.URL
	}
	return nil
}

// RegisterNamespace registers a new namespace for schema manager
func (manager *Manager) RegisterNamespace(namespace *Namespace) error {
	if namespace.Parent != "" {
		parentNamespace, ok := manager.Namespace(namespace.Parent)
		if !ok {
			return fmt.Errorf("Parent namespace %s of %s not found", namespace.Parent, namespace.ID)
		}
		namespace.SetParentNamespace(parentNamespace)
	}

	manager.namespaces[namespace.ID] = namespace

	return nil
}

//UnRegisterSchema unregister schema
func (manager *Manager) UnRegisterSchema(schema *Schema) error {
	delete(manager.schemas, schema.ID)
	return nil
}

//Schema gets schema from manager
func (manager *Manager) Schema(id string) (schema *Schema, ok bool) {
	schema, ok = manager.schemas[id]
	return
}

//Schemas gets schema from manager
func (manager *Manager) Schemas() Map {
	return manager.schemas
}

//OrderedSchemas gets schema from manager ordered
func (manager *Manager) OrderedSchemas() []*Schema {
	res := []*Schema{}
	for _, id := range manager.schemaOrder {
		schema, ok := manager.Schema(id)
		if ok {
			res = append(res, schema)
		}
	}
	return res
}

//Policies gets policies from manager
func (manager *Manager) Policies() []*Policy {
	return manager.policies
}

// Namespace gets namespace from manager
func (manager *Manager) Namespace(name string) (namespace *Namespace, ok bool) {
	namespace, ok = manager.namespaces[name]
	return
}

//Namespaces gets namespaces from manager
func (manager *Manager) Namespaces() map[string]*Namespace {
	return manager.namespaces
}

//LoadResource makes resource from datamap
func (manager *Manager) LoadResource(schemaID string, dataMap map[string]interface{}) (*Resource, error) {
	if schema, ok := manager.Schema(schemaID); ok {
		return NewResource(schema, dataMap)
	}
	return nil, fmt.Errorf("Schema Not Found: %s", schemaID)
}

//LoadResourceFromJSONString makes resource from jsonString
func (manager *Manager) LoadResourceFromJSONString(schemaID string, jsonData string) (*Resource, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return nil, err
	}
	return manager.LoadResource(schemaID, data)
}

//ValidateSchema validates json schema
func (manager *Manager) ValidateSchema(schemaPath, filePath string) error {
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
	documentLoader := gojsonschema.NewReferenceLoader("file://" + filePath)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		panic(err.Error())
	}
	if result.Valid() {
		return nil
	}
	var errMessage string
	for _, err := range result.Errors() {
		errMessage += fmt.Sprintf("%v   ", err)
	}
	return fmt.Errorf("Invalid json: %s", errMessage)
}

//LoadSchemasFromFiles calls LoadSchemaFromFile for each of provided filePaths
func (manager *Manager) LoadSchemasFromFiles(filePaths ...string) error {
	for _, filePath := range filePaths {
		err := manager.LoadSchemaFromFile(filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

//LoadSchemaFromFile loads schema from json file
func (manager *Manager) LoadSchemaFromFile(filePath string) error {
	log.Info("Loading schema %s ...", filePath)
	schemas, err := util.LoadFile(filePath)
	if err != nil {
		return err
	}

	if !(strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://")) {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		err = os.Chdir(filepath.Dir(filePath))
		if err != nil {
			return err
		}
		defer os.Chdir(workingDirectory)
	}

	namespaces, _ := schemas["namespaces"].([]interface{})
	for _, namespaceData := range namespaces {
		namespace, err := NewNamespace(namespaceData)
		if err != nil {
			return err
		}
		if err = manager.RegisterNamespace(namespace); err != nil {
			return err
		}
	}
	list, _ := schemas["schemas"].([]interface{})
	for _, schemaData := range list {
		schemaObj, err := NewSchemaFromObj(schemaData)
		if err != nil {
			return err
		}
		err = manager.RegisterSchema(schemaObj)
		if err != nil {
			return err
		}
	}
	policies, _ := schemas["policies"].([]interface{})
	if policies != nil {
		for _, policyData := range policies {
			policy, err := NewPolicy(policyData)
			if err != nil {
				return err
			}
			manager.policies = append(manager.policies, policy)
		}
	}
	extensions, _ := schemas["extensions"].([]interface{})
	if extensions != nil {
		for _, extensionData := range extensions {
			extension, err := NewExtension(extensionData)
			if err != nil {
				return err
			}
			manager.Extensions = append(manager.Extensions, extension)
		}
	}
	return nil
}

//LoadPolicies register policy by db object
func (manager *Manager) LoadPolicies(policies []*Resource) error {
	for _, policyData := range policies {
		policy, err := NewPolicy(policyData.Data())
		if err != nil {
			return err
		}
		manager.policies = append(manager.policies, policy)
	}
	return nil
}

//LoadExtensions register extension by db object
func (manager *Manager) LoadExtensions(extensions []*Resource) error {
	for _, extensionData := range extensions {
		extension, err := NewExtension(extensionData.Data())
		if err != nil {
			return err
		}
		manager.Extensions = append(manager.Extensions, extension)
	}
	return nil
}

//LoadNamespaces register namespaces by db object
func (manager *Manager) LoadNamespaces(namespaces []*Resource) error {
	for _, namespaceData := range namespaces {
		namespace, err := NewNamespace(namespaceData.Data())
		if err != nil {
			return err
		}
		manager.RegisterNamespace(namespace)
	}

	return nil
}

//ClearExtensions clears extensions
func (manager *Manager) ClearExtensions() {
	manager.Extensions = []*Extension{}
}

//GetManager get manager
func GetManager() *Manager {
	if manager == nil {
		manager = &Manager{
			schemas:     make(Map),
			schemaOrder: []string{},
			namespaces:  map[string]*Namespace{},
			policies:    []*Policy{},
			Extensions:  []*Extension{},
		}
	}
	registerGohanFormats(gojsonschema.FormatCheckers)
	return manager
}

//ClearManager clears manager
func ClearManager() {
	manager = nil
}

//PolicyValidate API request using policy statements
func (manager *Manager) PolicyValidate(action, path string, auth Authorization) (*Policy, *Role) {
	return PolicyValidate(action, path, auth, manager.policies)
}

//GetSchema returns the schema filtered and trimmed for a specific user or nil when the user shouldn't see it at all
func GetSchema(s *Schema, authorization Authorization) (result *Resource, err error) {
	manager := GetManager()
	metaschema, _ := manager.Schema("schema")
	policy, _ := manager.PolicyValidate("read", s.GetPluralURL(), authorization)
	if policy == nil {
		return
	}
	originalRawSchema := s.RawData.(map[string]interface{})
	rawSchema := map[string]interface{}{}
	for key, value := range originalRawSchema {
		rawSchema[key] = value
	}
	originalSchema := originalRawSchema["schema"].(map[string]interface{})
	schemaSchema := map[string]interface{}{}
	for key, value := range originalSchema {
		schemaSchema[key] = value
	}
	rawSchema["schema"] = schemaSchema
	originalProperties := originalSchema["properties"].(map[string]interface{})
	schemaProperties := map[string]interface{}{}
	for key, value := range originalProperties {
		schemaProperties[key] = value
	}
	var schemaPropertiesOrder []interface{}
	if _, ok := originalSchema["propertiesOrder"]; ok {
		originalPropertiesOrder := originalSchema["propertiesOrder"].([]interface{})
		for _, value := range originalPropertiesOrder {
			schemaPropertiesOrder = append(schemaPropertiesOrder, value)
		}
	}
	var schemaRequired []interface{}
	if _, ok := originalSchema["required"]; ok {
		originalRequired := originalSchema["required"].([]interface{})
		for _, value := range originalRequired {
			schemaRequired = append(schemaRequired, value)
		}
	}
	schemaProperties, schemaPropertiesOrder, schemaRequired = policy.MetaFilter(schemaProperties, schemaPropertiesOrder, schemaRequired)
	schemaSchema["properties"] = schemaProperties
	schemaSchema["propertiesOrder"] = schemaPropertiesOrder
	schemaSchema["required"] = schemaRequired
	result, err = NewResource(metaschema, rawSchema)
	if err != nil {
		log.Warning("%s %s", result, err)
		return
	}
	return
}
