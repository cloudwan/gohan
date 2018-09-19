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
	"regexp"
	"strings"

	"github.com/cloudwan/gohan/util"
	"github.com/pkg/errors"
)

const (
	// ActionGlob allows to perform all actions
	ActionGlob = "*"
	// ActionCreate allows to create a resource
	ActionCreate = "create"
	// ActionRead allows to list resources and show details
	ActionRead = "read"
	// ActionUpdate allows to update a resource
	ActionUpdate = "update"
	// ActionDelete allows to delete a resource
	ActionDelete = "delete"
	// ActionAttach allows a resource to have a relation to another resource
	ActionAttach = "__attach__"

	conditionIsOwner       = "is_owner"
	conditionTypeBelongsTo = "belongs_to"
	conditionProperty      = "property"
	conditionOr            = "or"
	conditionAnd           = "and"
	conditionMatch         = "match"

	globalRegexp = ".*"

	onlyOneOfTenantIDTenantNameError = "Only one of [tenant_id, tenant_name] should be specified"
)

// AllActions are all possible actions
var AllActions = []string{ActionCreate, ActionRead, ActionUpdate, ActionDelete}

func newTenant(tenantID, tenantName string) tenant {
	tenantIDRegexp, _ := getRegexp(tenantID)
	tenantNameRegexp, _ := getRegexp(tenantName)
	return tenant{ID: tenantIDRegexp, Name: tenantNameRegexp}
}

// Tenant ...
type tenant struct {
	ID   *regexp.Regexp
	Name *regexp.Regexp
}

func (t *tenant) equal(t2 tenant) bool {
	idMatch := t.ID.MatchString(t2.ID.String()) || t2.ID.MatchString(t.ID.String())
	nameMatch := t.Name.MatchString(t2.Name.String()) || t2.Name.MatchString(t.Name.String())
	return idMatch && nameMatch
}

func (t *tenant) notEqual(t2 tenant) bool {
	return !t.equal(t2)
}

func (t tenant) String() string {
	return fmt.Sprintf("%s (%s)", t.Name.String(), t.ID.String())
}

type conditionFilterType int

const (
	orFilter  conditionFilterType = iota
	andFilter conditionFilterType = iota
)

type conditionFilter struct {
	isOwner    bool
	matches    []map[string]interface{}
	orFilters  *conditionFilter
	andFilters *conditionFilter
	filterType conditionFilterType
}

func makeConditionFilter(filterType conditionFilterType) *conditionFilter {
	return &conditionFilter{
		filterType: filterType,
		matches:    make([]map[string]interface{}, 0),
		orFilters:  nil,
		andFilters: nil,
		isOwner:    false,
	}
}

//Policy describes policy configuration for APIs
type Policy struct {
	ID, Description, Principal, Action, Effect string
	resource                                   *resourceFilter
	tenantID                                   *regexp.Regexp
	tenantName                                 *regexp.Regexp
	currentResourceCondition                   *ResourceCondition
	relationPropertyName                       string
	otherResourceCondition                     *ResourceCondition
}

// Additional information for the "attach" action
type AttachInfo struct {
	SchemaID               string
	OtherResourceCondition *ResourceCondition
	RelationPropertyName   string
}

type ResourceCondition struct {
	Condition                     []interface{}
	actionTenantFilter            map[string][]tenant
	actionPropertyConditionFilter map[string][]map[string]interface{}
	actionFilter                  *conditionFilter
	requireOwner                  bool
}

//resourceFilter describes which resources should be filtered, and what properties are allowed
type resourceFilter struct {
	PropertiesFilter *Filter
	Path             *regexp.Regexp
}

//Authorization interface
type Authorization interface {
	TenantID() string
	TenantName() string
	AuthToken() string
	Roles() []*Role
	Catalog() []*Catalog
}

//BaseAuthorization is base struct for Authorization
type BaseAuthorization struct {
	tenantID   string
	tenantName string
	authToken  string
	roles      []*Role
	catalog    []*Catalog
}

