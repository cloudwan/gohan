// Copyright (C) 2016  Juniper Networks, Inc.
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

package lib

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudwan/gohan/cloud"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

func newClient(endpoint string) (*gophercloud.ProviderClient, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	hadPath := u.Path != ""
	u.Path, u.RawQuery, u.Fragment = "", "", ""
	base := u.String()

	endpoint = gophercloud.NormalizeURL(endpoint)
	base = gophercloud.NormalizeURL(base)
	timeout := time.Duration(20 * time.Second)

	if !hadPath {
		endpoint = ""
	}
	return &gophercloud.ProviderClient{
		IdentityBase:     base,
		IdentityEndpoint: endpoint,
		HTTPClient: http.Client{
			Timeout: timeout,
		},
	}, nil

}

func authenticatedClient(options gophercloud.AuthOptions) (*gophercloud.ProviderClient, error) {
	client, err := newClient(options.IdentityEndpoint)
	if err != nil {
		return nil, err
	}

	err = openstack.Authenticate(client, options)
	if err != nil {
		return nil, err
	}
	return client, nil
}

//GetOpenstackClient makes openstack client
func GetOpenstackClient(authURL, userName, password, domainName, tenantName, version string) (*gophercloud.ServiceClient, error) {
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
	client.HTTPClient = cloud.NewHTTPClient()
	if version == "v2.0" {
		return openstack.NewIdentityV2(client), nil
	} else if version == "v3" {
		return openstack.NewIdentityV3(client), nil
	} else {
		return nil, fmt.Errorf("Unsupported keystone version: %s", version)
	}
}

//OpenstackToken get auth token from client
func OpenstackToken(client *gophercloud.ServiceClient) string {
	return client.TokenID
}

//OpenstackGet gets a resource using OpenStack API
func OpenstackGet(client *gophercloud.ServiceClient, url string) (interface{}, error) {
	var response interface{}
	_, err := client.Get(url, &response, nil)
	if err != nil {
		return nil, err
	}
	return response, nil
}

//OpenstackEnsure keep resource status to sync
func OpenstackEnsure(client *gophercloud.ServiceClient, url string, postURL string, data interface{}) (interface{}, error) {
	var response interface{}
	resp, err := client.Get(url, &response, nil)
	if err != nil {
		if resp.StatusCode != http.StatusNotFound {
			return nil, err
		}
		return OpenstackPost(client, postURL, data)
	}
	return OpenstackPut(client, url, data)
}

//OpenstackPut puts a resource using OpenStack API
func OpenstackPut(client *gophercloud.ServiceClient, url string, data interface{}) (interface{}, error) {
	var response interface{}
	_, err := client.Put(url, data, &response, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 202},
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

//OpenstackPost posts a resource using OpenStack API
func OpenstackPost(client *gophercloud.ServiceClient, url string, data interface{}) (interface{}, error) {
	var response interface{}
	_, err := client.Post(url, data, &response, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 202},
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

//OpenstackDelete deletes a resource using OpenStack API
func OpenstackDelete(client *gophercloud.ServiceClient, url string) (interface{}, error) {
	_, err := client.Delete(url, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 202, 204, 404},
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

//OpenstackEndpoint returns API endpoint for each service name
func OpenstackEndpoint(client *gophercloud.ServiceClient, endpointType, name, region, availability string) (interface{}, error) {
	if availability == "" {
		availability = "public"
	}
	return client.EndpointLocator(
		gophercloud.EndpointOpts{
			Type:         endpointType,
			Name:         name,
			Region:       region,
			Availability: gophercloud.Availability(availability),
		})
}
