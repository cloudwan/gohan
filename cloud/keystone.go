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
	"fmt"
	"net/http"
	"regexp"

	"github.com/cloudwan/gohan/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	v2tenants "github.com/rackspace/gophercloud/openstack/identity/v2/tenants"
	v3tenants "github.com/rackspace/gophercloud/openstack/identity/v3/projects"
	v3tokens "github.com/rackspace/gophercloud/openstack/identity/v3/tokens"
	"github.com/rackspace/gophercloud/pagination"
)

const maxReauthAttempts = 3

//KeystoneIdentity middleware
type KeystoneIdentity struct {
	Client KeystoneClient
}

// VerifyToken verifies identity
func (identity *KeystoneIdentity) VerifyToken(token string) (schema.Authorization, error) {
	return identity.Client.VerifyToken(token)
}

// GetTenantID maps the given tenant/project name to the tenant's/project's ID
func (identity *KeystoneIdentity) GetTenantID(tenantName string) (string, error) {
	return identity.Client.GetTenantID(tenantName)
}

// GetTenantName maps the given tenant/project name to the tenant's/project's ID
func (identity *KeystoneIdentity) GetTenantName(tenantID string) (string, error) {
	return identity.Client.GetTenantName(tenantID)
}

// GetServiceAuthorization returns the master authorization with full permisions
func (identity *KeystoneIdentity) GetServiceAuthorization() (schema.Authorization, error) {
	return identity.Client.GetServiceAuthorization()
}

// GetClient returns openstack client
func (identity *KeystoneIdentity) GetClient() *gophercloud.ServiceClient {
	return identity.Client.GetClient()
}

//KeystoneClient represents keystone client
type KeystoneClient interface {
	GetTenantID(string) (string, error)
	GetTenantName(string) (string, error)
	VerifyToken(string) (schema.Authorization, error)
	GetServiceAuthorization() (schema.Authorization, error)
	GetClient() *gophercloud.ServiceClient
}

type keystoneV2Client struct {
	client *gophercloud.ServiceClient
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
func NewKeystoneIdentity(authURL, userName, password, domainName, tenantName, version string) (*KeystoneIdentity, error) {
	var client KeystoneClient
	var err error
	if version == "" {
		version = matchVersionFromAuthURL(authURL)
	}
	if version == "v2.0" {
		client, err = NewKeystoneV2Client(authURL, userName, password, tenantName)
	} else if version == "v3" {
		client, err = NewKeystoneV3Client(authURL, userName, password, domainName, tenantName)
	} else {
		return nil, fmt.Errorf("Unsupported keystone version: %s", version)
	}
	if err != nil {
		return nil, err
	}
	return &KeystoneIdentity{
		Client: client,
	}, nil
}

//RoundTripper limits number of Reauth attempts
type RoundTripper struct {
	rt                http.RoundTripper
	numReauthAttempts int
	maxReauthAttempts int
}

//NewHTTPClient returns http client with max reauth retry support
func NewHTTPClient() http.Client {
	return http.Client{
		Transport: &RoundTripper{
			rt:                http.DefaultTransport,
			maxReauthAttempts: maxReauthAttempts,
		},
	}
}

//RoundTrip limits number of Reauth attempts
func (lrt *RoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	var err error

	response, err := lrt.rt.RoundTrip(request)
	if response == nil {
		return nil, err
	}

	if response.StatusCode == http.StatusUnauthorized {
		if lrt.numReauthAttempts == lrt.maxReauthAttempts {
			return response, fmt.Errorf("Failed to reauthenticate to keystone with %d attempts", lrt.maxReauthAttempts)
		}
		lrt.numReauthAttempts++
	}
	return response, err
}

//NewKeystoneV2Client is an constructor for KeystoneV2Client
func NewKeystoneV2Client(authURL, userName, password, tenantName string) (KeystoneClient, error) {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: authURL,
		Username:         userName,
		Password:         password,
		TenantName:       tenantName,
		AllowReauth:      true,
	}

	client, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}
	return &keystoneV2Client{client: openstack.NewIdentityV2(client)}, nil
}

//NewKeystoneV3Client is an constructor for KeystoneV3Client
func NewKeystoneV3Client(authURL, userName, password, domainName, tenantName string) (KeystoneClient, error) {
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
	client.HTTPClient = NewHTTPClient()
	return &keystoneV3Client{client: openstack.NewIdentityV3(client)}, nil
}

//VerifyToken verifies keystone v3.0 token
func (client *keystoneV3Client) VerifyToken(token string) (schema.Authorization, error) {
	tokenResult := v3tokens.Get(client.client, token)
	if tokenResult.Err != nil {
		return nil, fmt.Errorf("Error during verifying token: %s", tokenResult.Err.Error())
	}
	_, err := tokenResult.Extract()
	if err != nil {
		return nil, fmt.Errorf("Invalid token")
	}
	tokenBody := tokenResult.Body.(map[string]interface{})["token"]
	roles := tokenBody.(map[string]interface{})["roles"]
	roleIDs := []string{}
	for _, roleBody := range roles.([]interface{}) {
		roleIDs = append(roleIDs, roleBody.(map[string]interface{})["name"].(string))
	}
	tokenBodyMap := tokenBody.(map[string]interface{})
	project := tokenBodyMap["project"].(map[string]interface{})
	tenantID := project["id"].(string)
	tenantName := project["name"].(string)
	catalogList, ok := tokenBodyMap["catalog"].([]interface{})
	catalogObj := []*schema.Catalog{}
	if ok {
		for _, rawCatalog := range catalogList {
			catalog := rawCatalog.(map[string]interface{})
			endPoints := []*schema.Endpoint{}
			rawEndpoints, ok := catalog["endpoints"].([]interface{})
			if ok {
				for _, rawEndpoint := range rawEndpoints {
					endpoint := rawEndpoint.(map[string]interface{})
					endPoints = append(endPoints,
						schema.NewEndpoint(endpoint["url"].(string), endpoint["region"].(string), endpoint["interface"].(string)))
				}
			}
			catalogObj = append(catalogObj, schema.NewCatalog(catalog["name"].(string), catalog["type"].(string), endPoints))
		}
	}
	return schema.NewAuthorization(tenantID, tenantName, token, roleIDs, catalogObj), nil
}

