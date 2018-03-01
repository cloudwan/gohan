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

package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/go-martini/martini"
	"github.com/rackspace/gophercloud"
)

type tenant struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type token struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires"`
	Tenant    tenant    `json:"tenant"`
}

type role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var allTenants = []tenant{
	{
		ID:          "fc394f2ab2df4114bde39905f800dc57",
		Name:        "demo",
		Description: "Demo tenant",
		Enabled:     true,
	},
	{
		ID:          "acf5662bbff44060b93ac3db3c25a590",
		Name:        "other",
		Description: "Other tenant",
		Enabled:     true,
	},
}

func getToken(id string, t tenant) token {
	return token{
		ID:        id,
		ExpiresAt: time.Now().Add(24 * time.Hour).In(time.UTC),
		Tenant:    t,
	}
}

var serviceCatalog = []interface{}{
	map[string]interface{}{
		"type": "gohan",
		"name": "Gohan",
		"endpoints": []interface{}{
			map[string]interface{}{
				"adminURL":    "http://127.0.0.1:9091",
				"internalURL": "http://127.0.0.1:9091",
				"publicURL":   "http://127.0.0.1:9091",
				"region":      "RegionOne",
				"id":          "2dad48f09e2a447a9bf852bcd93548ef",
			},
		},
	},
}

var fakeTokens = map[string]interface{}{
	"admin_token": map[string]interface{}{
		"access": map[string]interface{}{
			"token":          getToken("admin_token", allTenants[0]),
			"serviceCatalog": serviceCatalog,
			"user": map[string]interface{}{
				"id":   "admin",
				"name": "admin",
				"roles": []role{
					{
						Name: "admin",
					},
				},
				"roles_links": map[string]interface{}{},
				"username":    "admin",
			},
		},
	},
	"demo_token": map[string]interface{}{
		"access": map[string]interface{}{
			"token":          getToken("demo_token", allTenants[0]),
			"serviceCatalog": serviceCatalog,
			"user": map[string]interface{}{
				"id":   "demo",
				"name": "demo",
				"roles": []role{
					{
						Name: "Member",
					},
				},
				"roles_links": map[string]interface{}{},
				"username":    "demo",
			},
		},
	},
	"power_user_token": map[string]interface{}{
		"access": map[string]interface{}{
			"token":          getToken("power_user_token", allTenants[1]),
			"serviceCatalog": serviceCatalog,
			"user": map[string]interface{}{
				"id":   "power_user",
				"name": "power_user",
				"roles": []role{
					{
						Name: "Member",
					},
				},
				"roles_links": map[string]interface{}{},
				"username":    "power_user",
			},
		},
	},
	"visible_token": map[string]interface{}{
		"access": map[string]interface{}{
			"token":          getToken("visible_token", allTenants[0]),
			"serviceCatalog": serviceCatalog,
			"user": map[string]interface{}{
				"id":   "visible",
				"name": "visible",
				"roles": []role{
					{
						Name: "Visible",
					},
				},
				"roles_links": map[string]interface{}{},
				"username":    "visible",
			},
		},
	},
	"hidden_token": map[string]interface{}{
		"access": map[string]interface{}{
			"token":          getToken("hidden_token", allTenants[0]),
			"serviceCatalog": serviceCatalog,
			"user": map[string]interface{}{
				"id":   "hidden",
				"name": "hidden",
				"roles": []role{
					{
						Name: "Hidden",
					},
				},
				"roles_links": map[string]interface{}{},
				"username":    "hidden",
			},
		},
	},
}

//ReadJSON reads JSON from http request
func ReadJSON(r *http.Request) (map[string]interface{}, error) {
	var data interface{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	if _, ok := data.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("request body is not a data dictionary")
	}
	return data.(map[string]interface{}), nil
}

//FakeIdentity middleware
type FakeIdentity struct{}

//VerifyToken fake verify
func (*FakeIdentity) VerifyToken(tokenID string) (schema.Authorization, error) {
	rawToken, ok := fakeTokens[tokenID]
	if !ok {
		return nil, fmt.Errorf("authentication error")
	}

	access, _ := rawToken.(map[string]interface{})["access"].(map[string]interface{})
	tenantID := access["token"].(token).Tenant.ID
	tenantName := access["token"].(token).Tenant.Name
	role := access["user"].(map[string]interface{})["roles"].([]role)[0].Name

	return schema.NewAuthorization(tenantID, tenantName, tokenID, []string{role}, nil), nil
}

// GetTenantID maps the given tenant name to the tenant's ID
func (*FakeIdentity) GetTenantID(tenantName string) (string, error) {
	for _, tenant := range allTenants {
		if tenant.Name == tenantName {
			return tenant.ID, nil
		}
	}

	return "", nil
}

// GetTenantName maps the given tenant ID to the tenant's name
func (*FakeIdentity) GetTenantName(tenantID string) (string, error) {
	for _, tenant := range allTenants {
		if tenant.ID == tenantID {
			return tenant.Name, nil
		}
	}

	return "", nil
}

// GetServiceAuthorization returns the master authorization with full permission
func (identity *FakeIdentity) GetServiceAuthorization() (schema.Authorization, error) {
	return identity.VerifyToken("admin_token")
}

//FakeKeystone server for only test purpose
func FakeKeystone(martini *martini.ClassicMartini) {
	//mocking keystone v2.0 API
	martini.Post("/v2.0/tokens", func(w http.ResponseWriter, r *http.Request) {
		authRequest, err := ReadJSON(r)
		if err != nil {
			http.Error(w, "", http.StatusBadRequest)
		}
		var tokenKey string
		username, err := util.GetByJSONPointer(authRequest, "/auth/passwordCredentials/username")
		if err == nil {
			tokenKey = fmt.Sprintf("%v_token", username)
		} else {
			username, err = util.GetByJSONPointer(authRequest, "/auth/token/id")
			tokenKey = fmt.Sprintf("%v", username)
			if err != nil {
				http.Error(w, "", http.StatusBadRequest)
			}
		}

		token, ok := fakeTokens[tokenKey]
		if !ok {
			http.Error(w, "", http.StatusUnauthorized)
		}

		serializedToken, _ := json.Marshal(token)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(serializedToken)))
		w.Write(serializedToken)
	})

	martini.Get("/v2.0/tenants", func(w http.ResponseWriter, r *http.Request) {
		tenants := map[string]interface{}{
			"tenants": allTenants,
		}

		serializedToken, _ := json.Marshal(tenants)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(serializedToken)))
		w.Write(serializedToken)
	})

	for tokenID, rawToken := range fakeTokens {
		serializedToken, _ := json.Marshal(rawToken)
		martini.Get("/v2.0/tokens/"+tokenID, func(w http.ResponseWriter, r *http.Request) {
			w.Write(serializedToken)
		})
	}
}

// GetClient returns openstack client
func (identity *FakeIdentity) GetClient() *gophercloud.ServiceClient {
	return nil
}
