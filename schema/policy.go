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

	"github.com/cloudwan/gohan/extension/goext/filter"
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

	adminRole = "admin"
)

// AllActions are all possible actions
var AllActions = []string{ActionCreate, ActionRead, ActionUpdate, ActionDelete}

var adminPolicy = func() *Policy {
	policy, err := NewPolicy(map[string]interface{}{
		"action": "*",
		"effect": "allow",
		"id": "admin_statement",
		"principal": "admin",
		"resource": map[string]interface{}{
			"path": ".*",
		},
	})
	if err != nil {
		panic(errors.Errorf("Failed to create the admin policy: %v", err))
	}
	return policy
}()

func newTenantMatcher(tenantID, tenantName string) tenantMatcher {
	tenantIDRegexp, _ := getRegexp(tenantID)
	tenantNameRegexp, _ := getRegexp(tenantName)
	return tenantMatcher{ID: tenantIDRegexp, Name: tenantNameRegexp}
}

// tenantMatcher matches given tenant
type tenantMatcher struct {
	ID   *regexp.Regexp
	Name *regexp.Regexp
}

func (t *tenantMatcher) equal(t2 tenantMatcher) bool {
	idMatch := t.ID.MatchString(t2.ID.String()) || t2.ID.MatchString(t.ID.String())
	nameMatch := t.Name.MatchString(t2.Name.String()) || t2.Name.MatchString(t.Name.String())
	return idMatch && nameMatch
}

func (t *tenantMatcher) notEqual(t2 tenantMatcher) bool {
	return !t.equal(t2)
}

func (t tenantMatcher) String() string {
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
	RawData                                    interface{}
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
	actionTenantFilter            map[string][]tenantMatcher
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
	DomainID() string
	DomainName() string
	Roles() []*Role
	IsAdmin() bool
	getResourceFilters(schema *Schema) []map[string]interface{}
	checkAccessToResource(cond *ResourceCondition, action string, resource map[string]interface{}) error
	getTenantAndDomainFilters(cond *ResourceCondition, action string) (tenantFilter []string, domainFilter []string)
}

type TenantScopedAuthorization struct {
	DomainScopedAuthorization
	tenant Tenant
}

type DomainScopedAuthorization struct {
	domain Domain
	roles  []*Role
}

// AdminScopedAuthorization represents authorization for the admin,
// i.e. an authorization scoped to an admin project
type AdminAuthorization struct {
	TenantScopedAuthorization
}

type AuthorizationBuilder struct {
	authViaKeystoneV2 bool
	tenant            Tenant
	domain            Domain
	roles             []*Role
}

func NewAuthorizationBuilder() *AuthorizationBuilder {
	return &AuthorizationBuilder{
		domain: DefaultDomain,
		roles:  []*Role{},
	}
}

func (ab *AuthorizationBuilder) WithKeystoneV2Compatibility() *AuthorizationBuilder {
	ab.authViaKeystoneV2 = true
	return ab
}

func (ab *AuthorizationBuilder) WithTenant(tenant Tenant) *AuthorizationBuilder {
	ab.tenant = tenant
	return ab
}

func (ab *AuthorizationBuilder) WithDomain(domain Domain) *AuthorizationBuilder {
	ab.domain = domain
	return ab
}

func (ab *AuthorizationBuilder) WithRoleIDs(roleIDs ...string) *AuthorizationBuilder {
	roles := []*Role{}
	for _, id := range roleIDs {
		roles = append(roles, &Role{Name: id})
	}
	ab.roles = roles
	return ab
}

func (ab *AuthorizationBuilder) BuildScopedToTenant() Authorization {
	if ab.authViaKeystoneV2 {
		// When using Keystone V2, user is an admin if they have an admin role in the current project
		for _, role := range ab.roles {
			if role.Name == adminRole {
				return ab.BuildAdmin()
			}
		}
	}

	return &TenantScopedAuthorization{
		tenant: ab.tenant,
		DomainScopedAuthorization: DomainScopedAuthorization{
			domain: ab.domain,
			roles:  ab.roles,
		},
	}
}

func (ab *AuthorizationBuilder) BuildScopedToDomain() Authorization {
	return &DomainScopedAuthorization{
		domain: ab.domain,
		roles:  ab.roles,
	}
}

func (ab *AuthorizationBuilder) BuildAdmin() Authorization {
	return &AdminAuthorization{
		TenantScopedAuthorization: TenantScopedAuthorization{
			tenant: ab.tenant,
			DomainScopedAuthorization: DomainScopedAuthorization{
				domain: ab.domain,
				roles:  ab.roles,
			},
		},
	}
}

// TenantScopedAuthorization

func (auth *TenantScopedAuthorization) TenantID() string {
	return auth.tenant.ID
}

func (auth *TenantScopedAuthorization) TenantName() string {
	return auth.tenant.Name
}

func (auth *TenantScopedAuthorization) getResourceFilters(schema *Schema) []map[string]interface{} {
	tenantFilter := getFilterByPropertyIfPresent(schema, "tenant_id", auth.TenantID())
	domainFilter := getFilterByPropertyIfPresent(schema, "domain_id", auth.DomainID())
	return makeAndFilters(append(tenantFilter, domainFilter...))
}

func (auth *TenantScopedAuthorization) checkAccessToResource(cond *ResourceCondition, action string, resource map[string]interface{}) error {
	if err := checkTenantAccess(cond, action, auth.tenant, resource); err != nil {
		return err
	}

	return checkDomainAccess(auth.domain, resource)
}

func (auth *TenantScopedAuthorization) getTenantAndDomainFilters(cond *ResourceCondition, action string) (tenantFilter []string, domainFilter []string) {
	for _, t := range cond.actionTenantFilter[action] {
		tenantFilter = append(tenantFilter, t.ID.String())
	}
	tenantFilter = append(tenantFilter, auth.TenantID())
	domainFilter = []string{auth.DomainID()}
	return
}

