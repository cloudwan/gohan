package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/flosch/pongo2"
	"github.com/mohae/deepcopy"
	"github.com/urfave/cli"
	"github.com/vattle/sqlboiler/strmangle"
)

const (
	configFileFlagName           = "config-file"
	templateFlagName             = "template"
	templateFlagWithShortName    = "template, t"
	splitByResourceFlagName      = "split-by-resource"
	splitByResourceGroupFlagName = "split-by-resource-group"
	policyFlagName               = "policy"
	scopeFlagName                = "scope"
	outputPathFlagName           = "output-path"
	versionFlagName              = "version"
	titleFlagName                = "title"
	descriptionFlagName          = "description"

	schemasPathOpenAPIv3 = "#/components/schemas/"
	schemasPathSwagger   = "#/definitions/"
)

func deleteGohanExtendedProperties(node map[string]interface{}) {
	extendedProperties := [...]string{"unique", "permission", "relation",
		"relation_property", "view", "detail_view", "propertiesOrder",
		"on_delete_cascade", "indexed", "relation_column", "sql", "indexes",
		"patternProperties"}

	for _, extendedProperty := range extendedProperties {
		delete(node, extendedProperty)
	}
}

func fixEnumDefaultValue(node map[string]interface{}) {
	if defaultValue, ok := node["default"]; ok {
		if enums, ok := node["enum"]; ok {
			if defaultValueStr, ok := defaultValue.(string); ok {
				enumsArr := util.MaybeStringList(enums)
				if !util.ContainsString(enumsArr, defaultValueStr) {
					delete(node, "default")
				}
			}
		}
	}
}

func getType(input interface{}) (valueType string, nullable bool, err error) {
	valueType, _ = input.(string)
	if valueType != "" {
		return
	}

	types, _ := input.([]interface{})
	for _, item := range types {
		item := item.(string)
		if item == "null" {
			nullable = true
		} else {
			if valueType != "" {
				return "", false, fmt.Errorf("more than one type in %+v", input)
			}
			valueType = item
		}
	}
	return
}

func fixNullableTypes(node map[string]interface{}) {
	valueType, nullable, err := getType(node["type"])
	if err != nil || valueType == "" {
		return
	}
	node["type"] = valueType
	if nullable {
		node["nullable"] = true
	}
}

func removeEmptyRequiredList(node map[string]interface{}) {
	const requiredProperty = "required"

	if required, ok := node[requiredProperty]; ok {
		switch list := required.(type) {
		case []string:
			if len(list) == 0 {
				delete(node, requiredProperty)
			}
		case []interface{}:
			if len(list) == 0 {
				delete(node, requiredProperty)
			}
		}
	}
}

func removeNotSupportedFormat(node map[string]interface{}) {
	const formatProperty string = "format"
	var allowedFormats = []string{"uri", "uuid", "email", "int32", "int64", "float", "double",
		"byte", "binary", "date", "date-time", "password"}

	if format, ok := node[formatProperty]; ok {
		if format, ok := format.(string); ok {
			if !util.ContainsString(allowedFormats, format) {
				delete(node, formatProperty)
			}
		}
	}
}

func fixRelations(node map[string]interface{}) {
	properties, _ := node["properties"].(map[string]interface{})

	for _, property := range properties {
		property, _ := property.(map[string]interface{})
		relation, _ := property["relation"].(string)
		relationProperty, _ := property["relation_property"].(string)

		if relationProperty != "" && relation != "" {
			properties[relationProperty] = map[string]interface{}{
				"$ref": schemasPathOpenAPIv3 + relation,
			}
		}
	}
}

func fixPropertyTree(node map[string]interface{}) {
	fixNullableTypes(node)
	fixRelations(node)
	deleteGohanExtendedProperties(node)
	fixEnumDefaultValue(node)
	removeEmptyRequiredList(node)
	removeNotSupportedFormat(node)

	for _, value := range node {
		switch childs := value.(type) {
		case map[string]interface{}:
			fixPropertyTree(childs)
		case map[string]map[string]interface{}:
			for _, value := range childs {
				fixPropertyTree(value)
			}
		}
	}
}

func toSwagger(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	value, err := toOpenAPIv3(in, param)
	if err != nil {
		return nil, err
	}
	data, ok := value.Interface().(string)
	if !ok {
		return nil, &pongo2.Error{OrigError: fmt.Errorf("toOpenAPIv3 didn't return string")}
	}

	data = strings.Replace(data, schemasPathOpenAPIv3, schemasPathSwagger, -1)
	return pongo2.AsValue(data), nil
}