//NewAuthorization is a constructor for auth info
func NewAuthorization(tenantID, tenantName, authToken string, roleIDs []string, catalog []*Catalog) Authorization {
	roles := []*Role{}
	for _, roleID := range roleIDs {
		roles = append(roles, &Role{Name: roleID})
	}
	return &BaseAuthorization{
		tenantID:   tenantID,
		roles:      roles,
		tenantName: tenantName,
		authToken:  authToken,
		catalog:    catalog,
	}
}

//Roles returns authorized roles
func (auth *BaseAuthorization) Roles() []*Role {
	return auth.roles
}

//TenantID returns authorized tenant
func (auth *BaseAuthorization) TenantID() string {
	return auth.tenantID
}

//TenantName returns authorized tenant name
func (auth *BaseAuthorization) TenantName() string {
	return auth.tenantName
}

//AuthToken returns X_AUTH_TOKEN
func (auth *BaseAuthorization) AuthToken() string {
	return auth.authToken
}

//Catalog returns service catalog
func (auth *BaseAuthorization) Catalog() []*Catalog {
	return auth.catalog
}

//Role describes user role
type Role struct {
	Name string
}

//Endpoint represents Endpoint information
type Endpoint struct {
	URL       string
	Region    string
	Interface string
}

//NewEndpoint initializes Endpoint
func NewEndpoint(url, region, iface string) *Endpoint {
	return &Endpoint{URL: url, Region: region, Interface: iface}
}

//Catalog represents service catalog info
type Catalog struct {
	Name      string
	Type      string
	Endpoints []*Endpoint
}

//NewCatalog initializes Catalog
func NewCatalog(name, catalogType string, endPoints []*Endpoint) *Catalog {
	return &Catalog{Name: name, Type: catalogType, Endpoints: endPoints}
}

//Match checks if this role is for this principal
func (r *Role) Match(principal string) bool {
	return r.Name == principal
}

//NewPolicy returns new policy from object
func NewPolicy(raw interface{}) (*Policy, error) {
	typeData := raw.(map[string](interface{}))
	policy := &Policy{}
	policy.ID, _ = typeData["id"].(string)
	policy.Description, _ = typeData["description"].(string)
	policy.Principal, _ = typeData["principal"].(string)
	policy.Action, _ = typeData["action"].(string)
	policy.Effect, _ = typeData["effect"].(string)
	resourceData, _ := typeData["resource"].(map[string]interface{})
	resource := &resourceFilter{}
	policy.resource = resource
	path, _ := resourceData["path"].(string)
	match, err := regexp.Compile(path)
	if err != nil {
		return nil, err
	}
	resource.Path = match

	rawTenantID, _ := typeData["tenant_id"].(string)
	tenantID, err := getRegexp(rawTenantID)
	if err != nil {
		return nil, err
	}
	policy.tenantID = tenantID

	rawTenantName, _ := typeData["tenant_name"].(string)
	tenantName, err := getRegexp(rawTenantName)
	if err != nil {
		return nil, err
	}
	policy.tenantName = tenantName

	if tenantName.String() != globalRegexp && tenantID.String() != globalRegexp {
		return nil, fmt.Errorf(onlyOneOfTenantIDTenantNameError)
	}

	filterFactory := FilterFactory{}
	if resource.PropertiesFilter, err = filterFactory.CreateFilterFromProperties(
		getStringSliceFromMap(resourceData, "properties"),
		getStringSliceFromMap(resourceData, "blacklistProperties"),
	); err != nil {
		return nil, err
	}

	if policy.Action == ActionAttach {
		// source_relation_property is required
		relationProperty, hasSource := typeData["relation_property"].(string)
		if !hasSource {
			return nil, errors.New("\"relation_property\" is required in an attach policy")
		}

		// target_condition is required
		rawTargetCondition, hasTarget := typeData["target_condition"].([]interface{})
		if !hasTarget {
			return nil, errors.New("\"target_condition\" is required in an attach policy")
		}

		policy.currentResourceCondition = &ResourceCondition{}
		policy.relationPropertyName = relationProperty
		policy.otherResourceCondition, err = NewResourceCondition(rawTargetCondition, policy.ID)
		if err != nil {
			return nil, err
		}
	} else {
		rawCondition, _ := typeData["condition"].([]interface{})
		condition, err := NewResourceCondition(rawCondition, policy.ID)
		if err != nil {
			return nil, err
		}
		policy.currentResourceCondition = condition
	}

	return policy, nil
}

