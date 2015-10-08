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
)

const (
	conditionIsOwner       = "is_owner"
	conditionTypeBelongsTo = "belongs_to"
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

	globalRegexp = ".*"

	onlyOneOfTenantIDTenantNameError = "Only one of [tenant_id, tenant_name] should be specified"
)

var allActions = []string{ActionCreate, ActionRead, ActionUpdate, ActionDelete}

func newTenant(tenantID, tenantName string) Tenant {
	tenantIDRegexp, _ := getRegexp(tenantID)
	tenantNameRegexp, _ := getRegexp(tenantName)
	return Tenant{ID: tenantIDRegexp, Name: tenantNameRegexp}
}

// Tenant ...
type Tenant struct {
	ID   *regexp.Regexp
	Name *regexp.Regexp
}

func (t *Tenant) equal(t2 Tenant) bool {
	idMatch := t.ID.MatchString(t2.ID.String()) || t2.ID.MatchString(t.ID.String())
	nameMatch := t.Name.MatchString(t2.Name.String()) || t2.Name.MatchString(t.Name.String())
	return idMatch && nameMatch
}

func (t *Tenant) notEqual(t2 Tenant) bool {
	return !t.equal(t2)
}

func (t Tenant) String() string {
	return fmt.Sprintf("%s (%s)", t.Name.String(), t.ID.String())
}

//Policy describes policy configuraion for APIs
type Policy struct {
	ID, Description, Principal, Action, Effect string
	Condition                                  []interface{}
	Resource                                   *ResourcePolicy
	RawData                                    interface{}
	TenantID                                   *regexp.Regexp
	TenantName                                 *regexp.Regexp
	requireOwner                               bool
	actionTenantFilter                         map[string][]Tenant
}

//ResourcePolicy describes targe resources
type ResourcePolicy struct {
	Properties []interface{}
	Path       *regexp.Regexp
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
	policy.Condition, _ = typeData["condition"].([]interface{})
	policy.RawData = raw
	resourceData, _ := typeData["resource"].(map[string]interface{})
	resource := &ResourcePolicy{}
	policy.Resource = resource
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
	policy.TenantID = tenantID

	rawTenantName, _ := typeData["tenant_name"].(string)
	tenantName, err := getRegexp(rawTenantName)
	if err != nil {
		return nil, err
	}
	policy.TenantName = tenantName

	if tenantName.String() != globalRegexp && tenantID.String() != globalRegexp {
		return nil, fmt.Errorf(onlyOneOfTenantIDTenantNameError)
	}

	properties, ok := resourceData["properties"]
	resource.Properties = nil
	if ok {
		resource.Properties = properties.([]interface{})
	}
	if err := policy.precomputeConditions(); err != nil {
		return nil, err
	}
	return policy, nil
}

func (p *Policy) precomputeConditions() error {
	p.actionTenantFilter = map[string][]Tenant{}
	for _, condition := range p.Condition {
		switch condition.(type) {
		case string:
			switch condition {
			case conditionIsOwner:
				p.requireOwner = true
			default:
				return fmt.Errorf("Unknown condition '%s' for policy '%s'", condition, p.ID)
			}
		case map[string]interface{}:
			conditionObject := condition.(map[string]interface{})
			switch conditionObject["type"] {
			case conditionTypeBelongsTo:
				actions := allActions
				if action, ok := conditionObject["action"]; ok && action != ActionGlob {
					actions = []string{action.(string)}
				}
				rawTenantID, _ := conditionObject["tenant_id"].(string)
				tenantID, err := getRegexp(rawTenantID)
				if err != nil {
					return err
				}

				rawTenantName, _ := conditionObject["tenant_name"].(string)
				tenantName, err := getRegexp(rawTenantName)
				if err != nil {
					return err
				}

				if tenantName.String() != globalRegexp && tenantID.String() != globalRegexp {
					return fmt.Errorf(onlyOneOfTenantIDTenantNameError)
				}

				for _, action := range actions {
					p.AddTenantToFilter(action, Tenant{ID: tenantID, Name: tenantName})
				}
			default:
				return fmt.Errorf("Unknown condition type '%s' for policy '%s'", conditionObject["type"], p.ID)
			}
		default:
			return fmt.Errorf("Invalid condition format for policy '%s'", p.ID)
		}
	}

	return nil
}

//NewEmptyPolicy Return Empty policy which match everything
func NewEmptyPolicy() *Policy {
	return &Policy{Resource: &ResourcePolicy{}}
}