func toOpenAPIv3(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := deepcopy.Copy(in.Interface())
	m := i.(map[string]interface{})

	fixPropertyTree(m)

	data, _ := json.MarshalIndent(i, param.String(), "    ")
	return pongo2.AsValue(string(data)), nil
}

func toSwaggerPath(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.String()
	r := regexp.MustCompile(":([^/]+)")
	return pongo2.AsValue(r.ReplaceAllString(i, "{$1}")), nil
}

func hasIDParam(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.String()
	return pongo2.AsValue(strings.Contains(i, ":id")), nil
}

func toNonNullType(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	valueType, _, err := getType(in.Interface())
	if err != nil {
		return nil, &pongo2.Error{OrigError: err}
	}
	if valueType == "" {
		return nil, &pongo2.Error{OrigError: fmt.Errorf("there is no not null type")}
	}
	return pongo2.AsValue(valueType), nil
}

// SnakeToCamel  changes value from snake case to camel case
func SnakeToCamel(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.String()
	return pongo2.AsValue(strmangle.TitleCase(i)), nil
}

func toGoType(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.String()
	switch in.String() {
	case "string":
		return pongo2.AsValue("null.String"), nil
	case "number":
		// #TODO support more format
		return pongo2.AsValue(""), nil
	case "boolean":
		// #TODO support more format
		return pongo2.AsValue("null.Int8"), nil
	case "object":
		// #TODO support more format
		return pongo2.AsValue("interface{}"), nil
	case "array":
		// #TODO support more format
		return pongo2.AsValue("interface{}"), nil
	}
	return pongo2.AsValue(strings.Contains(i, ":id")), nil
}

func init() {
	pongo2.RegisterFilter("openapi3", toOpenAPIv3)
	pongo2.RegisterFilter("swagger", toSwagger)
	pongo2.RegisterFilter("swagger_path", toSwaggerPath)
	pongo2.RegisterFilter("swagger_has_id_param", hasIDParam)
	pongo2.RegisterFilter("to_go_type", toGoType)
	pongo2.RegisterFilter("to_non_null_type", toNonNullType)
	pongo2.RegisterFilter("snake_to_camel", SnakeToCamel)
}

// SchemaWithPolicy ...
type SchemaWithPolicy struct {
	Schema   *schema.Schema
	Policies []string

	JSONSchema         map[string]interface{}
	JSONSchemaOnCreate map[string]interface{}
	JSONSchemaOnUpdate map[string]interface{}
}

func NewSchemaWithPolicy(schema *schema.Schema, policies []string) *SchemaWithPolicy {
	return &SchemaWithPolicy{
		schema,
		policies,
		nil,
		nil,
		nil,
	}
}

func doTemplate(c *cli.Context) {
	template := c.String(templateFlagName)
	manager := schema.GetManager()
	configFile := c.String(configFileFlagName)
	config := util.GetConfig()
	err := config.ReadConfig(configFile)
	if err != nil {
		util.ExitFatal(err)
		return
	}
	templateCode, err := util.GetContent(template)
	if err != nil {
		util.ExitFatal(err)
		return
	}
	pwd, _ := os.Getwd()
	os.Chdir(path.Dir(configFile))
	defer os.Chdir(pwd)
	schemaFiles := config.GetStringList("schemas", nil)
	if schemaFiles == nil {
		util.ExitFatal("No schema specified in configuration")
	} else {
		err = manager.LoadSchemasFromFiles(schemaFiles...)
		if err != nil {
			util.ExitFatal(err)
			return
		}
	}
	schemas := manager.OrderedSchemas()
	schemasMap := manager.Schemas()

	if err != nil {
		util.ExitFatal(err)
		return
	}
	tpl, err := pongo2.FromString(string(templateCode))
	if err != nil {
		util.ExitFatal(err)
		return
	}

	scopes := []schema.Scope{}
	keystoneV2Compatible := false
	for _, scope := range c.StringSlice(scopeFlagName) {
		scopes = append(scopes, schema.Scope(scope))
	}
	if len(scopes) == 0 {
		// default scopes. Cannot use cli "Value" due to the library bug
		// See: https://github.com/urfave/cli/issues/160
		scopes = schema.AllTokenTypes
		keystoneV2Compatible = true
	}

	policies := manager.Policies()
	role := c.String(policyFlagName)
	title := c.String(titleFlagName)
	version := c.String(versionFlagName)
	description := c.String(descriptionFlagName)
	outputPath := c.String(outputPathFlagName)
	schemasWithPolicy := filterSchemasForPolicy(role, scopes, policies, schemas)
	calculateAllowedProperties(schemasWithPolicy, scopes, role, manager, keystoneV2Compatible)
	convertCustomActionOutput(schemasWithPolicy)
	if c.IsSet(splitByResourceGroupFlagName) {
		applyTemplateForEachResourceGroup(schemasWithPolicy, schemasMap, tpl, version, description, outputPath)
	} else if c.IsSet(splitByResourceFlagName) {
		applyTemplateForEachResource(schemasWithPolicy, schemasMap, tpl, outputPath)
	} else {
		applyTemplateForAll(schemasWithPolicy, schemasMap, tpl, title, version, description, outputPath)
	}
}

