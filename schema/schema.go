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
	"fmt"

	"strings"

	"sort"

	"github.com/cloudwan/gohan/util"
	"github.com/flosch/pongo2"
	"github.com/xeipuuv/gojsonschema"
)

//Schema type for defining data type
type Schema struct {
	ID, Plural, Title, Description string
	Type                           string
	Extends                        []string
	ParentSchema                   *Schema
	Parent                         string
	NamespaceID                    string
	Namespace                      *Namespace
	Metadata                       map[string]interface{}
	Prefix                         string
	Properties                     []Property
	Indexes                        []Index
	JSONSchema                     map[string]interface{}
	JSONSchemaOnCreate             map[string]interface{}
	JSONSchemaOnUpdate             map[string]interface{}
	Actions                        []Action
	Singular                       string
	URL                            string
	URLWithParents                 string
	RawData                        interface{}
	IsolationLevel                 map[string]interface{}
	OnParentDeleteCascade          bool
}

const (
	abstract string = "abstract"
)

// LockPolicy is type lock policy
type LockPolicy int

// LockRelatedResources is type of LockPolicy
const (
	LockRelatedResources LockPolicy = iota
	SkipRelatedResources
	NoLocking
)

//Schemas is a list of schema
//This struct is needed for json decode
type Schemas struct {
	Schemas []*Schema
}

//Map is a map of schema
type Map map[string]*Schema

type typeAssertionError struct {
	field string
}

func (e *typeAssertionError) Error() string {
	return fmt.Sprintf("Type Assertion Error: invalid schema %v field", e.field)
}

//NewSchema is a constructor for a schema
func NewSchema(id, plural, title, description, singular string) *Schema {
	schema := &Schema{
		ID:          id,
		Title:       title,
		Plural:      plural,
		Description: description,
		Singular:    singular,
		Extends:     []string{},
	}
	return schema
}

//NewSchemaFromObj is a constructor for a schema by obj
func NewSchemaFromObj(rawTypeData interface{}) (*Schema, error) {
	metaschema, _ := GetManager().Schema("schema")
	return newSchemaFromObj(rawTypeData, metaschema)
}

func newSchemaFromObj(rawTypeData interface{}, metaschema *Schema) (*Schema, error) {
	typeData := rawTypeData.(map[string]interface{})

	if metaschema != nil {
		err := metaschema.Validate(metaschema.JSONSchema, typeData)
		if err != nil {
			return nil, err
		}
	}

	id := util.MaybeString(typeData["id"])
	if id == "" {
		return nil, &typeAssertionError{"id"}
	}
	plural := util.MaybeString(typeData["plural"])
	if plural == "" {
		return nil, &typeAssertionError{"plural"}
	}
	title := util.MaybeString(typeData["title"])
	if title == "" {
		return nil, &typeAssertionError{"title"}
	}
	description := util.MaybeString(typeData["description"])
	if description == "" {
		return nil, &typeAssertionError{"description"}
	}
	singular := util.MaybeString(typeData["singular"])
	if singular == "" {
		return nil, &typeAssertionError{"singular"}
	}

	schema := NewSchema(id, plural, title, description, singular)

	schema.Prefix = util.MaybeString(typeData["prefix"])
	schema.URL = util.MaybeString(typeData["url"])
	schema.Type = util.MaybeString(typeData["type"])
	schema.Parent = util.MaybeString(typeData["parent"])
	schema.OnParentDeleteCascade, _ = typeData["on_parent_delete_cascade"].(bool)
	schema.NamespaceID = util.MaybeString(typeData["namespace"])
	schema.IsolationLevel = util.MaybeMap(typeData["isolation_level"])
	jsonSchema, ok := typeData["schema"].(map[string]interface{})
	if !ok {
		return nil, &typeAssertionError{"schema"}
	}
	schema.JSONSchema = jsonSchema

	schema.Metadata = util.MaybeMap(typeData["metadata"])
	schema.Extends = util.MaybeStringList(typeData["extends"])

	actions := util.MaybeMap(typeData["actions"])
	schema.Actions = []Action{}
	for actionID, actionBody := range actions {
		action, err := NewActionFromObject(actionID, actionBody)
		if err != nil {
			return nil, err
		}
		schema.Actions = append(schema.Actions, action)
	}

	if err := schema.Init(); err != nil {
		return nil, err
	}
	return schema, nil
}

