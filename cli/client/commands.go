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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gophercloud/gophercloud"

	"github.com/cloudwan/gohan/schema"
)

var (
	noResponseError             = "No response"
	multipleResourcesFoundError = "Multiple %s with name '%s' found"
	resourceNotFoundError       = "Resource not found"
	unexpectedResponse          = "Unexpected response: %v"
)

type gohanCommand struct {
	Name   string
	Schema *schema.Schema
	Action func(args []string) (string, error)
}

func (gohanClientCLI *GohanClientCLI) request(method, url string, opts *gophercloud.RequestOpts) (interface{}, error) {
	if opts == nil {
		opts = &gophercloud.RequestOpts{
			JSONBody: map[string]interface{}{},
		}
	}
	gohanClientCLI.logRequest(method, url, gohanClientCLI.provider.TokenID, opts.JSONBody.(map[string]interface{}))
	return gohanClientCLI.handleResponse(gohanClientCLI.provider.Request(method, url, opts))
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
		commands = append(commands, gohanClientCLI.getCustomCommands(s)...)
	}
	return commands
}

func (gohanClientCLI *GohanClientCLI) getListCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name:   fmt.Sprintf("%s list", s.ID),
		Schema: s,
		Action: func(args []string) (string, error) {
			_, err := gohanClientCLI.handleArguments(args, s)
			if err != nil {
				return "", err
			}
			url := fmt.Sprintf("%s%s%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, gohanClientCLI.getFieldsParam(true))
			result, err := gohanClientCLI.request("GET", url, nil)
			return gohanClientCLI.formatOutput(s, result), err
		},
	}
}

func (gohanClientCLI *GohanClientCLI) getGetCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name:   fmt.Sprintf("%s show", s.ID),
		Schema: s,
		Action: func(args []string) (string, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("Wrong number of arguments")
			}
			_, err := gohanClientCLI.handleArguments(args[:len(args)-1], s)
			if err != nil {
				return "", err
			}
			id, err := gohanClientCLI.getResourceID(s, args[len(args)-1])
			if err != nil {
				return "", err
			}
			url := fmt.Sprintf("%s%s/%s%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, id, gohanClientCLI.getFieldsParam(true))
			result, err := gohanClientCLI.request("GET", url, nil)
			if err != nil {
				return "", err
			}
			buffer := bytes.NewBufferString("")
			buffer.WriteString(gohanClientCLI.formatOutput(s, result))
			for _, childSchema := range gohanClientCLI.schemas {
				if s.ID == childSchema.Parent {
					buffer.WriteString("\n")
					buffer.WriteString(childSchema.Title)
					buffer.WriteString("\n")
					parentSchemaPropertyID := childSchema.ParentSchemaPropertyID()
					url := fmt.Sprintf("%s%s?%s=%s%s", gohanClientCLI.opts.gohanEndpointURL, childSchema.URL, parentSchemaPropertyID, id, gohanClientCLI.getFieldsParam(false))
					result, err := gohanClientCLI.request("GET", url, nil)
					if err != nil {
						return "", err
					}
					buffer.WriteString(gohanClientCLI.formatOutput(childSchema, result))
				}
			}
			return buffer.String(), nil
		},
	}
}

func (gohanClientCLI *GohanClientCLI) getPostCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name:   fmt.Sprintf("%s create", s.ID),
		Schema: s,
		Action: func(args []string) (string, error) {
			argsMap, err := gohanClientCLI.handleArguments(args, s)
			if err != nil {
				return "", err
			}
			parsedArgs, err := gohanClientCLI.handleRelationArguments(s, argsMap)
			if err != nil {
				return "", err
			}
			opts := gophercloud.RequestOpts{
				JSONBody: parsedArgs,
				OkCodes:  []int{201, 202, 400},
			}
			url := fmt.Sprintf("%s%s", gohanClientCLI.opts.gohanEndpointURL, s.URL)
			result, err := gohanClientCLI.request("POST", url, &opts)
			return gohanClientCLI.formatOutput(s, result), err
		},
	}
}

func (gohanClientCLI *GohanClientCLI) getPutCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name:   fmt.Sprintf("%s set", s.ID),
		Schema: s,
		Action: func(args []string) (string, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("Wrong number of arguments")
			}
			argsMap, err := gohanClientCLI.handleArguments(args[:len(args)-1], s)
			if err != nil {
				return "", err
			}
			parsedArgs, err := gohanClientCLI.handleRelationArguments(s, argsMap)
			if err != nil {
				return "", err
			}
			opts := gophercloud.RequestOpts{
				JSONBody: parsedArgs,
				OkCodes:  []int{200, 201, 202, 400},
			}
			id, err := gohanClientCLI.getResourceID(s, args[len(args)-1])
			if err != nil {
				return "", err
			}
			url := fmt.Sprintf("%s%s/%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, id)
			result, err := gohanClientCLI.request("PUT", url, &opts)
			return gohanClientCLI.formatOutput(s, result), err
		},
	}
}