func calculateAllowedProperties(schemas []*SchemaWithPolicy, scopes []schema.Scope, role string, manager *schema.Manager, keystoneV2Compatible bool) {
	if keystoneV2Compatible {
		auth := schema.NewAuthorizationBuilder().
			WithRoleIDs(role).
			WithKeystoneV2Compatibility().
			BuildScopedToTenant()

		calculateAllowedPropertiesForAuth(schemas, auth, manager)
		return
	}

	for _, scope := range scopes {
		var auth schema.Authorization
		switch scope {
		case schema.AdminScope:
			auth = schema.NewAuthorizationBuilder().
				WithRoleIDs(role).
				BuildAdmin()
		case schema.DomainScope:
			auth = schema.NewAuthorizationBuilder().
				WithRoleIDs(role).
				BuildScopedToDomain()
		case schema.TenantScope:
			auth = schema.NewAuthorizationBuilder().
				WithRoleIDs(role).
				BuildScopedToTenant()
		}
		if auth == nil {
			continue
		}

		calculateAllowedPropertiesForAuth(schemas, auth, manager)
	}
}

func calculateAllowedPropertiesForAuth(schemas []*SchemaWithPolicy, auth schema.Authorization, manager *schema.Manager) {
	for _, sch := range schemas {
		for _, action := range sch.Policies {
			switch action {
			case schema.ActionRead:
				if sch.JSONSchema == nil {
					sch.JSONSchema = sch.Schema.JSONSchema
				}
				filteredProperties := calculateFilteredPropertiesByAction(action, sch.Schema, auth, manager)
				sch.JSONSchema = filterJSONSchemaByProperties(sch.JSONSchema, filteredProperties)
			case schema.ActionCreate:
				if sch.JSONSchemaOnCreate == nil {
					sch.JSONSchemaOnCreate = sch.Schema.JSONSchemaOnCreate
				}
				filteredProperties := calculateFilteredPropertiesByAction(action, sch.Schema, auth, manager)
				sch.JSONSchemaOnCreate = filterJSONSchemaByProperties(sch.JSONSchemaOnCreate, filteredProperties)
			case schema.ActionUpdate:
				if sch.JSONSchemaOnUpdate == nil {
					sch.JSONSchemaOnUpdate = sch.Schema.JSONSchemaOnUpdate
				}
				filteredProperties := calculateFilteredPropertiesByAction(action, sch.Schema, auth, manager)
				sch.JSONSchemaOnUpdate = filterJSONSchemaByProperties(sch.JSONSchemaOnUpdate, filteredProperties)
			}
		}
	}
}

func calculateFilteredPropertiesByAction(action string, schema *schema.Schema, auth schema.Authorization, manager *schema.Manager) []string {
	p, _ := manager.PolicyValidate(action, schema.GetPluralURL(), auth)
	return p.RemoveHiddenPropertyID(schema.PropertyIDs())
}

func filterJSONSchemaByProperties(jsonSchema map[string]interface{}, filterProperty []string) map[string]interface{} {
	resultJSONSchema := deepcopy.Copy(jsonSchema).(map[string]interface{})
	resultJSONSchemaProperties := make(map[string]interface{})

	if jsonSchema["properties"] == nil {
		return resultJSONSchema
	}

	jsonSchemaProperties := jsonSchema["properties"].(map[string]interface{})
	for property, value := range jsonSchemaProperties {
		if util.ContainsString(filterProperty, property) {
			resultJSONSchemaProperties[property] = value
		}
	}
	resultJSONSchema["properties"] = resultJSONSchemaProperties
	return resultJSONSchema
}