// PropertyOrder is type of property order
type PropertyOrder struct {
	properties      []Property
	propertiesOrder []string
}

func (p PropertyOrder) Len() int {
	return len(p.properties)
}

func (p PropertyOrder) Swap(i, j int) {
	p.properties[i], p.properties[j] = p.properties[j], p.properties[i]
}

func (p PropertyOrder) Less(i, j int) bool {
	// lookup in propertiesOrder from schema
	// Then "id"
	// Then indexed columns
	// Then columns with relation
	// Then alphabetical order on name

	lhv := p.properties[i]
	rhv := p.properties[j]
	iIdx := util.Index(p.propertiesOrder, lhv.ID)
	jIdx := util.Index(p.propertiesOrder, rhv.ID)
	if iIdx != -1 && jIdx == -1 {
		return true
	} else if iIdx == -1 && jIdx != -1 {
		return false
	} else if iIdx != -1 && jIdx != -1 {
		return iIdx < jIdx
	}
	if lhv.ID == "id" {
		return true
	}
	if lhv.Indexed && !rhv.Indexed {
		return true
	} else if !lhv.Indexed && rhv.Indexed {
		return false
	} else {
		if lhv.Relation != "" && rhv.Relation == "" {
			return true
		} else if lhv.Relation == "" && rhv.Relation != "" {
			return false
		} else {
			return lhv.ID < rhv.ID
		}
	}
}

func (p PropertyOrder) String() string {
	props := ""
	for _, property := range p.properties {
		props += property.ID + ","
	}
	return fmt.Sprintf("Order: %s, Properties: %s", p.propertiesOrder, props)
}

// Init initializes schema
func (schema *Schema) Init() error {
	if schema.IsAbstract() {
		return nil
	}
	jsonSchema := schema.JSONSchema
	parent := schema.Parent

	required := util.MaybeStringList(jsonSchema["required"])
	properties := util.MaybeMap(jsonSchema["properties"])
	indexes := util.MaybeMap(jsonSchema["indexes"])
	propertiesOrder := util.MaybeStringList(jsonSchema["propertiesOrder"])
	if parent != "" && properties[FormatParentID(parent)] == nil {
		properties[FormatParentID(parent)] = getParentPropertyObj(parent, parent)
		propertiesOrder = append(propertiesOrder, FormatParentID(parent))
		required = append(required, FormatParentID(parent))
	}

	jsonSchema["required"] = required

	schema.JSONSchemaOnCreate = filterSchemaByPermission(jsonSchema, "create")
	schema.JSONSchemaOnUpdate = filterSchemaByPermission(jsonSchema, "update")

	schema.Properties = []Property{}
	for id, property := range properties {
		propertyRequired := util.ContainsString(required, id)
		propertyObj, err := NewPropertyFromObj(id, property, propertyRequired)
		if err != nil {
			return fmt.Errorf("Invalid schema: Properties is missing %v", err)
		}
		schema.Properties = append(schema.Properties, *propertyObj)
	}

	schema.Indexes = []Index{}
	for name, property := range indexes {
		indexObj, err := NewIndexFromObj(name, property)
		if err != nil {
			return fmt.Errorf("Invalid schema: err: %v", err)
		}
		schema.Indexes = append(schema.Indexes, *indexObj)
	}

	order := PropertyOrder{propertiesOrder: propertiesOrder, properties: schema.Properties}
	sort.Sort(order)

	for _, property := range schema.Properties {
		if !util.ContainsString(propertiesOrder, property.ID) {
			propertiesOrder = append(propertiesOrder, property.ID)
		}
	}
	jsonSchema["propertiesOrder"] = propertiesOrder

	return nil
}

