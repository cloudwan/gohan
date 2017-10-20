//
// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package otto

import (
	"fmt"

	"github.com/xyproto/otto"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/resources"
)

const (
	noEnvironmentForSchemaErrorMessageFormat = "No environment for schema '%s'"
	wrongResponseErrorMessageFormat          = "Wrong response from %s"
	notADictionaryErrorMessageFormat         = "Not a dictionary: '%v'"
)

func handleChainError(env *Environment, call *otto.FunctionCall, err error) {
	switch err := err.(type) {
	default:
		ThrowOttoException(call, err.Error())
	case resources.ResourceError:
		ThrowOtto(call, "ResourceException", err.Message, err.Problem)
	case extension.Error:
		exceptionInfo, _ := env.VM.ToValue(err.ExceptionInfo)
		ThrowOtto(call, "ExtensionException", err.Error(), exceptionInfo)
	}
}

func init() {
	gohanChainingInit := func(env *Environment) {
		vm := env.VM
		builtins := map[string]interface{}{
			"gohan_model_list": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_list", 3)
				context, err := GetMap(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				filterMap, err := GetMap(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				resources, err := GohanModelList(context, schemaID, filterMap)
				if err != nil {
					handleChainError(env, &call, err)
				}

				value, _ := vm.ToValue(resources)
				return value
			},
			"gohan_model_fetch": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_fetch", 4)
				context, err := GetMap(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				resourceID, err := GetString(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				tenantIDs, err := GetStringList(call.Argument(3))
				if err != nil {
					tenantIDs = nil
				}

				resource, err := GohanModelFetch(context, schemaID, resourceID, tenantIDs)
				if err != nil {
					handleChainError(env, &call, err)
				}

				value, _ := vm.ToValue(resource)
				return value
			},
			"gohan_model_create": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_create", 3)
				context, err := GetMap(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				dataMap, err := GetMap(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				resource, err := GohanModelCreate(context, schemaID, dataMap)
				if err != nil {
					handleChainError(env, &call, err)
				}

				value, _ := vm.ToValue(resource)
				return value
			},
			"gohan_model_update": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_update", 5)
				context, err := GetMap(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				resourceID, err := GetString(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				dataMap, err := GetMap(call.Argument(3))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				tenantIDs, err := GetStringList(call.Argument(4))
				if err != nil {
					tenantIDs = nil
				}

				resource, err := GohanModelUpdate(context, schemaID, resourceID, dataMap, tenantIDs)
				if err != nil {
					handleChainError(env, &call, err)
				}

				value, _ := vm.ToValue(resource)
				return value
			},
			"gohan_model_delete": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_delete", 3)
				context, err := GetMap(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				schemaID, err := GetString(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				resourceID, err := GetString(call.Argument(2))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				err = GohanModelDelete(context, schemaID, resourceID)
				if err != nil {
					handleChainError(env, &call, err)
				}

				return otto.Value{}
			},
		}

		for name, object := range builtins {
			vm.Set(name, object)
		}
	}
	RegisterInit(gohanChainingInit)
}

//GohanModelList lists gohan resources and running extensions
func GohanModelList(context map[string]interface{}, schemaID string,
	filterMap map[string]interface{}) (interface{}, error) {

	currentSchema, err := getSchema(schemaID)
	if err != nil {
		return nil, err
	}
	context["schema"] = currentSchema
	context["path"] = currentSchema.GetPluralURL()

	filter := map[string]interface{}{}
	for key, value := range filterMap {
		switch value := value.(type) {
		default:
			return nil, fmt.Errorf("Filter not a string nor array of strings")
		case string:
			filter[key] = value
		case []interface{}:
			for _, val := range value {
				v, ok := val.(string)
				if !ok {
					return nil, fmt.Errorf("Filter not a string nor array of strings")
				}
				filter[key] = v
			}
		case []string:
			for _, val := range value {
				filter[key] = val
			}

		}
	}

	if err := resources.GetResourcesInTransaction(
		context, currentSchema, filter, nil); err != nil {
		return nil, err
	}
	response, ok := context["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("No response")
	}
	resources, ok := response[currentSchema.Plural]
	if !ok {
		return nil, fmt.Errorf(wrongResponseErrorMessageFormat, "GetMultipleResources.")
	}
	return resources, nil
}

//GohanModelFetch fetch gohan resource and running extensions
func GohanModelFetch(context map[string]interface{}, schemaID string, resourceID string,
	tenantIDs []string) (interface{}, error) {

	currentSchema, err := getSchema(schemaID)
	if err != nil {
		return nil, err
	}
	context["schema"] = currentSchema
	context["path"] = currentSchema.GetPluralURL()

	if err := resources.GetSingleResourceInTransaction(
		context, currentSchema, resourceID, tenantIDs); err != nil {
		return nil, err
	}
	response, ok := context["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("No response")
	}
	return response[currentSchema.Singular], nil
}

//GohanModelCreate creates gohan resource and running extensions
func GohanModelCreate(context map[string]interface{}, schemaID string,
	dataMap map[string]interface{}) (interface{}, error) {

	currentSchema, err := getSchema(schemaID)
	if err != nil {
		return nil, err
	}
	context["schema"] = currentSchema
	context["path"] = currentSchema.GetPluralURL()

	manager := schema.GetManager()
	resourceObj, err := manager.LoadResource(currentSchema.ID, dataMap)
	if err != nil {
		return nil, err
	}

	if err := resources.CreateResourceInTransaction(
		context, currentSchema, resourceObj); err != nil {
		return nil, err
	}
	response, ok := context["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("No response")
	}
	return response[currentSchema.Singular], nil
}

//GohanModelUpdate updates gohan resource and running extensions
func GohanModelUpdate(context map[string]interface{}, schemaID string, resourceID string, dataMap map[string]interface{}, tenantIDs []string) (interface{}, error) {

	currentSchema, err := getSchema(schemaID)
	if err != nil {
		return nil, err
	}
	context["schema"] = currentSchema
	context["path"] = currentSchema.GetPluralURL()

	err = resources.UpdateResourceInTransaction(context, currentSchema, resourceID, dataMap, tenantIDs)
	if err != nil {
		return nil, err
	}
	response, ok := context["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("No response")
	}
	return response[currentSchema.Singular], nil
}

//GohanModelDelete deletes gohan resources and running extensions
func GohanModelDelete(context map[string]interface{}, schemaID string, resourceID string) error {

	currentSchema, err := getSchema(schemaID)
	if err != nil {
		return err
	}
	context["schema"] = currentSchema
	context["path"] = currentSchema.GetPluralURL()

	return resources.DeleteResourceInTransaction(context, currentSchema, resourceID)
}
