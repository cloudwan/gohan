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

package otto

import (
	"github.com/dop251/otto"

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
		throwOtto(call, "ResourceException", err.Message, err.Problem)
	case extension.Error:
		exceptionInfo, _ := env.VM.ToValue(err.ExceptionInfo)
		throwOtto(call, "ExtensionException", err.Error(), exceptionInfo)
	}
}

func init() {
	gohanChainingInit := func(env *Environment) {
		vm := env.VM
		builtins := map[string]interface{}{
			"gohan_model_list": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_list", 3)
				rawContext, _ := call.Argument(0).Export()
				context, ok := rawContext.(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, noContextMessage)
				}
				schemaID := call.Argument(1).String()
				manager := schema.GetManager()
				currentSchema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				context["schema"] = currentSchema
				context["path"] = currentSchema.GetPluralURL()
				filterObj, _ := call.Argument(2).Export()
				filter := map[string]interface{}{}
				if filterObj != nil {
					filterMap := filterObj.(map[string]interface{})
					for key, value := range filterMap {
						switch value := value.(type) {
						default:
							ThrowOttoException(&call, "Filter not a string nor array of strings")
						case string:
							filter[key] = value
						case []interface{}:
							for _, val := range value {
								v, ok := val.(string)
								if !ok {
									ThrowOttoException(&call, "Filter not a string nor array of strings")
								}
								filter[key] = v
							}
						}
					}
				}
				if err := resources.GetResourcesInTransaction(
					context, currentSchema, filter, nil); err != nil {
					handleChainError(env, &call, err)
				}
				response, ok := context["response"].(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, "No response")
				}
				resources, ok := response[currentSchema.Plural]
				if !ok {
					ThrowOttoException(&call, wrongResponseErrorMessageFormat, "GetMultipleResources.")
				}
				value, _ := vm.ToValue(resources)
				return value
			},
			"gohan_model_fetch": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_fetch", 4)
				rawContext, _ := call.Argument(0).Export()
				context, ok := rawContext.(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, noContextMessage)
				}
				schemaID := call.Argument(1).String()
				manager := schema.GetManager()
				currentSchema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				context["schema"] = currentSchema
				context["path"] = currentSchema.GetPluralURL()
				resourceID := call.Argument(2).String()
				rawTenantIDs, _ := call.Argument(3).Export()
				tenantIDs, ok := rawTenantIDs.([]string)
				if !ok {
					tenantIDs = nil
				}
				if err := resources.GetSingleResourceInTransaction(
					context, currentSchema, resourceID, tenantIDs); err != nil {
					handleChainError(env, &call, err)
				}
				response, ok := context["response"].(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, "No response")
				}
				resource := response[currentSchema.Singular]
				value, _ := vm.ToValue(resource)
				return value
			},
			"gohan_model_create": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_create", 3)
				rawContext, _ := call.Argument(0).Export()
				context, ok := rawContext.(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, noContextMessage)
				}
				schemaID := call.Argument(1).String()
				manager := schema.GetManager()
				currentSchema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				context["schema"] = currentSchema
				context["path"] = currentSchema.GetPluralURL()
				data, _ := call.Argument(2).Export()
				dataMap, ok := data.(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, notADictionaryErrorMessageFormat, dataMap)
				}
				resourceObj, err := manager.LoadResource(currentSchema.ID, dataMap)
				if err != nil {
					handleChainError(env, &call, err)
				}
				if err := resources.CreateResourceInTransaction(
					context, resourceObj); err != nil {
					handleChainError(env, &call, err)
				}
				response, ok := context["response"].(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, "No response")
				}
				resource := response[currentSchema.Singular]
				value, _ := vm.ToValue(resource)
				return value
			},
			"gohan_model_update": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_update", 5)
				rawContext, _ := call.Argument(0).Export()
				context, ok := rawContext.(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, noContextMessage)
				}
				schemaID := call.Argument(1).String()
				manager := schema.GetManager()
				currentSchema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				context["schema"] = currentSchema
				context["path"] = currentSchema.GetPluralURL()
				resourceID := call.Argument(2).String()
				data, _ := call.Argument(3).Export()

				rawTenantIDs, _ := call.Argument(4).Export()
				tenantIDs, ok := rawTenantIDs.([]string)
				if !ok {
					tenantIDs = nil
				}

				dataMap, ok := data.(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, notADictionaryErrorMessageFormat, dataMap)
				}
				err := resources.UpdateResourceInTransaction(context, currentSchema, resourceID, dataMap, tenantIDs)
				if err != nil {
					handleChainError(env, &call, err)
				}
				response, ok := context["response"].(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, "No response")
				}
				resource := response[currentSchema.Singular]
				value, _ := vm.ToValue(resource)
				return value
			},
			"gohan_model_delete": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_model_delete", 3)
				rawContext, _ := call.Argument(0).Export()
				context, ok := rawContext.(map[string]interface{})
				if !ok {
					ThrowOttoException(&call, noContextMessage)
				}
				schemaID := call.Argument(1).String()
				manager := schema.GetManager()
				currentSchema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				context["schema"] = currentSchema
				context["path"] = currentSchema.GetPluralURL()
				resourceID := call.Argument(2).String()
				err := resources.DeleteResourceInTransaction(context, currentSchema, resourceID)
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
