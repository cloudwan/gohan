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
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudwan/gohan/util"
	"github.com/xeipuuv/gojsonschema"

	"github.com/cloudwan/gohan/singleton"
)

const nobodyPrincipal = "Nobody"

//Manager manages handling of schemas
//Manager manages routing with external data
//and gohan resource representation
//This is a singleton class
type Manager struct {
	schemas     Map
	schemaOrder []string
	policies    []*Policy
	Extensions  []*Extension
	TimeLimit   time.Duration         // default time limit for an extension
	TimeLimits  []*PathEventTimeLimit // a list of exceptions for time limits
	namespaces  map[string]*Namespace
	mu          sync.RWMutex
}

func (manager *Manager) String() string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

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

//registerSchema registers new schema for schema manager
func (manager *Manager) registerSchema(schema *Schema) error {
	if _, ok := manager.schemas[schema.ID]; ok {
		log.Warning("Overwriting schema %s", schema.ID)
		return nil
	}
	manager.schemas[schema.ID] = schema
	manager.schemaOrder = append(manager.schemaOrder, schema.ID)
	baseURL := "/"
	if schema.Parent != "" {
		parentSchema, ok := manager.schema(schema.Parent)
		if !ok {
			return fmt.Errorf("Parent schema %s of %s not found", schema.Parent, schema.ID)
		}
		schema.SetParentSchema(parentSchema)
	}
	if schema.NamespaceID != "" {
		namespace, ok := manager.namespace(schema.NamespaceID)
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

//registerNamespace registers a new namespace for schema manager
func (manager *Manager) registerNamespace(namespace *Namespace) error {
	if namespace.Parent != "" {
		parentNamespace, ok := manager.namespace(namespace.Parent)
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
	manager.mu.Lock()
	defer manager.mu.Unlock()

	delete(manager.schemas, schema.ID)
	return nil
}

//Schema gets schema from manager
func (manager *Manager) Schema(id string) (*Schema, bool) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	return manager.schema(id)
}

func (manager *Manager) schema(id string) (*Schema, bool) {
	schema, ok := manager.schemas[id]
	return schema, ok
}

//Schemas gets schema from manager
func (manager *Manager) Schemas() Map {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	m := make(Map, len(manager.schemas))
	for k, v := range manager.schemas {
		m[k] = v
	}

	return m
}

//OrderedSchemas gets schema from manager ordered
func (manager *Manager) OrderedSchemas() []*Schema {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	s := []*Schema{}
	for _, id := range manager.schemaOrder {
		schema, ok := manager.schema(id)
		if ok {
			s = append(s, schema)
		}
	}
	return s
}

//reorderSchema tries reorder schemas using Tarjan's algorithm
type state int

const (
	notVisited state = iota
	visited
	temporaryVisited
)

func visitSchema(schemas []*Schema, schemaList []string, schema *Schema, visitedSchema map[string]state) ([]string, error) {
	if visitedSchema[schema.ID] == temporaryVisited {
		return nil, fmt.Errorf("Schemas aren't DAG. We can't reorder automatically")
	}
	if visitedSchema[schema.ID] == visited {
		return schemaList, nil
	}
	visitedSchema[schema.ID] = temporaryVisited
	relatedSchemas := schema.relatedSchemas()
	var err error
	for _, relatedSchemaID := range relatedSchemas {
		for _, candidate := range schemas {
			if candidate.ID == relatedSchemaID {
				schemaList, err = visitSchema(schemas, schemaList, candidate, visitedSchema)
				if err != nil {
					return nil, err
				}
				break
			}
		}
		if err != nil {
			return nil, err
		}
	}
	visitedSchema[schema.ID] = visited
	schemaList = append(schemaList, schema.ID)
	return schemaList, nil
}

func reorderSchemas(schemas []*Schema) ([]string, error) {
	var err error
	schemaList := []string{}
	visitedSchema := map[string]state{}
	for _, schema := range schemas {
		visitedSchema[schema.ID] = notVisited
	}
	for {
		var schema *Schema
		for _, s := range schemas {
			if visitedSchema[s.ID] == notVisited {
				schema = s
			}
		}
		if schema == nil {
			return schemaList, nil
		}
		schemaList, err = visitSchema(schemas, schemaList, schema, visitedSchema)
		if err != nil {
			return nil, err
		}
	}
}

//Policies gets policies from manager
func (manager *Manager) Policies() []*Policy {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	s := make([]*Policy, len(manager.policies), len(manager.policies))
	for i, m := range manager.policies {
		s[i] = m
	}
	return s
}

// Namespace gets namespace from manager
func (manager *Manager) Namespace(name string) (*Namespace, bool) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	return manager.namespace(name)
}

func (manager *Manager) namespace(name string) (*Namespace, bool) {
	namespace, ok := manager.namespaces[name]
	return namespace, ok
}

//Namespaces gets namespaces from manager
func (manager *Manager) Namespaces() map[string]*Namespace {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	m := make(map[string]*Namespace)
	for k, v := range manager.namespaces {
		m[k] = v
	}
	return m
}

//LoadResource makes resource from datamap
func (manager *Manager) LoadResource(schemaID string, dataMap map[string]interface{}) (*Resource, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	if schema, ok := manager.schema(schemaID); ok {
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

//OrderedLoadSchemasFromFiles calls LoadSchemaFromFile for each file in right order - first abstract then parent and rest on the end
func (manager *Manager) OrderedLoadSchemasFromFiles(filePaths []string) error {
	MaxDepth := 8 // maximum number of nested schemas
	for i := MaxDepth; i > 0 && len(filePaths) > 0; i-- {
		rest := make([]string, 0)
		for _, filePath := range filePaths {
			if filePath == "" {
				continue
			}
			err := manager.LoadSchemaFromFile(filePath)
			if err != nil && err.Error() != "data isn't map" {
				if i == 1 {
					return err
				}
				rest = append(rest, filePath)
			}
		}
		filePaths = rest
	}
	return nil
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
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.loadSchemaFromFile(filePath)
}

//loadSchemaFromFile loads schema from json file - recursive version for nested schemas
func (manager *Manager) loadSchemaFromFile(filePath string) error {
	log.Info("Loading schema %s ...", filePath)
	schemas, err := util.LoadMap(filePath)
	if err != nil {
		return err
	}

	namespaces, _ := schemas["namespaces"].([]interface{})
	for _, namespaceData := range namespaces {
		namespace, err := NewNamespace(namespaceData)
		if err != nil {
			return err
		}
		if err = manager.registerNamespace(namespace); err != nil {
			return err
		}
	}

	schemaObjList := []*Schema{}
	schemaMap := map[string]*Schema{}
	list, _ := schemas["schemas"].([]interface{})
	for _, schemaData := range list {
		if fileName, ok := schemaData.(string); ok {
			err := manager.loadSchemaFromFile(fileName) // recursive call for included files
			if err != nil {
				return err
			}
		} else {
			metaschema, _ := manager.schema("schema")
			schemaObj, err := newSchemaFromObj(schemaData, metaschema)
			if err != nil {
				return err
			}
			schemaMap[schemaObj.ID] = schemaObj
			schemaObjList = append(schemaObjList, schemaObj)
		}
	}
	_, err = reorderSchemas(schemaObjList)
	if err != nil {
		log.Warning("Error in reordering schema %s", err)
	}
	for _, schemaObj := range schemaObjList {
		if schemaObj.IsAbstract() {
			// Register abstract schema
			manager.registerSchema(schemaObj)
		} else {
			for _, baseSchemaID := range schemaObj.Extends {
				baseSchema, ok := manager.schema(baseSchemaID)
				if !ok {
					return fmt.Errorf("Base Schema %s not found", baseSchemaID)
				}
				if !baseSchema.IsAbstract() {
					return fmt.Errorf("Base Schema %s isn't abstract type", baseSchemaID)
				}
				schemaObj.Extend(baseSchema)
			}
		}
	}
	// Reorder schema by relation topology
	schemaOrder, err := reorderSchemas(schemaObjList)
	if err != nil {
		log.Warning("Error in reordering schema %s", err)
	}

	for _, id := range schemaOrder {
		schemaObj := schemaMap[id]
		if !schemaObj.IsAbstract() {
			err = manager.registerSchema(schemaObj)
			if err != nil {
				return err
			}
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
	if extensions == nil {
		return nil
	}

	for _, extensionData := range extensions {
		d := extensionData.(map[string](interface{}))
		rawurl, ok := d["url"].(string)
		if ok {
			d["url"], err = fixRelativeURL(rawurl, filepath.Dir(filePath))
			if err != nil {
				return err
			}
		}

		extension, err := NewExtension(extensionData)
		if err != nil {
			return err
		}
		extension.File = filePath
		manager.Extensions = append(manager.Extensions, extension)
	}

	return nil
}

//LoadPolicies register policy by db object
func (manager *Manager) LoadPolicies(policies []*Resource) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

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
	manager.mu.Lock()
	defer manager.mu.Unlock()

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
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for _, namespaceData := range namespaces {
		namespace, err := NewNamespace(namespaceData.Data())
		if err != nil {
			return err
		}
		manager.registerNamespace(namespace)
	}

	return nil
}

//ClearExtensions clears extensions
func (manager *Manager) ClearExtensions() {
	manager.mu.Lock()
	manager.Extensions = manager.Extensions[:0]
	manager.mu.Unlock()
}

var (
	gohanFormatsRegisteredOnce sync.Once
)

//GetManager get manager
func GetManager() *Manager {
	gohanFormatsRegisteredOnce.Do(func() {
		registerGohanFormats(gojsonschema.FormatCheckers)
	})

	return singleton.Get("schema/manager", func() interface{} {
		return &Manager{
			schemas:     make(Map),
			schemaOrder: []string{},
			namespaces:  map[string]*Namespace{},
			policies:    []*Policy{},
			Extensions:  []*Extension{},
		}
	}).(*Manager)
}

//ClearManager clears manager
func ClearManager() {
	singleton.Clear("schema/manager")
}

//PolicyValidate API request using policy statements
func (manager *Manager) PolicyValidate(action, path string, auth Authorization) (*Policy, *Role) {
	return PolicyValidate(action, path, auth, manager.policies)
}

//NobodyResourcePaths returns a list of paths that do not require authorization
func (manager *Manager) NobodyResourcePaths() []*regexp.Regexp {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	nobodyResourcePaths := []*regexp.Regexp{}
	for _, policy := range manager.policies {
		if policy.Principal == nobodyPrincipal {
			log.Debug("Adding nobody resource path: " + policy.Resource.Path.String())
			nobodyResourcePaths = append(nobodyResourcePaths, policy.Resource.Path)
		}
	}

	return nobodyResourcePaths
}