// GetTenantID maps the given v3.0 project ID to the projects's name
func (client *keystoneV3Client) GetTenantID(tenantName string) (string, error) {
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
	tenant, err := client.getTenant(func(tenant *v3tenants.Project) bool { return tenant.ID == tenantID })
	if err != nil {
		return "", err
	}

	if tenant == nil {
		return "", fmt.Errorf("Tenant with ID '%s' not found", tenantID)
	}

	return tenant.Name, nil
}

// GetServiceAuthorization returns the master authorization with full permisions
func (client *keystoneV3Client) GetServiceAuthorization() (schema.Authorization, error) {
	return client.VerifyToken(client.client.TokenID)
}

// GetClient returns openstack client
func (client *keystoneV3Client) GetClient() *gophercloud.ServiceClient {
	return client.client
}

//VerifyToken verifies keystone v2.0 token
func (client *keystoneV2Client) VerifyToken(token string) (schema.Authorization, error) {
	tokenResult, err := verifyV2Token(client.client, token)
	if err != nil {
		return nil, fmt.Errorf("Invalid token")
	}
	tokenBody := tokenResult.(map[string]interface{})["access"]
	userBody := tokenBody.(map[string]interface{})["user"]
	roles := userBody.(map[string]interface{})["roles"]
	roleIDs := []string{}
	for _, roleBody := range roles.([]interface{}) {
		roleIDs = append(roleIDs, roleBody.(map[string]interface{})["name"].(string))
	}
	tokenBodyMap := tokenBody.(map[string]interface{})
	tenant := tokenBodyMap["token"].(map[string]interface{})["tenant"].(map[string]interface{})
	tenantID := tenant["id"].(string)
	tenantName := tenant["name"].(string)
	catalogList := tokenBodyMap["serviceCatalog"].([]interface{})
	catalogObj := []*schema.Catalog{}
	for _, rawCatalog := range catalogList {
		catalog := rawCatalog.(map[string]interface{})
		endPoints := []*schema.Endpoint{}
		rawEndpoints := catalog["endpoints"].([]interface{})
		for _, rawEndpoint := range rawEndpoints {
			endpoint := rawEndpoint.(map[string]interface{})
			region := endpoint["region"].(string)
			adminURL, ok := endpoint["adminURL"].(string)
			if ok {
				endPoints = append(endPoints,
					schema.NewEndpoint(adminURL, region, "admin"))
			}
			internalURL, ok := endpoint["internalURL"].(string)
			if ok {
				endPoints = append(endPoints,
					schema.NewEndpoint(internalURL, region, "internal"))
			}
			publicURL, ok := endpoint["publicURL"].(string)
			if ok {
				endPoints = append(endPoints,
					schema.NewEndpoint(publicURL, region, "public"))
			}
		}
		catalogObj = append(catalogObj, schema.NewCatalog(catalog["name"].(string), catalog["type"].(string), endPoints))
	}
	return schema.NewAuthorization(tenantID, tenantName, token, roleIDs, catalogObj), nil
}

// GetTenantID maps the given v2.0 project name to the tenant's id
func (client *keystoneV2Client) GetTenantID(tenantName string) (string, error) {
	tenant, err := client.getTenant(func(tenant *v2tenants.Tenant) bool { return tenant.Name == tenantName })
	if err != nil {
		return "", err
	}

	if tenant == nil {
		return "", fmt.Errorf("Tenant with name '%s' not found", tenantName)
	}

	return tenant.ID, nil
}

// GetTenantName maps the given v2.0 project id to the tenant's name
func (client *keystoneV2Client) GetTenantName(tenantID string) (string, error) {
	tenant, err := client.getTenant(func(tenant *v2tenants.Tenant) bool { return tenant.ID == tenantID })
	if err != nil {
		return "", err
	}

	if tenant == nil {
		return "", fmt.Errorf("Tenant with ID '%s' not found", tenantID)
	}

	return tenant.Name, nil
}

// GetServiceAuthorization returns the master authorization with full permisions
func (client *keystoneV2Client) GetServiceAuthorization() (schema.Authorization, error) {
	return client.VerifyToken(client.client.TokenID)
}

func (client *keystoneV2Client) getTenant(filter func(*v2tenants.Tenant) bool) (*v2tenants.Tenant, error) {
	opts := v2tenants.ListOpts{}
	pager := v2tenants.List(client.client, &opts)
	var result *v2tenants.Tenant
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		tenantsList, err := v2tenants.ExtractTenants(page)
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

//TODO(nati) this should be implemented in openstack go client side package
func verifyV2Token(c *gophercloud.ServiceClient, token string) (interface{}, error) {
	var result interface{}
	_, err := c.Get(tokenURL(c, token), &result, &gophercloud.RequestOpts{
		OkCodes: []int{200, 203},
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func tokenURL(c *gophercloud.ServiceClient, token string) string {
	return c.ServiceURL("tokens", token)
}

// GetClient returns openstack client
func (client *keystoneV2Client) GetClient() *gophercloud.ServiceClient {
	return client.client
}