// DomainScopedAuthorization

func (auth *DomainScopedAuthorization) TenantID() string {
	return ""
}

func (auth *DomainScopedAuthorization) TenantName() string {
	return ""
}

func (auth *DomainScopedAuthorization) DomainID() string {
	return auth.domain.ID
}

func (auth *DomainScopedAuthorization) DomainName() string {
	return auth.domain.Name
}

func (auth *DomainScopedAuthorization) Roles() []*Role {
	return auth.roles
}

func (auth *DomainScopedAuthorization) IsAdmin() bool {
	return false
}

func (auth *DomainScopedAuthorization) getResourceFilters(schema *Schema) []map[string]interface{} {
	return getFilterByPropertyIfPresent(schema, "domain_id", auth.DomainID())
}

func (auth *DomainScopedAuthorization) checkAccessToResource(cond *ResourceCondition, action string, resource map[string]interface{}) error {
	return checkDomainAccess(auth.domain, resource)
}

func (auth *DomainScopedAuthorization) getTenantAndDomainFilters(cond *ResourceCondition, action string) (tenantFilter []string, domainFilter []string) {
	domainFilter = []string{auth.DomainID()}
	return
}

// AdminScopedAuthorization

func (auth *AdminAuthorization) IsAdmin() bool {
	return true
}

func (auth *AdminAuthorization) getResourceFilters(schema *Schema) []map[string]interface{} {
	return []map[string]interface{}{}
}

func (auth *AdminAuthorization) checkAccessToResource(cond *ResourceCondition, action string, resource map[string]interface{}) error {
	return nil
}

func (auth *AdminAuthorization) getTenantAndDomainFilters(cond *ResourceCondition, action string) (tenantFilter []string, domainFilter []string) {
	return
}

func getFilterByPropertyIfPresent(schema *Schema, propertyName string, propertyValue interface{}) []map[string]interface{} {
	if _, err := schema.GetPropertyByID(propertyName); err == nil {
		return []map[string]interface{}{
			filter.Eq(propertyName, propertyValue),
		}
	}
	return nil
}

func checkTenantAccess(cond *ResourceCondition, action string, tenant Tenant, resource map[string]interface{}) error {
	ownerID, _ := resource["tenant_id"].(string)
	ownerName, _ := resource["tenant_name"].(string)
	owner := newTenantMatcher(ownerID, ownerName)
	caller := newTenantMatcher(tenant.ID, tenant.Name)

	if caller.notEqual(owner) && !cond.isTenantAllowed(action, owner, caller) {
		return errors.New("Operating on resources from other tenant is prohibited")
	}

	return nil
}

func checkDomainAccess(domain Domain, resource map[string]interface{}) error {
	resourceDomainID, setsDomain := resource["domain_id"].(string)
	if setsDomain && domain.ID != resourceDomainID {
		return errors.New("Operating on resources from other domain is prohibited")
	}

	return nil
}

type Tenant struct {
	ID   string
	Name string
}

type Domain struct {
	ID   string
	Name string
}

var DefaultDomain = Domain{
	ID:   "default",
	Name: "Default",
}

//Role describes user role
type Role struct {
	Name string
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
	policy.RawData = raw
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
	p.actionTenantFilter = map[string][]tenantMatcher{}
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
						p.addTenantToFilter(action, tenantMatcher{ID: tenantID, Name: tenantName})
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

// addTenantToFilter adds tenantMatcher to filter for given action
func (p *ResourceCondition) addTenantToFilter(action string, tenant tenantMatcher) {
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
		if err := authorization.checkAccessToResource(currCond, action, data); err != nil {
			return err
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

func (p *ResourceCondition) GetTenantAndDomainFilters(action string, auth Authorization) (tenantFilter []string, domainFilter []string) {
	if !p.requireOwner {
		return nil, nil
	}
	return auth.getTenantAndDomainFilters(p, action)
}

// getTenantFilter returns tenants filter for the action performed by the tenantMatcher
func (p *ResourceCondition) getTenantFilter(action string, tenant tenantMatcher) []tenantMatcher {
	if !p.requireOwner {
		return nil
	}
	return append(p.actionTenantFilter[action], tenant)
}

func (p *ResourceCondition) isTenantAllowed(action string, owner, tenant tenantMatcher) bool {
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
func (policy *ResourceCondition) AddCustomFilters(schema *Schema, filters map[string]interface{}, auth Authorization) {
	addCustomFilters(schema, filters, auth, policy.actionFilter)
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

func addCustomFilters(schema *Schema, f map[string]interface{}, auth Authorization, conditionFilters *conditionFilter) {
	if conditionFilters == nil {
		return
	}
	filters := make([]map[string]interface{}, 0)
	if conditionFilters.isOwner {
		filters = append(filters, auth.getResourceFilters(schema)...)
	}
	for _, match := range conditionFilters.matches {
		filters = append(filters, match)
	}
	if conditionFilters.andFilters != nil {
		andFilter := map[string]interface{}{}
		addCustomFilters(schema, andFilter, auth, conditionFilters.andFilters)
		filters = append(filters, andFilter)
	}
	if conditionFilters.orFilters != nil {
		orFilter := map[string]interface{}{}
		addCustomFilters(schema, orFilter, auth, conditionFilters.orFilters)
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
	if auth.IsAdmin() {
		policies = append([]*Policy{adminPolicy}, policies...)
	}
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

func makeAndFilters(filters []map[string]interface{}) []map[string]interface{} {
	if len(filters) <= 1 {
		return filters
	}
	return []map[string]interface{}{
		filter.And(filters...),
	}
}