func BuildDefaultPolicy(schema, relatedSchema *Schema, property *Property) []*Policy {
	if !hasTenant(relatedSchema) {
		return nil
	}

	if isTenantSchema(relatedSchema) {
		return nil
	}

	if !hasPublicCondition(relatedSchema) {
		policy, err := NewPolicy(
			map[string]interface{}{
				"action":            ActionAttach,
				"effect":            "allow",
				"id":                fmt.Sprintf("default_%s_to_%s_basic_attach_policy", schema.ID, property.ID),
				"principal":         "Member",
				"relation_property": property.ID,
				"target_condition":  []interface{}{conditionIsOwner},
				"resource": map[string]interface{}{
					"path": schema.GetPluralURL(),
				}})

		if err != nil {
			panic(err)
		}
		return []*Policy{policy}
	}

	publicPolicyId := fmt.Sprintf("default_%s_to_%s_public_attach_policy", schema.ID, property.ID)
	condition := []interface{}{
		map[string]interface{}{
			"or": []interface{}{
				conditionIsOwner,
				map[string]interface{}{
					"match": map[string]interface{}{
						"property": "is_public",
						"type":     "eq",
						"value":    true,
					},
				},
			},
		},
	}

	policy, err := NewPolicy(
		map[string]interface{}{
			"action":            ActionAttach,
			"effect":            "allow",
			"id":                publicPolicyId,
			"principal":         "Member",
			"relation_property": property.ID,
			"target_condition":  condition,
			"resource": map[string]interface{}{
				"path": schema.GetPluralURL(),
			}})

	if err != nil {
		panic(err)
	}

	return []*Policy{policy}
}

func isTenantSchema(s *Schema) bool {
	return s.ID == "tenant"
}

func hasTenant(s *Schema) bool {
	for _, property := range s.Properties {
		if property.ID == "tenant_id" {
			return true
		}
	}

	return false
}

func hasPublicCondition(s *Schema) bool {
	for _, property := range s.Properties {
		if property.ID == "is_public" {
			return true
		}
	}

	return false
}

func (p *Policy) GetCurrentResourceCondition() *ResourceCondition {
	return p.currentResourceCondition
}

func getStringSliceFromMap(data map[string]interface{}, key string) []string {
	switch slice := data[key].(type) {
	case []string:
		return slice
	case []interface{}:
		return getStringSliceFromRawSlice(slice)
	default:
		return nil
	}
}

func getStringSliceFromRawSlice(data []interface{}) []string {
	var result []string
	for _, rawItem := range data {
		if item, ok := rawItem.(string); ok {
			result = append(result, item)
		}
	}
	return result
}