func (p *Policy) match(action, path string, auth Authorization) *Role {
	if p.Action != "*" && action != p.Action {
		return nil
	}
	if !p.Resource.Path.MatchString(path) {
		return nil
	}

	if !p.TenantID.MatchString(auth.TenantID()) {
		return nil
	}

	if !p.TenantName.MatchString(auth.TenantName()) {
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

func (p *Policy) isAllow() bool {
	return p.Effect == "Allow"
}

//RequireOwner ...
func (p *Policy) RequireOwner() bool {
	return p.requireOwner
}

//Filter ...
func (p *Policy) Filter(data map[string]interface{}) map[string]interface{} {
	properties := p.Resource.Properties
	if properties == nil {
		return data
	}
	result := map[string]interface{}{}
	for _, property := range properties {
		propertyString := property.(string)
		value, ok := data[propertyString]
		if ok {
			result[propertyString] = value
		}
	}
	return result
}

//MetaFilter filters properties in the schema itself
func (p *Policy) MetaFilter(properties map[string]interface{},
	propertiesOrder, required []interface{}) (map[string]interface{}, []interface{}, []interface{}) {
	allowedProperties := p.Resource.Properties
	if allowedProperties == nil {
		return properties, propertiesOrder, required
	}
	filteredProperties := map[string]interface{}{}
	filteredPropertiesOrder := []interface{}{}
	filteredRequired := []interface{}{}
	if propertiesOrder == nil {
		filteredPropertiesOrder = nil
	}
	if required == nil {
		filteredRequired = nil
	}
	for _, propertyRaw := range allowedProperties {
		property := propertyRaw.(string)
		if _, ok := properties[property]; !ok {
			continue
		}
		filteredProperties[property] = properties[property]
	}
	for _, propertyRaw := range propertiesOrder {
		property := propertyRaw.(string)
		if _, ok := filteredProperties[property]; ok {
			filteredPropertiesOrder = append(filteredPropertiesOrder, property)
		}
	}
	for _, propertyRaw := range required {
		property := propertyRaw.(string)
		if _, ok := filteredProperties[property]; ok {
			filteredRequired = append(filteredRequired, property)
		}
	}
	return filteredProperties, filteredPropertiesOrder, filteredRequired
}

func contains(list []interface{}, key string) bool {
	for _, value := range list {
		if key == value.(string) {
			return true
		}
	}
	return false
}

//Check ...
func (p *Policy) Check(action string, authorization Authorization, data map[string]interface{}) error {
	if p.RequireOwner() {
		ownerID, _ := data["tenant_id"].(string)
		ownerName, _ := data["tenant_name"].(string)
		owner := newTenant(ownerID, ownerName)
		caller := newTenant(authorization.TenantID(), authorization.TenantName())

		if caller.notEqual(owner) && !p.isTenantAllowed(action, owner, caller) {
			return fmt.Errorf("Tenant '%s' is prohibited from operating on resources of tenant '%s'", caller, owner)
		}
	}

	properties := p.Resource.Properties
	if properties == nil {
		log.Debug("No properties in resource policy. Allowing all property access")
		return nil
	}
	for key := range data {
		if key == "tenant_name" {
			continue
		}
		if !contains(properties, key) {
			return fmt.Errorf("%s is prohibited for this user", key)
		}
	}
	return nil
}

// AddTenantToFilter adds tenant to filter for given action
func (p *Policy) AddTenantToFilter(action string, tenant Tenant) {
	p.actionTenantFilter[action] = append(p.actionTenantFilter[action], tenant)
}

// GetTenantIDFilter returns tenants filter for the action performed by the tenant
func (p *Policy) GetTenantIDFilter(action string, tenantID string) []string {
	if !p.requireOwner {
		return nil
	}
	result := []string{}
	for _, t := range p.actionTenantFilter[action] {
		result = append(result, t.ID.String())
	}
	return append(result, tenantID)
}

// GetTenantFilter returns tenants filter for the action performed by the tenant
func (p *Policy) GetTenantFilter(action string, tenant Tenant) []Tenant {
	if !p.requireOwner {
		return nil
	}
	return append(p.actionTenantFilter[action], tenant)
}

func (p *Policy) isTenantAllowed(action string, owner, tenant Tenant) bool {
	for _, allowedTenant := range p.GetTenantFilter(action, tenant) {
		if owner.equal(allowedTenant) {
			return true
		}
	}
	return false
}

//PolicyValidate validates api request using policy validation
func PolicyValidate(action, path string, auth Authorization, policies []*Policy) (*Policy, *Role) {
	for _, policy := range policies {
		if role := policy.match(action, path, auth); role != nil {
			return policy, role
		}
	}
	return nil, nil
}

func getRegexp(input string) (*regexp.Regexp, error) {
	if input == "" {
		input = globalRegexp
	}
	return regexp.Compile(input)
}