// IsAbstract checks if this schema is abstract or not
func (schema *Schema) IsAbstract() bool {
	return schema.Type == abstract
}

// ParentID returns parent property ID
func (schema *Schema) ParentID() string {
	if schema.Parent == "" {
		return ""
	}
	return FormatParentID(schema.Parent)
}

// GetSingleURL returns a URL for access to a single schema object
func (schema *Schema) GetSingleURL() string {
	return fmt.Sprintf("%s/:id", schema.URL)
}

// GetActionURL returns a URL for access to resources actions
func (schema *Schema) GetActionURL(path string) string {
	return schema.URL + path
}

// GetActionURLWithParents returns a URL for access to resources actions with parent suffix
func (schema *Schema) GetActionURLWithParents(path string) string {
	return schema.URLWithParents + path
}

// GetPluralURL returns a URL for access to all schema objects
func (schema *Schema) GetPluralURL() string {
	return schema.URL
}

// GetSingleURLWithParents returns a URL for access to a single schema object
func (schema *Schema) GetSingleURLWithParents() string {
	return fmt.Sprintf("%s/:id", schema.URLWithParents)
}

// GetPluralURLWithParents returns a URL for access to all schema objects
func (schema *Schema) GetPluralURLWithParents() string {
	return schema.URLWithParents
}

// GetDbTableName returns a name of DB table used for storing schema instances
func (schema *Schema) GetDbTableName() string {
	config := util.GetConfig()
	if config.GetBool("database/legacy", true) {
		return schema.ID + "s"
	}
	return schema.Plural
}

// GetParentURL returns Parent URL
func (schema *Schema) GetParentURL() string {
	if schema.ParentSchema == nil {
		return ""
	}

	return schema.ParentSchema.GetParentURL() + "/" + schema.ParentSchema.Plural + "/:" + schema.Parent
}

func filterSchemaByPermission(schema map[string]interface{}, permission string) map[string]interface{} {
	filteredSchema := map[string]interface{}{"type": "object"}
	filteredProperties := map[string]map[string]interface{}{}
	filteredRequirements := []string{}
	for id, property := range util.MaybeMap(schema["properties"]) {
		propertyMap := util.MaybeMap(property)
		allowedList := util.MaybeStringList(propertyMap["permission"])
		for _, allowed := range allowedList {
			if allowed == permission {
				filteredProperties[id] = propertyMap
			}
		}
	}

	filteredSchema["properties"] = filteredProperties
	requirements := util.MaybeStringList(schema["required"])

	if permission != "create" {
		// required property is used on only create event
		requirements = []string{}
	}

	for _, requirement := range requirements {
		if _, ok := filteredProperties[requirement]; ok {
			filteredRequirements = append(filteredRequirements, requirement)
		}
	}

	filteredSchema["required"] = filteredRequirements
	filteredSchema["additionalProperties"] = false
	return filteredSchema
}

func getParentPropertyObj(title, parent string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"relation":    parent,
		"title":       title,
		"description": "parent object",
		"unique":      false,
		"permission":  []interface{}{"create"},
	}
}

//ValidateOnCreate validates json object using jsoncschema on object creation
func (schema *Schema) ValidateOnCreate(object interface{}) error {
	return schema.Validate(schema.JSONSchemaOnCreate, object)
}

//ValidateGoOnCreate validates json object using jsoncschema on object creation
func (schema *Schema) ValidateGoOnCreate(resource interface{}) error {
	// FIXME
	return nil
}

//ValidateOnUpdate validates json object using jsoncschema on object update
func (schema *Schema) ValidateOnUpdate(object interface{}) error {
	return schema.Validate(schema.JSONSchemaOnUpdate, object)
}

//Validate validates json object using jsoncschema
func (schema *Schema) Validate(jsonSchema interface{}, object interface{}) error {
	schemaLoader := gojsonschema.NewGoLoader(jsonSchema)
	documentLoader := gojsonschema.NewGoLoader(object)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}
	if result.Valid() {
		return nil
	}
	errDescription := "Json validation error:"
	for _, err := range result.Errors() {
		errDescription += fmt.Sprintf("\n\t%v,", err)
	}
	return fmt.Errorf(errDescription)
}