func NewResourceCondition(rawCondition []interface{}, policyID string) (*ResourceCondition, error) {
	p := &ResourceCondition{Condition: rawCondition}
	p.actionTenantFilter = map[string][]tenant{}
	p.actionPropertyConditionFilter = map[string][]map[string]interface{}{}
	for _, condition := range p.Condition {
		switch condition.(type) {
		case string:
			switch condition {
			case conditionIsOwner:
				p.requireOwner = true
			default:
				panic(fmt.Sprintf("Unknown condition '%s' for policy '%s'", condition, policyID))
			}
		case map[string]interface{}:
			conditionObject := condition.(map[string]interface{})
			if conditionType, ok := conditionObject["type"]; ok {
				switch conditionType {
				case conditionTypeBelongsTo:
					actions := AllActions
					if action, ok := conditionObject["action"]; ok && action != ActionGlob {
						actions = []string{action.(string)}
					}
					rawTenantID, _ := conditionObject["tenant_id"].(string)
					tenantID, err := getRegexp(rawTenantID)
					if err != nil {
						return nil, err
					}

					rawTenantName, _ := conditionObject["tenant_name"].(string)
					tenantName, err := getRegexp(rawTenantName)
					if err != nil {
						return nil, err
					}

					if tenantName.String() != globalRegexp && tenantID.String() != globalRegexp {
						panic(onlyOneOfTenantIDTenantNameError)
					}

					for _, action := range actions {
						p.addTenantToFilter(action, tenant{ID: tenantID, Name: tenantName})
					}
				case conditionProperty:
					actions := AllActions
					if action, ok := conditionObject["action"]; ok && action != ActionGlob {
						actions = []string{action.(string)}
					}
					match, ok := conditionObject[conditionMatch].(map[string]interface{})
					if !ok {
						panic("match should be dict")
					}
					for _, action := range actions {
						p.addPropertyConditionFilter(action, match)
					}
				default:
					panic(fmt.Sprintf(
						"Unknown condition type '%s' for policy '%s'",
						conditionObject["type"],
						policyID,
					))
				}
			} else if andType, ok := conditionObject[conditionAnd]; ok {
				var err error
				if p.actionFilter, err = precomputeAndCondition(andType, policyID); err != nil {
					return nil, err
				}
			} else if orType, ok := conditionObject[conditionOr]; ok {
				var err error
				if p.actionFilter, err = precomputeOrCondition(orType, policyID); err != nil {
					return nil, err
				}
			}
		default:
			panic(fmt.Sprintf("Invalid condition format for policy '%s'", policyID))
		}
	}

	return p, nil
}

// addTenantToFilter adds tenant to filter for given action
func (p *ResourceCondition) addTenantToFilter(action string, tenant tenant) {
	p.actionTenantFilter[action] = append(p.actionTenantFilter[action], tenant)
}

// addPropertyConditionFilter adds property based filter for action
func (p *ResourceCondition) addPropertyConditionFilter(action string, match map[string]interface{}) {
	p.actionPropertyConditionFilter[action] = append(p.actionPropertyConditionFilter[action], match)
}

//NewEmptyPolicy Return Empty policy which match everything
func NewEmptyPolicy() *Policy {
	return &Policy{resource: &resourceFilter{}, currentResourceCondition: &ResourceCondition{}}
}

func (p *Policy) match(action, path string, auth Authorization) *Role {
	if p.Action == ActionAttach || action == ActionAttach {
		if p.Action != action {
			return nil
		}
	} else if p.Action != "*" && action != p.Action {
		return nil
	}
	if !p.resource.Path.MatchString(path) {
		return nil
	}

	if !p.tenantID.MatchString(auth.TenantID()) {
		return nil
	}

	if !p.tenantName.MatchString(auth.TenantName()) {
		return nil
	}

	roles := auth.Roles()
	for _, role := range roles {
		if role.Match(p.Principal) {
			return role
		}
	}

	return nil
}

func (p *Policy) matchAttach(path string, auth Authorization) bool {
	if p.match(ActionAttach, path, auth) == nil {
		return false
	}

	return true
}

func (p *Policy) IsDeny() bool {
	return strings.ToLower(p.Effect) == "deny"
}

//RequireOwner ...
func (p *ResourceCondition) RequireOwner() bool {
	return p.requireOwner
}

//RemoveHiddenProperty removes hidden data from data by Policy
// This method returns nil if all data get filtered out
func (p *Policy) RemoveHiddenProperty(data map[string]interface{}) map[string]interface{} {
	return p.resource.PropertiesFilter.RemoveHiddenKeysFromMap(data)
}

//FilterSchema filters properties in the schema itself
func (p *Policy) FilterSchema(
	properties map[string]interface{},
	propertiesOrder, required []string,
) (map[string]interface{}, []string, []string) {
	filter := p.resource.PropertiesFilter
	return filter.RemoveHiddenKeysFromMap(properties),
		filter.RemoveHiddenKeysFromSlice(propertiesOrder),
		filter.RemoveHiddenKeysFromSlice(required)
}

