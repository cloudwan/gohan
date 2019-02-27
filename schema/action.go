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
	"sort"
)

// Action struct
type Action struct {
	ID                   string
	Method               string
	Path                 string
	Description          string
	InputSchema          map[string]interface{}
	OutputSchema         map[string]interface{}
	Parameters           map[string]interface{}
	HasResponseOwnership bool
	Protocol             string
}

// NewAction create Action
func NewAction(id, method, path, description, protocol string, inputSchema, outputSchema, parameters map[string]interface{}, hasResponseOwnership bool) Action {
	return Action{
		ID:                   id,
		Method:               method,
		Path:                 path,
		Description:          description,
		InputSchema:          inputSchema,
		OutputSchema:         outputSchema,
		Parameters:           parameters,
		HasResponseOwnership: hasResponseOwnership,
		Protocol:             protocol,
	}
}

// NewActionFromObject create Action object from json
func NewActionFromObject(id string, rawData interface{}) (Action, error) {
	actionData, _ := rawData.(map[string]interface{})
	method, _ := actionData["method"].(string)
	path, _ := actionData["path"].(string)
	description, _ := actionData["description"].(string)
	inputSchema, _ := actionData["input"].(map[string]interface{})
	outputSchema, _ := actionData["output"].(map[string]interface{})
	parameters, _ := actionData["parameters"].(map[string]interface{})
	hasResponseOwnership, _ := actionData["has_response_ownership"].(bool)
	protocol, _ := actionData["protocol"].(string)
	return NewAction(id, method, path, description, protocol, inputSchema, outputSchema, parameters, hasResponseOwnership), nil
}

// TakesID checks if action takes ID as a parameter
func (action *Action) TakesID() bool {
	re := regexp.MustCompile(`.*/:id(/.*)?$`)
	return len(re.FindString(action.Path)) != 0
}

// GetInputType gets action input type
func (action *Action) GetInputType() (string, error) {
	if action.InputSchema == nil {
		return "", fmt.Errorf("Action does not take input")
	}
	inputType, ok := action.InputSchema["type"].(string)
	if !ok {
		return "", fmt.Errorf("Input schema does not have a type")
	}
	return inputType, nil
}

// GetInputParameterNames gets action input parameter names
func (action *Action) GetInputParameterNames() ([]string, error) {
	if action.InputSchema == nil {
		return nil, fmt.Errorf("Action does not take input")
	}
	properties, ok := action.InputSchema["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Input schema does not have properties")
	}

	keys := make([]string, len(properties))
	i := 0
	for k := range properties {
		keys[i] = k
		i++
	}
	return keys, nil
}

// GetInputParameterType gets input parameter type
func (action *Action) GetInputParameterType(parameter string) (string, error) {
	if action.InputSchema == nil {
		return "", fmt.Errorf("Action does not take input")
	}
	properties, ok := action.InputSchema["properties"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("Input schema does not have properties")
	}
	parameterObject, ok := properties[parameter].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("Property with ID %s not found", parameter)
	}
	parameterType, ok := parameterObject["type"].(string)
	if !ok {
		return "", fmt.Errorf("Parameter with ID %s does not have a type", parameter)
	}
	return parameterType, nil
}

// TakesNoArgs checks if action takes no arguments
func (action *Action) TakesNoArgs() bool {
	return action.InputSchema == nil && !action.TakesID()
}

// Sorting
type actions []Action

func (a actions) Len() int {
	return len(a)
}

func (a actions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a actions) Less(i, j int) bool {
	return a[i].ID < a[j].ID
}

// SortActions sort actions by id
func SortActions(schema *Schema) {
	sort.Sort(actions(schema.Actions))
}