func (gohanClientCLI *GohanClientCLI) getDeleteCommand(s *schema.Schema) gohanCommand {
	return gohanCommand{
		Name:   fmt.Sprintf("%s delete", s.ID),
		Schema: s,
		Action: func(args []string) (string, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("Wrong number of arguments")
			}
			_, err := gohanClientCLI.handleArguments(args[:len(args)-1], s)
			if err != nil {
				return "", err
			}
			id, err := gohanClientCLI.getResourceID(s, args[len(args)-1])
			if err != nil {
				return "", err
			}
			url := fmt.Sprintf("%s%s/%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, id)
			result, err := gohanClientCLI.request("DELETE", url, nil)
			return gohanClientCLI.formatOutput(s, result), err
		},
	}
}

// Assumes gohan client is called as follows:
// gohan client [common_params...] [action_input] [resource_id]
// where common_params are in form '--name value' and 'name' exists in commonParams
// action_input conforms to action's InputSchema specification
// resource_id is ID of the resource this action acts upon
func (gohanClientCLI *GohanClientCLI) getCustomCommands(s *schema.Schema) []gohanCommand {
	ret := make([]gohanCommand, 0, len(s.Actions))
	for _, act := range s.Actions {
		ret = append(ret, gohanCommand{
			Name:   s.ID + " " + act.ID,
			Schema: s,
			Action: gohanClientCLI.createActionFunc(act, s),
		})
	}
	return ret
}

func (gohanClientCLI *GohanClientCLI) createActionFunc(
	act schema.Action,
	s *schema.Schema,
) func(args []string) (string, error) {
	return func(args []string) (string, error) {
		params, input, id, err := splitArgs(args, &act)
		if err != nil {
			return "", err
		}
		if len(id) > 0 {
			id, err = gohanClientCLI.getResourceID(s, id)
			if err != nil {
				return "", err
			}
		}
		argsMap, err := gohanClientCLI.getCustomArgsAsMap(params, input, act)
		if err != nil {
			return "", err
		}
		opts := gophercloud.RequestOpts{
			JSONBody: argsMap,
			OkCodes:  okCodes(act.Method),
		}
		url := gohanClientCLI.opts.gohanEndpointURL + s.URL + substituteID(act.Path, id)
		result, err := gohanClientCLI.request(act.Method, url, &opts)
		if err != nil {
			return "", err
		}
		result = gohanClientCLI.formatCustomOutput(result)
		return gohanClientCLI.formatOutput(s, result), err
	}
}

func okCodes(method string) []int {
	switch {
	case method == "GET":
		return []int{200}
	case method == "POST":
		return []int{201, 202, 400}
	case method == "PUT":
		return []int{200, 201, 202, 400}
	case method == "PATCH":
		return []int{200, 204}
	case method == "DELETE":
		return []int{202, 204}
	}

	return []int{}
}

func (gohanClientCLI *GohanClientCLI) formatCustomOutput(rawOutput interface{}) interface{} {
	if rawOutput == nil {
		return rawOutput
	}
	switch gohanClientCLI.opts.outputFormat {
	case outputFormatTable:
		return map[string]interface{}{
			"output": rawOutput,
		}
	default:
		// outputFormatJSON
		return rawOutput
	}
}

// Splits command line arguments into id, action input and remaining parameters
func splitArgs(
	args []string,
	action *schema.Action,
) (remainingArgs []string, input []string, id string, err error) {
	remainingArgs = args
	argCount := 0
	takesID := action.TakesID()
	if takesID {
		argCount++
	}
	if action.InputSchema != nil {
		if (len(args)-argCount)%2 == 0 {
			var parameters []string
			parameters, err = action.GetInputParameterNames()
			if err != nil {
				argCount++
			} else {
				argCount += 2 * len(parameters)
			}
		} else {
			argCount++
		}
	}
	if len(args) < argCount {
		err = fmt.Errorf("Wrong number of arguments")
		return
	}
	if err != nil {
		return
	}
	if takesID {
		id = remainingArgs[len(remainingArgs)-1]
		remainingArgs = remainingArgs[:len(remainingArgs)-1]
		argCount--
	}
	if action.InputSchema != nil {
		input = make([]string, argCount)
		length := len(remainingArgs)
		for i := 0; i < argCount; i++ {
			input[i] = remainingArgs[length-argCount+i]
		}
		remainingArgs = remainingArgs[:length-argCount]
	}
	return
}

func substituteID(path, id string) string {
	re := regexp.MustCompile(`(.*/)(:id)(/.*)?$`)
	match := re.FindStringSubmatch(path)
	if len(match) == 0 {
		return path
	}
	return match[1] + id + match[3]
}

func (gohanClientCLI *GohanClientCLI) handleResponse(response *http.Response, err error) (interface{}, error) {
	if response == nil {
		return nil, fmt.Errorf(noResponseError)
	}
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

func (gohanClientCLI *GohanClientCLI) getFieldsParam(prependQuestionMark bool) string {
	if len(gohanClientCLI.opts.fields) == 0 {
		return ""
	}

	param := ""

	if prependQuestionMark {
		param = param + "?"
	}

	for index, field := range gohanClientCLI.opts.fields {
		if index > 0 {
			param = param + "&"
		}

		param = param + "_fields=" + field
	}

	return param
}