//Checks if user is authorized to perform given action
func (p *Policy) Check(action string, authorization Authorization, data map[string]interface{}) error {
	currCond := p.GetCurrentResourceCondition()
	if currCond.RequireOwner() {
		ownerID, _ := data["tenant_id"].(string)
		ownerName, _ := data["tenant_name"].(string)
		owner := newTenant(ownerID, ownerName)
		caller := newTenant(authorization.TenantID(), authorization.TenantName())

		if caller.notEqual(owner) && !currCond.isTenantAllowed(action, owner, caller) {
			return fmt.Errorf("Tenant '%s' is prohibited from operating on resources of tenant '%s'", caller, owner)
		}
	}

	for key := range data {
		if key == "tenant_name" {
			continue
		}
		if p.resource.PropertiesFilter.IsForbidden(key) {
			return fmt.Errorf("%s is prohibited for this user", key)
		}
	}

	return nil
}

// ApplyPropertyConditionFilter applies filter based on Property
// You need to pass candidate update value in updateCandidateData on update API, so
// that we can limit allowed update value.
// Let's say we would like to only allow to update from ACTIVE to ERROR on an API.
// We can define this policy like this.
//
//   - action: 'update'
//     condition:
//     - property:
//         status:
//            ACTIVE: ERROR
//     effect: allow
//     id: member
//     principal: Member
//
// This policy check error in case of followings
// - Original value isn't ACTIVE
// - Update candidate value isn't ERROR
func (p *ResourceCondition) ApplyPropertyConditionFilter(action string, data map[string]interface{}, updateCandidateData map[string]interface{}) error {
	filters, ok := p.actionPropertyConditionFilter[action]
	if !ok {
		return nil
	}
FilterLoop:
	for _, filter := range filters {
		for key, rawAllowedValue := range filter {
			value, _ := data[key]
			switch rawAllowedValue.(type) {
			// A policy should be map in case you need to use previous value & update candidate value
			case map[string]interface{}:
				stringValue, ok := value.(string)
				if !ok {
					return fmt.Errorf("Rejected by property filter")
				}
				allowedValueMap, _ := rawAllowedValue.(map[string]interface{})
				allowedNextValue, ok := allowedValueMap[stringValue]
				if !ok {
					return fmt.Errorf("Rejected by property filter %s %s", allowedValueMap, value)
				}
				if updateCandidateData == nil {
					continue FilterLoop
				} else {
					if !util.Match(allowedNextValue, updateCandidateData[key]) {
						return fmt.Errorf("Rejected by property filter %s %s", allowedNextValue, updateCandidateData[stringValue])
					}
				}
			default:
				if !util.Match(rawAllowedValue, value) {
					return fmt.Errorf("Rejected by property filter %s %s", rawAllowedValue, value)
				}
				continue
			}
		}
	}
	return nil
}

// GetTenantIDFilter returns tenants filter for the action performed by the tenant
func (p *ResourceCondition) GetTenantIDFilter(action string, tenantID string) []string {
	if !p.requireOwner {
		return nil
	}
	result := []string{}
	for _, t := range p.actionTenantFilter[action] {
		result = append(result, t.ID.String())
	}
	return append(result, tenantID)
}

// getTenantFilter returns tenants filter for the action performed by the tenant
func (p *ResourceCondition) getTenantFilter(action string, tenant tenant) []tenant {
	if !p.requireOwner {
		return nil
	}
	return append(p.actionTenantFilter[action], tenant)
}

func (p *ResourceCondition) isTenantAllowed(action string, owner, tenant tenant) bool {
	for _, allowedTenant := range p.getTenantFilter(action, tenant) {
		if owner.equal(allowedTenant) {
			return true
		}
	}
	return false
}

func precomputeAndCondition(andCondition interface{}, policyID string) (*conditionFilter, error) {
	return precomputeCondition(andCondition, andFilter, policyID)
}

func precomputeOrCondition(orCondition interface{}, policyID string) (*conditionFilter, error) {
	return precomputeCondition(orCondition, orFilter, policyID)
}

