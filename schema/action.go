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

// Action struct
type Action struct {
	ID           string
	Method       string
	Path         string
	Description  string
	InputSchema  map[string]interface{}
	OutputSchema map[string]interface{}
	Parameters   map[string]interface{}
}

// NewAction create Action
func NewAction(id, method, path, description string, inputSchema, outputSchema, parameters map[string]interface{}) Action {
	return Action{
		ID:           id,
		Method:       method,
		Path:         path,
		Description:  description,
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Parameters:   parameters,
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
	return NewAction(id, method, path, description, inputSchema, outputSchema, parameters), nil
}