func convertCustomActionOutput(schemas []*SchemaWithPolicy) {
	for _, schema := range schemas {
		for _, action := range schema.Schema.Actions {
			if val := action.OutputSchema["type"]; val != nil {
				if stringVal, ok := val.(string); ok && stringVal != "object" {
					delete(action.OutputSchema, "type")
					action.OutputSchema["$ref"] = schemasPathOpenAPIv3 + stringVal
				}
			}
		}
	}
}

func applyTemplateForAll(schemas []*SchemaWithPolicy, schemasMap schema.Map, tpl *pongo2.Template, title, version, description, outputPath string) {
	output, err := tpl.Execute(pongo2.Context{"schemas": schemas, "schemasMap": schemasMap, "title": title, "version": version, "description": description})
	if err != nil {
		util.ExitFatal(err)
		return
	}
	if outputPath == "" {
		fmt.Println(output)
	} else {
		ioutil.WriteFile(outputPath, []byte(output), 0644)
	}
}

func applyTemplateForEachResourceGroup(schemas []*SchemaWithPolicy, schemasMap schema.Map, tpl *pongo2.Template, version, description, outputPath string) {
	for _, resourceGroup := range getAllResourceGroupsFromSchemas(schemas) {
		resourceSchemas := filerSchemasByResourceGroup(resourceGroup, schemas)
		output, err := tpl.Execute(pongo2.Context{"schemas": resourceSchemas, "schemasMap": schemasMap, "title": resourceGroup, "version": version, "description": description})
		if err != nil {
			util.ExitFatal(err)
			return
		}
		outputPath = strings.Replace(outputPath, "__resource__", resourceGroup, 1)
		ioutil.WriteFile(outputPath, []byte(output), 0644)
	}
}

func applyTemplateForEachResource(schemas []*SchemaWithPolicy, schemasMap schema.Map, tpl *pongo2.Template, outputPath string) {
	for _, schemaWithPolicy := range schemas {
		schema := schemaWithPolicy.Schema
		if schema.Metadata["type"] == "metaschema" || schema.Type == "abstract" {
			continue
		}
		output, err := tpl.Execute(pongo2.Context{"schema": schemaWithPolicy, "schemasMap": schemasMap})
		if err != nil {
			util.ExitFatal(err)
			return
		}
		outputPathForResource := strings.Replace(outputPath, "__resource__", schema.ID, 1)
		ioutil.WriteFile(outputPathForResource, []byte(output), 0644)
	}
}

func getAllResourceGroupsFromSchemas(schemas []*SchemaWithPolicy) []string {
	resourcesSet := make(map[string]bool)
	for _, schema := range schemas {
		metadata, _ := schema.Schema.Metadata["resource_group"].(string)
		resourcesSet[metadata] = true
	}
	resources := make([]string, 0, len(resourcesSet))
	for resource := range resourcesSet {
		resources = append(resources, resource)
	}
	return resources
}

func filerSchemasByResourceGroup(resource string, schemas []*SchemaWithPolicy) []*SchemaWithPolicy {
	var filteredSchemas []*SchemaWithPolicy
	for _, schema := range schemas {
		if schema.Schema.Metadata["resource_group"] == resource {
			filteredSchemas = append(filteredSchemas, schema)
		}
	}
	return filteredSchemas
}

func filterSchemasForPolicy(principal string, scopes []schema.Scope, policies []*schema.Policy, schemas []*schema.Schema) []*SchemaWithPolicy {
	var schemasWithPolicy []*SchemaWithPolicy
	if principal == "" {
		for _, sch := range schemas {
			schemasWithPolicy = append(schemasWithPolicy, NewSchemaWithPolicy(sch, schema.AllActions))
		}
		return schemasWithPolicy
	}
	matchedPolicies := filterPolicies(principal, scopes, policies)
	principalNobody := "Nobody"
	nobodyPolicies := filterPolicies(principalNobody, scopes, policies)
	if principal == principalNobody {
		nobodyPolicies = nil
	}
	for _, schemaOriginal := range schemas {
		matchedPolicies := getMatchingPolicy(schemaOriginal, matchedPolicies)
		if len(matchedPolicies) == 0 {
			continue
		}
		schemaCopy := *schemaOriginal
		schemaCopy.Actions = filterActions(schemaOriginal, nobodyPolicies, matchedPolicies)
		matchedPoliciesNames := getListOfPoliciesNames(scopes, matchedPolicies)
		if len(matchedPoliciesNames) == 0 {
			continue
		}
		schemasWithPolicy = append(schemasWithPolicy, NewSchemaWithPolicy(&schemaCopy, matchedPoliciesNames))
	}
	return schemasWithPolicy
}

