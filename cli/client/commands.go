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

package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rackspace/gophercloud"

	"github.com/cloudwan/gohan/schema"
)

var (
	multipleResourcesFoundError = "Multiple %s with name '%s' found"
	resourceNotFoundError       = "Resource not found"
	unexpectedResponse          = "Unexpected response: %v"
)

type gohanCommand struct {
	Name   string
	Action func(args []string) (interface{}, error)
}

func (gohanClientCLI *GohanClientCLI) request(method, url string, opts *gophercloud.RequestOpts) (interface{}, error) {
	if opts == nil {
		opts = &gophercloud.RequestOpts{
			JSONBody: map[string]interface{}{},
		}
	}
	gohanClientCLI.logRequest(method, url, gohanClientCLI.provider.TokenID, opts.JSONBody.(map[string]interface{}))
	return gohanClientCLI.handleResponse(gohanClientCLI.provider.Request(method, url, *opts))
}

func (gohanClientCLI *GohanClientCLI) getCommands() []gohanCommand {
	commands := []gohanCommand{}
	for _, s := range gohanClientCLI.schemas {
		commands = append(commands,
			gohanClientCLI.getListCommand(s),
			gohanClientCLI.getGetCommand(s),
			gohanClientCLI.getPostCommand(s),
			gohanClientCLI.getPutCommand(s),
			gohanClientCLI.getDeleteCommand(s),
		)
	}
	return commands
}

func (gohanClientCLI *GohanClientCLI) getListCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name: fmt.Sprintf("%s list", s.ID),
		Action: func(args []string) (interface{}, error) {
			_, err := gohanClientCLI.handleArguments(args, s)
			if err != nil {
				return nil, err
			}
			url := fmt.Sprintf("%s%s", gohanClientCLI.opts.gohanEndpointURL, s.URL)
			return gohanClientCLI.request("GET", url, nil)
		},
	}
}

func (gohanClientCLI *GohanClientCLI) getGetCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name: fmt.Sprintf("%s show", s.ID),
		Action: func(args []string) (interface{}, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("Wrong number of arguments")
			}
			_, err := gohanClientCLI.handleArguments(args[:len(args)-1], s)
			if err != nil {
				return nil, err
			}
			id, err := gohanClientCLI.getResourceID(s, args[len(args)-1])
			if err != nil {
				return nil, err
			}
			url := fmt.Sprintf("%s%s/%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, id)
			return gohanClientCLI.request("GET", url, nil)
		},
	}
}

func (gohanClientCLI *GohanClientCLI) getPostCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name: fmt.Sprintf("%s create", s.ID),
		Action: func(args []string) (interface{}, error) {
			argsMap, err := gohanClientCLI.handleArguments(args, s)
			if err != nil {
				return nil, err
			}
			parsedArgs, err := gohanClientCLI.handleRelationArguments(s, argsMap)
			if err != nil {
				return nil, err
			}
			opts := gophercloud.RequestOpts{
				JSONBody: parsedArgs,
				OkCodes:  []int{201, 202, 400},
			}
			url := fmt.Sprintf("%s%s", gohanClientCLI.opts.gohanEndpointURL, s.URL)
			return gohanClientCLI.request("POST", url, &opts)
		},
	}
}

func (gohanClientCLI *GohanClientCLI) getPutCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name: fmt.Sprintf("%s set", s.ID),
		Action: func(args []string) (interface{}, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("Wrong number of arguments")
			}
			argsMap, err := gohanClientCLI.handleArguments(args[:len(args)-1], s)
			if err != nil {
				return nil, err
			}
			parsedArgs, err := gohanClientCLI.handleRelationArguments(s, argsMap)
			if err != nil {
				return nil, err
			}
			opts := gophercloud.RequestOpts{
				JSONBody: parsedArgs,
				OkCodes:  []int{200, 201, 202, 400},
			}
			id, err := gohanClientCLI.getResourceID(s, args[len(args)-1])
			if err != nil {
				return nil, err
			}
			url := fmt.Sprintf("%s%s/%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, id)
			return gohanClientCLI.request("PUT", url, &opts)
		},
	}
}

func (gohanClientCLI *GohanClientCLI) getDeleteCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name: fmt.Sprintf("%s delete", s.ID),
		Action: func(args []string) (interface{}, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("Wrong number of arguments")
			}
			_, err := gohanClientCLI.handleArguments(args[:len(args)-1], s)
			if err != nil {
				return nil, err
			}
			id, err := gohanClientCLI.getResourceID(s, args[len(args)-1])
			if err != nil {
				return nil, err
			}
			url := fmt.Sprintf("%s%s/%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, id)
			return gohanClientCLI.request("DELETE", url, nil)
		},
	}
}

func (gohanClientCLI *GohanClientCLI) handleResponse(response *http.Response, err error) (interface{}, error) {
	defer response.Body.Close()
	var result interface{}
	json.NewDecoder(response.Body).Decode(&result)

	gohanClientCLI.logResponse(response.Status, result)

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf(resourceNotFoundError)
	}
	if err != nil {
		return nil, fmt.Errorf(unexpectedResponse, response.Status)
	}

	return result, err
}