func precomputeCondition(
	conds interface{},
	filterType conditionFilterType,
	policyID string,
) (*conditionFilter, error) {
	conditions := conds.([]interface{})
	actionFilter := makeConditionFilter(filterType)
	for _, condition := range conditions {
		switch condition.(type) {
		case string:
			stringValue := condition.(string)
			if stringValue != conditionIsOwner {
				return nil, fmt.Errorf("Unknown condition '%s' for policy '%s'", condition, policyID)
			}
			actionFilter.isOwner = true
		case map[string]interface{}:
			conditionObject := condition.(map[string]interface{})
			if _, ok := conditionObject[conditionOr]; ok {
				if orFilter, err := precomputeOrCondition(conditionObject[conditionOr], policyID); err != nil {
					return nil, err
				} else {
					actionFilter.orFilters = orFilter
				}
			} else if _, ok := conditionObject[conditionAnd]; ok {
				if andFilter, err := precomputeAndCondition(conditionObject[conditionAnd], policyID); err != nil {
					return nil, err
				} else {
					actionFilter.andFilters = andFilter
				}
			} else if _, ok := conditionObject[conditionMatch]; ok {
				actionFilter.matches = append(actionFilter.matches, conditionObject[conditionMatch].(map[string]interface{}))
			} else {
				return nil, fmt.Errorf("Unknown condition '%s' for policy '%s'", condition, policyID)
			}
		default:
			return nil, fmt.Errorf("Unknown condition '%s' for policy '%s'", condition, policyID)
		}
	}
	return actionFilter, nil
}

// Adds custom filters based on this policy to the `filters` map
func (policy *ResourceCondition) AddCustomFilters(filters map[string]interface{}, tenantId string) {
	addCustomFilters(filters, tenantId, policy.actionFilter)
}

func (policy *Policy) GetResourcePathRegexp() *regexp.Regexp {
	return policy.resource.Path
}

func (policy *Policy) GetPropertyFilter() *Filter {
	return policy.resource.PropertiesFilter
}

func (policy *Policy) GetRelationPropertyName() string {
	return policy.relationPropertyName
}

func (policy *Policy) GetOtherResourceCondition() *ResourceCondition {
	return policy.otherResourceCondition
}

func addCustomFilters(f map[string]interface{}, tenantId string, conditionFilters *conditionFilter) {
	if conditionFilters == nil {
		return
	}
	filters := make([]map[string]interface{}, 0)
	if conditionFilters.isOwner {
		filters = append(filters, map[string]interface{}{"property": "tenant_id", "type": "eq", "value": tenantId})
	}
	for _, match := range conditionFilters.matches {
		filters = append(filters, match)
	}
	if conditionFilters.andFilters != nil {
		andFilter := map[string]interface{}{}
		addCustomFilters(andFilter, tenantId, conditionFilters.andFilters)
		filters = append(filters, andFilter)
	}
	if conditionFilters.orFilters != nil {
		orFilter := map[string]interface{}{}
		addCustomFilters(orFilter, tenantId, conditionFilters.orFilters)
		filters = append(filters, orFilter)
	}
	if conditionFilters.filterType == orFilter {
		f["__or__"] = filters
	} else {
		f["__and__"] = filters
	}
}

//PolicyValidate validates api request using policy validation
func PolicyValidate(action, path string, auth Authorization, policies []*Policy) (foundPolicy *Policy, foundRole *Role) {
	for _, policy := range policies {
		if role := policy.match(action, path, auth); role != nil {
			if policy.IsDeny() {
				return nil, nil
			} else if foundPolicy == nil {
				foundPolicy = policy
				foundRole = role
			}
		}
	}
	return
}

func GetAttachmentPolicies(path string, auth Authorization, policies []*Policy) []*Policy {
	attachmentPolicies := []*Policy{}
	for _, policy := range policies {
		if policy.matchAttach(path, auth) {
			attachmentPolicies = append(attachmentPolicies, policy)
		}
	}
	return attachmentPolicies
}

func getRegexp(input string) (*regexp.Regexp, error) {
	if input == "" {
		input = globalRegexp
	}
	return regexp.Compile(input)
}
