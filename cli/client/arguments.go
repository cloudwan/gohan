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
	u "net/url"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud"

	"github.com/cloudwan/gohan/schema"
)

func getValueFromRaw(key, rawValue, valueType string) (interface{}, error) {
	var value interface{}
	var err error
	if rawValue == "<null>" {
		value = nil
	} else {
		switch valueType {
		case "integer", "number":
			value, err = strconv.ParseInt(rawValue, 10, 64)
		case "boolean":
			value, err = strconv.ParseBool(rawValue)
		case "array", "object":
			err = json.Unmarshal([]byte(rawValue), &value)
		default:
			value = rawValue
		}
		if err != nil {
			return nil, fmt.Errorf("Error parsing parameter '%v': %v", key, err)
		}
	}
	return value, nil
}

func (gohanClientCLI *GohanClientCLI) getCustomArgsAsMap(
	args []string,
	actionInput []string,
	action schema.Action,
) (map[string]interface{}, error) {
	argsMap := map[string]interface{}{}
	if action.InputSchema != nil {
		inputType, err := action.GetInputType()
		if err != nil {
			return nil, err
		}
		if len(actionInput) == 1 {
			value, err := getValueFromRaw("input", actionInput[0], inputType)
			if err != nil {
				return nil, err
			}
			argsMap[action.ID] = value
		} else {
			if inputType != "object" {
				err := fmt.Errorf("Input schema type must be an object to pass input with parameters")
				return nil, err
			}
			data := map[string]interface{}{}
			for i := 0; i < len(actionInput); i += 2 {
				key := strings.TrimPrefix(actionInput[i], "--")
				valueType, err := action.GetInputParameterType(key)
				if err != nil {
					return nil, err
				}
				value, err := getValueFromRaw(key, actionInput[i+1], valueType)
				if err != nil {
					return nil, err
				}
				data[key] = value
			}
			argsMap[action.ID] = data
		}
	}
	for i := 0; i < len(args); i += 2 {
		key := strings.TrimPrefix(args[i], "--")
		if _, ok := commonParams[key]; !ok {
			return nil, fmt.Errorf("Error parsing parameter %s", key)
		}
		value := args[i+1]
		argsMap[key] = value
	}
	err := gohanClientCLI.handleCommonArguments(argsMap)
	return argsMap, err
}

func (gohanClientCLI *GohanClientCLI) handleArguments(args []string, s *schema.Schema) (map[string]interface{}, error) {
	argsMap, err := getArgsAsMap(args, s)
	if err != nil {
		return nil, err
	}
	err = gohanClientCLI.handleCommonArguments(argsMap)
	if err != nil {
		return nil, err
	}
	return argsMap, nil
}

func getArgsAsMap(args []string, s *schema.Schema) (map[string]interface{}, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("Parameters should be in [--param-name value]... format")
	}
	result := map[string]interface{}{}
	for i := 0; i < len(args); i += 2 {
		key := strings.TrimPrefix(args[i], "--")
		valueType := "string"
		if property, err := s.GetPropertyByID(key); err == nil {
			valueType = property.Type
		}
		value, err := getValueFromRaw(key, args[i+1], valueType)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

func (gohanClientCLI *GohanClientCLI) handleCommonArguments(args map[string]interface{}) error {
	if outputFormatOpt, ok := args[outputFormatKey]; ok {
		outputFormat, err := findOutputFormat(outputFormatOpt)
		if err != nil {
			return err
		}
		delete(args, outputFormatKey)
		gohanClientCLI.opts.outputFormat = outputFormat
	}

	if verbosity, ok := args[logLevelKey]; ok {
		logLevel, err := parseLogLevel(verbosity)
		if err != nil {
			return err
		}
		delete(args, logLevelKey)
		gohanClientCLI.opts.logLevel = logLevel
		setUpLogging(logLevel)
	}

	if fieldsOpt, ok := args[fieldsKey]; ok {
		fields, err := findFields(fieldsOpt)
		if err != nil {
			return err
		}
		delete(args, fieldsKey)
		gohanClientCLI.opts.fields = fields
	}

	return nil
}

func (gohanClientCLI *GohanClientCLI) handleRelationArguments(s *schema.Schema, args map[string]interface{}) (map[string]interface{}, error) {
	parsedArgs := map[string]interface{}{}
	for arg, value := range args {
		if arg == s.Parent {
			parentID, err := gohanClientCLI.getResourceIDForSchemaID(s.Parent, value.(string))
			if err != nil {
				return nil, err
			}
			parsedArgs[s.ParentSchemaPropertyID()] = parentID
			continue
		}
		property, _ := s.GetPropertyByID(arg)
		if property == nil {
			property, _ = s.GetPropertyByID(arg + "_id")
			if property != nil && property.Relation != "" {
				relatedID, err := gohanClientCLI.getResourceIDForSchemaID(property.Relation, value.(string))
				if err != nil {
					return nil, err
				}
				parsedArgs[property.ID] = relatedID
				continue
			}
		}
		parsedArgs[arg] = value
	}
	return parsedArgs, nil
}

func (gohanClientCLI *GohanClientCLI) getResourceIDForSchemaID(schemaID, identifier string) (string, error) {
	relatedSchema, err := gohanClientCLI.getSchemaByID(schemaID)
	if err != nil {
		return "", err
	}
	return gohanClientCLI.getResourceID(relatedSchema, identifier)
}

func (gohanClientCLI *GohanClientCLI) getResourceID(s *schema.Schema, identifier string) (string, error) {
	url := fmt.Sprintf("%s%s/%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, u.QueryEscape(identifier))
	gohanClientCLI.logRequest("GET", url, gohanClientCLI.provider.TokenID, nil)
	_, err := gohanClientCLI.handleResponse(gohanClientCLI.provider.Request(http.MethodGet, url, &gophercloud.RequestOpts{}))
	if err == nil {
		return identifier, nil
	}

	url = fmt.Sprintf("%s%s?name=%s", gohanClientCLI.opts.gohanEndpointURL, s.URL, u.QueryEscape(identifier))
	gohanClientCLI.logRequest("GET", url, gohanClientCLI.provider.TokenID, nil)
	result, err := gohanClientCLI.handleResponse(gohanClientCLI.provider.Request(http.MethodGet, url, &gophercloud.RequestOpts{}))
	if err != nil {
		return "", err
	}
	resourcesMap, ok := result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf(resourceNotFoundError)
	}
	resources, ok := resourcesMap[s.Plural].([]interface{})
	if !ok {
		return "", fmt.Errorf(resourceNotFoundError)
	}

	if len(resources) == 1 {
		return resources[0].(map[string]interface{})["id"].(string), nil
	}
	if len(resources) > 1 {
		return "", fmt.Errorf(multipleResourcesFoundError, s.Plural, identifier)
	}

	return "", fmt.Errorf(resourceNotFoundError)
}
