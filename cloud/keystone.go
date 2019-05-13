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

package cloud

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	v3tenants "github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	v3tokens "github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/pagination"
)

//KeystoneIdentity middleware
type KeystoneIdentity interface {
	GetTenantID(string) (string, error)
	GetTenantName(string) (string, error)
	VerifyToken(string) (schema.Authorization, error)
	GetServiceAuthorization() (schema.Authorization, error)
	GetServiceTokenID() string
	ValidateTenantID(string) (bool, error)
}

type keystoneV3Client struct {
	client *gophercloud.ServiceClient
}

func matchVersionFromAuthURL(authURL string) (version string) {
	re := regexp.MustCompile(`(?P<version>v[\d\.]+)/?$`)
	match := re.FindStringSubmatch(authURL)
	for i, name := range re.SubexpNames() {
		if name == "version" && i < len(match) {
			version = match[i]
			break
		}
	}
	return
}

//NewKeystoneIdentity is an constructor for KeystoneIdentity middleware
func NewKeystoneIdentity(authURL, userName, password, domainName, tenantName string) (KeystoneIdentity, error) {
	if version := matchVersionFromAuthURL(authURL); version != "v3" {
		return nil, fmt.Errorf("Unsupported keystone version: %s", version)
	}

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: authURL,
		Username:         userName,
		Password:         password,
		DomainName:       domainName,
		TenantName:       tenantName,
		AllowReauth:      true,
	}

	client, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}
	identityClient, err := openstack.NewIdentityV3(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	return &keystoneV3Client{client: identityClient}, nil
}

//VerifyToken verifies keystone v3.0 token
func (client *keystoneV3Client) VerifyToken(token string) (schema.Authorization, error) {
	defer client.measureTime(time.Now(), "verify_token")

	tokenResult := v3tokens.Get(client.client, token)
	if tokenResult.Err != nil {
		return nil, fmt.Errorf("Error during verifying token: %s", tokenResult.Err.Error())
	}
	_, err := tokenResult.ExtractToken()
	_, ok := tokenResult.Body.(map[string]interface{})
	// tricky gophercloud behavior.
	// If system token doesn't need reauth, err is set when token is invalid
	// If system token needed reauth and user token is invalid, err wont be propagated neither by return nor Err field,
	// but response is nil
	if err != nil || !ok {
		client.updateCounter(1, "error.token.invalid")
		return nil, fmt.Errorf("Invalid token")
	}

	// Get roles
	roles, err := tokenResult.ExtractRoles()
	if err != nil {
		client.updateCounter(1, "error.token.extract_roles")
		return nil, err
	}
	roleIDs := []string{}
	for _, r := range roles {
		roleIDs = append(roleIDs, r.Name)
	}

	// Get project/tenant
	project, err := tokenResult.ExtractProject()
	if err != nil {
		client.updateCounter(1, "error.token.extract_project")
		return nil, err
	}
	if project != nil {
		tenant := schema.Tenant{
			ID:   project.ID,
			Name: project.Name,
		}
		domain := schema.Domain{
			ID:   project.Domain.ID,
			Name: project.Domain.Name,
		}
		builder := schema.NewAuthorizationBuilder().
			WithTenant(tenant).
			WithDomain(domain).
			WithRoleIDs(roleIDs...)

		if isTokenScopedToAdminProject(&tokenResult) {
			return builder.BuildAdmin(), nil
		}
		return builder.BuildScopedToTenant(), nil
	} else {
		dom, err := extractDomain(&tokenResult)
		if err != nil {
			client.updateCounter(1, "error.token.extract_domain")
			return nil, err
		}
		if dom == nil {
			client.updateCounter(1, "error.token.unscoped")
			return nil, errors.New("Token is unscoped")
		}
		domain := schema.Domain{
			ID:   dom.ID,
			Name: dom.Name,
		}
		auth := schema.NewAuthorizationBuilder().
			WithDomain(domain).
			WithRoleIDs(roleIDs...).
			BuildScopedToDomain()
		return auth, nil
	}
}

// GetTenantID maps the given v3.0 project ID to the projects's name
func (client *keystoneV3Client) GetTenantID(tenantName string) (string, error) {
	defer client.measureTime(time.Now(), "get_tenant.id")

	tenant, err := client.getTenant(func(tenant *v3tenants.Project) bool { return tenant.Name == tenantName })
	if err != nil {
		return "", err
	}

	if tenant == nil {
		return "", fmt.Errorf("Tenant with name '%s' not found", tenantName)
	}

	return tenant.ID, nil
}

// GetTenantName maps the given v3.0 project name to the projects's ID
func (client *keystoneV3Client) GetTenantName(tenantID string) (string, error) {
	defer client.measureTime(time.Now(), "get_tenant.name")

	tenant, err := client.getTenant(func(tenant *v3tenants.Project) bool { return tenant.ID == tenantID })
	if err != nil {
		return "", err
	}

	if tenant == nil {
		return "", fmt.Errorf("Tenant with ID '%s' not found", tenantID)
	}

	return tenant.Name, nil
}

// GetServiceAuthorization returns the master authorization with full permissions
func (client *keystoneV3Client) GetServiceAuthorization() (schema.Authorization, error) {
	defer client.measureTime(time.Now(), "get_service_authorization")

	return client.VerifyToken(client.client.TokenID)
}

func (client *keystoneV3Client) GetServiceTokenID() string {
	return client.client.TokenID
}

func extractDomain(result *v3tokens.GetResult) (*v3tokens.Domain, error) {
	var s struct {
		Domain *v3tokens.Domain `json:"domain"`
	}
	err := result.ExtractInto(&s)
	return s.Domain, err
}

func isTokenScopedToAdminProject(result *v3tokens.GetResult) bool {
	var s struct {
		IsAdminProject bool `json:"is_admin_project"`
	}
	err := result.ExtractInto(&s)
	return (err == nil) && s.IsAdminProject
}

func (client *keystoneV3Client) getTenant(filter func(*v3tenants.Project) bool) (*v3tenants.Project, error) {
	opts := v3tenants.ListOpts{}
	pager := v3tenants.List(client.client, opts)
	var result *v3tenants.Project
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		tenantsList, err := v3tenants.ExtractProjects(page)
		if err != nil {
			return false, err
		}

		for _, tenant := range tenantsList {
			if filter(&tenant) {
				result = &tenant
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (client *keystoneV3Client) updateCounter(delta int64, action string) {
	metrics.UpdateCounter(delta, "auth.v3.%s", action)
}

func (client *keystoneV3Client) measureTime(timeStarted time.Time, action string) {
	metrics.UpdateTimer(timeStarted, "auth.v3.%s", action)
}

func (client *keystoneV3Client) ValidateTenantID(id string) (bool, error) {
	defer client.measureTime(time.Now(), "validate_tenant_id")

	tenant, err := client.getTenant(func(tenant *v3tenants.Project) bool { return tenant.ID == id })
	if err != nil {
		return false, err
	}

	return tenant != nil, nil
}