func getListOfPoliciesNames(scopes []schema.Scope, policies []*schema.Policy) []string {
	var allowedPoliciesNames []string
	var deniedPoliciesNames []string
	for _, policy := range policies {
		var matchScope bool
		for _, scope := range scopes {
			if policy.HasScope(scope) {
				matchScope = true
				break
			}
		}

		if !matchScope {
			continue
		}

		addPolicyNames := schema.AllActions
		if policy.Action != "*" {
			addPolicyNames = []string{policy.Action}
		}
		if policy.IsDeny() {
			deniedPoliciesNames = append(deniedPoliciesNames, addPolicyNames...)
		} else {
			allowedPoliciesNames = append(allowedPoliciesNames, addPolicyNames...)
		}
	}

	var matchedPoliciesNames []string
	for _, policyName := range allowedPoliciesNames {
		if !util.ContainsString(deniedPoliciesNames, policyName) && !util.ContainsString(matchedPoliciesNames, policyName) {
			matchedPoliciesNames = append(matchedPoliciesNames, policyName)
		}
	}
	return matchedPoliciesNames
}

func getMatchingPolicy(schemaUsed *schema.Schema, policies []*schema.Policy) []*schema.Policy {
	var matchedPolicies []*schema.Policy
	for _, policy := range policies {
		if policy.GetResourcePathRegexp().MatchString(schemaUsed.URL) {
			matchedPolicies = append(matchedPolicies, policy)
		}
	}
	return matchedPolicies
}

func filterActions(schemaToFilter *schema.Schema, nobodyPolicies []*schema.Policy, policies []*schema.Policy) []schema.Action {
	actions := make([]schema.Action, 0)
	for _, action := range schemaToFilter.Actions {
		if !hasMatchingPolicy(action, nobodyPolicies) && canUseAction(action, policies, schemaToFilter.URL) {
			actions = append(actions, action)
		}
	}
	return actions
}

func hasMatchingPolicy(action schema.Action, policies []*schema.Policy) bool {
	for _, policy := range policies {
		if action.ID == policy.Action {
			return true
		}
	}
	return false
}

func canUseAction(action schema.Action, policies []*schema.Policy, url string) bool {
	for _, policy := range policies {
		if policy.GetResourcePathRegexp().MatchString(url) && isMatchingPolicy(action, policy) {
			return true
		}
	}
	return false
}

func isMatchingPolicy(action schema.Action, policy *schema.Policy) bool {
	return action.ID == policy.Action || policy.Action == "*" || (policy.Action == "read" && action.Method == "GET") || (policy.Action == "update" && action.Method == "POST")
}

func policyMatchesPrincipal(policy *schema.Policy, principal string) bool {
	return policy.Principal == principal
}

func policyMatchesScopes(policy *schema.Policy, scopes []schema.Scope) bool {
	for _, scope := range scopes {
		if policy.HasScope(scope) {
			return true
		}
	}
	return false
}

func policyMatches(policy *schema.Policy, principal string, scopes []schema.Scope) bool {
	if !policyMatchesPrincipal(policy, principal) {
		return false
	}
	if !policyMatchesScopes(policy, scopes) {
		return false
	}
	return true
}

func filterPolicies(principal string, scopes []schema.Scope, policies []*schema.Policy) []*schema.Policy {
	var matchedPolicies []*schema.Policy
	for _, policy := range policies {
		if policyMatches(policy, principal, scopes) {
			matchedPolicies = append(matchedPolicies, policy)
		}
	}
	return matchedPolicies
}

var defaultScope = cli.StringSlice{string(schema.TenantScope), string(schema.DomainScope), string(schema.AdminScope)}