//SetParentSchema sets parent schema
func (schema *Schema) SetParentSchema(parentSchema *Schema) {
	schema.ParentSchema = parentSchema
}

// SetNamespace sets namespace
func (schema *Schema) SetNamespace(namespace *Namespace) {
	schema.Namespace = namespace
}

//ParentSchemaPropertyID get property id for parent relation
func (schema *Schema) ParentSchemaPropertyID() string {
	if schema.Parent == "" {
		return ""
	}
	return FormatParentID(schema.Parent)
}

//GetPropertyByID get a property object using ID
func (schema *Schema) GetPropertyByID(id string) (*Property, error) {
	for _, p := range schema.Properties {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("Property with ID %s not found", id)
}

//StateVersioning whether resources created from this schema should track state and config versions
func (schema *Schema) StateVersioning() bool {
	statefulRaw, ok := schema.Metadata["state_versioning"]
	if !ok {
		return false
	}
	stateful, ok := statefulRaw.(bool)
	if !ok {
		return false
	}
	return stateful
}

//SyncKeyTemplate - for custom paths in etcd
func (schema *Schema) SyncKeyTemplate() (syncKeyTemplate string, ok bool) {
	syncKeyTemplateRaw, ok := schema.Metadata["sync_key_template"]
	if !ok {
		return
	}
	syncKeyTemplate, ok = syncKeyTemplateRaw.(string)
	return
}

//GenerateCustomPath - returns custom path based on sync_key_template
func (schema *Schema) GenerateCustomPath(data map[string]interface{}) (path string, err error) {
	syncKeyTemplate, ok := schema.SyncKeyTemplate()
	if !ok {
		err = fmt.Errorf("Failed to read sync_key_template from schema %v", schema.URL)
		return
	}
	tpl, err := pongo2.FromString(syncKeyTemplate)
	if err != nil {
		return
	}
	path, err = tpl.Execute(pongo2.Context{}.Update(data))
	return
}

//GetResourceIDFromPath - parse path and gets resourceID from it
func (schema *Schema) GetResourceIDFromPath(schemaPath string) string {
	syncKeyTemplate, ok := schema.SyncKeyTemplate()
	if !ok {
		return strings.TrimPrefix(schemaPath, schema.URL+"/")
	}

	syncKeyTemplateSplit := strings.Split(syncKeyTemplate, "/")
	schemaPathSplit := strings.Split(schemaPath, "/")
	if len(schemaPathSplit) >= len(syncKeyTemplateSplit) {
		resourceID := ""
		for k, v := range syncKeyTemplateSplit {
			if v == "{{id}}" {
				resourceID = schemaPathSplit[k]
				break
			}
		}
		return resourceID
	}
	return strings.TrimPrefix(schemaPath, schema.URL+"/")
}

//GetSchemaByURLPath - gets schema by resource path (from API)
func GetSchemaByURLPath(path string) *Schema {
	for _, schema := range GetManager().Schemas() {
		if strings.HasPrefix(path+"/", schema.URL+"/") {
			return schema
		}
	}
	return nil
}

//GetSchemaByPath - gets schema by sync_key_template path
func GetSchemaByPath(path string) *Schema {
	for _, schema := range GetManager().Schemas() {
		syncKeyTemplate, ok := schema.SyncKeyTemplate()
		if ok {
			if checkIfPathMatchesSyncKeyTemplate(path, syncKeyTemplate) {
				return schema
			}
		} else if strings.HasPrefix(path, schema.URL) {
			return schema
		}
	}
	return nil
}

func checkIfPathMatchesSyncKeyTemplate(path string, syncKeyTemplate string) bool {
	syncKeyTemplateSplit := strings.Split(syncKeyTemplate, "/")
	schemaPathSplit := strings.Split(path, "/")
	if len(schemaPathSplit) == len(syncKeyTemplateSplit) {
		for k, v := range syncKeyTemplateSplit {
			if strings.HasPrefix(v, "{{") {
				continue
			} else if schemaPathSplit[k] != syncKeyTemplateSplit[k] {
				return false
			}
		}
		return true
	}
	return false
}

// FormatParentID ...
func FormatParentID(parent string) string {
	return fmt.Sprintf("%s_id", parent)
}

func (schema *Schema) relatedSchemas() []string {
	schemas := []string{}
	for _, p := range schema.Properties {
		if p.Relation != "" {
			schemas = append(schemas, p.Relation)
		}
	}
	schemas = util.ExtendStringList(schemas, schema.Extends)
	return schemas
}

// Extend extends target schema
func (schema *Schema) Extend(fromSchema *Schema) error {
	if schema.Parent == "" {
		schema.Parent = fromSchema.Parent
	}
	if schema.Prefix == "" {
		schema.Prefix = fromSchema.Prefix
	}
	if schema.URL == "" {
		schema.URL = fromSchema.URL
	}
	if schema.NamespaceID == "" {
		schema.NamespaceID = fromSchema.NamespaceID
	}
	schema.JSONSchema["properties"] = util.ExtendMap(
		util.MaybeMap(schema.JSONSchema["properties"]),
		util.MaybeMap(fromSchema.JSONSchema["properties"]))

	schema.JSONSchema["propertiesOrder"] = util.ExtendStringList(
		util.MaybeStringList(fromSchema.JSONSchema["propertiesOrder"]),
		util.MaybeStringList(schema.JSONSchema["propertiesOrder"]))

	schema.JSONSchema["required"] = util.ExtendStringList(
		util.MaybeStringList(fromSchema.JSONSchema["required"]),
		util.MaybeStringList(schema.JSONSchema["required"]))
MergeAction:
	for _, action := range fromSchema.Actions {
		for _, existingAction := range schema.Actions {
			if action.ID == existingAction.ID {
				continue MergeAction
			}
		}
		schema.Actions = append(schema.Actions, action)
	}
	schema.Metadata = util.ExtendMap(schema.Metadata, fromSchema.Metadata)
	schema.IsolationLevel = util.ExtendMap(schema.IsolationLevel, fromSchema.IsolationLevel)
	return schema.Init()
}

//Titles returns list of Titles
func (schema *Schema) Titles() []string {
	titles := make([]string, 0, len(schema.Properties))
	for _, property := range schema.Properties {
		titles = append(titles, property.Title)
	}
	return titles
}

//JSON returns json format of schema
func (schema *Schema) JSON() map[string]interface{} {
	actions := map[string]interface{}{}
	for _, a := range schema.Actions {
		actions[a.ID] = map[string]interface{}{
			"method": a.Method,
			"path":   a.Path,
			"input":  a.InputSchema,
			"output": a.OutputSchema,
		}
	}
	return map[string]interface{}{
		"id":          schema.ID,
		"plural":      schema.Plural,
		"title":       schema.Title,
		"description": schema.Description,
		"parent":      schema.Parent,
		"singular":    schema.Singular,
		"prefix":      schema.Prefix,
		"url":         schema.URL,
		"namespace":   schema.NamespaceID,
		"schema":      schema.JSONSchema,
		"actions":     actions,
		"metadata":    schema.Metadata,
	}
}

// GetLockingPolicy gets locking policy for given schema and event
func (schema *Schema) GetLockingPolicy(event string) LockPolicy {
	if schema.Metadata["locking_policy"] == nil {
		return NoLocking
	}

	policies := util.MaybeMap(schema.Metadata["locking_policy"])
	policy := util.MaybeString(policies[event])
	switch policy {
	case "lock_related":
		return LockRelatedResources
	case "skip_related":
		return SkipRelatedResources
	case "":
		return NoLocking
	default:
		panic(fmt.Sprintf("Unknown locking policy '%s' for event %s in schema %s", policy, event, schema.ID))
	}
}

// GetActionFromCommand gets action with given id
func (schema *Schema) GetActionFromCommand(command string) *Action {
	for _, action := range schema.Actions {
		if action.ID == command {
			return &action
		}
	}
	return nil
}