func getTemplateCommand() cli.Command {
	return cli.Command{
		Name:        "template",
		ShortName:   "template",
		Usage:       "Convert gohan schema using pongo2 template",
		Description: "Convert gohan schema using pongo2 template",
		Flags: []cli.Flag{
			cli.StringFlag{Name: configFileFlagName, Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: templateFlagWithShortName, Value: "", Usage: "Template File"},
			cli.StringFlag{Name: splitByResourceGroupFlagName, Value: "", Usage: "Split output file for each resource groups"},
			cli.StringFlag{Name: splitByResourceFlagName, Value: "", Usage: "Split output file for each resources"},
			cli.StringFlag{Name: outputPathFlagName, Value: "__resource__.json", Usage: "Output Path. You can use __resource__ as a resource name"},
			cli.StringFlag{Name: policyFlagName, Value: "", Usage: "Policy"},
			cli.StringSliceFlag{Name: scopeFlagName, Usage: "Token Scope"},
		},
		Action: doTemplate,
	}
}

func getOpenAPICommand() cli.Command {
	return cli.Command{
		Name:        "openapi",
		ShortName:   "openapi",
		Usage:       "Convert gohan schema to OpenAPI",
		Description: "Convert gohan schema to OpenAPI",
		Flags: []cli.Flag{
			cli.StringFlag{Name: configFileFlagName, Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: templateFlagWithShortName, Value: "embed://etc/templates/openapi.tmpl", Usage: "Template File"},
			cli.StringFlag{Name: splitByResourceGroupFlagName, Value: "", Usage: "Group by resource"},
			cli.StringFlag{Name: policyFlagName, Value: "admin", Usage: "Policy"},
			cli.StringSliceFlag{Name: scopeFlagName, Usage: "Token Scope"},
			cli.StringFlag{Name: versionFlagName, Value: "0.1", Usage: "API version"},
			cli.StringFlag{Name: titleFlagName, Value: "gohan API", Usage: "API title"},
			cli.StringFlag{Name: descriptionFlagName, Value: "", Usage: "API description"},
		},
		Action: doTemplate,
	}
}

func getOpenAPI3Command() cli.Command {
	return cli.Command{
		Name:        "openapi3",
		ShortName:   "openapi3",
		Usage:       "Convert gohan schema to OpenAPI v3",
		Description: "Convert gohan schema to OpenAPI v3",
		Flags: []cli.Flag{
			cli.StringFlag{Name: configFileFlagName, Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: templateFlagWithShortName, Value: "embed://etc/templates/openapi3.tmpl", Usage: "Template File"},
			cli.StringFlag{Name: splitByResourceGroupFlagName, Value: "", Usage: "Group by resource"},
			cli.StringFlag{Name: policyFlagName, Value: "admin", Usage: "Policy"},
			cli.StringSliceFlag{Name: scopeFlagName, Usage: "Token Scope"},
			cli.StringFlag{Name: versionFlagName, Value: "0.1", Usage: "API version"},
			cli.StringFlag{Name: titleFlagName, Value: "gohan API", Usage: "API title"},
			cli.StringFlag{Name: descriptionFlagName, Value: "", Usage: "API description"},
		},
		Action: doTemplate,
	}
}

func getMarkdownCommand() cli.Command {
	return cli.Command{
		Name:        "markdown",
		ShortName:   "markdown",
		Usage:       "Convert gohan schema to markdown doc",
		Description: "Convert gohan schema to markdown doc",
		Flags: []cli.Flag{
			cli.StringFlag{Name: configFileFlagName, Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: templateFlagWithShortName, Value: "embed://etc/templates/markdown.tmpl", Usage: "Template File"},
			cli.StringFlag{Name: splitByResourceGroupFlagName, Value: "", Usage: "Group by resource"},
			cli.StringFlag{Name: policyFlagName, Value: "admin", Usage: "Policy"},
			cli.StringSliceFlag{Name: scopeFlagName, Usage: "Token Scope"},
		},
		Action: doTemplate,
	}
}

func getDotCommand() cli.Command {
	return cli.Command{
		Name:        "dot",
		ShortName:   "dot",
		Usage:       "Convert gohan schema to dot file for graphviz",
		Description: "Convert gohan schema to dot file for graphviz",
		Flags: []cli.Flag{
			cli.StringFlag{Name: configFileFlagName, Value: "gohan.yaml", Usage: "Server config File"},
			cli.StringFlag{Name: templateFlagWithShortName, Value: "embed://etc/templates/dot.tmpl", Usage: "Template File"},
			cli.StringFlag{Name: splitByResourceGroupFlagName, Value: "", Usage: "Group by resource"},
			cli.StringFlag{Name: policyFlagName, Value: "admin", Usage: "Policy"},
			cli.StringSliceFlag{Name: scopeFlagName, Usage: "Token Scope"},
		},
		Action: doTemplate,
	}
}
