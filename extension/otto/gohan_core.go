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

	"github.com/cloudwan/gohan/schema"
	"github.com/op/go-logging"

	l "github.com/cloudwan/gohan/log"
)

var log = logging.MustGetLogger(l.GetModuleName())

func init() {
	gohanInit := func(env *Environment) {
		vm := env.VM
		builtins := map[string]interface{}{
			"require": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "require", 1)
				moduleName, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				value, _ := vm.ToValue(RequireModule(moduleName))
				return value
			},
			"gohan_schemas": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_schemas", 0)
				manager := schema.GetManager()
				response := []interface{}{}
				for _, schema := range manager.OrderedSchemas() {
					response = append(response, schema)
				}
				value, _ := vm.ToValue(response)
				return value
			},
			"gohan_schema_url": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_schema_url", 1)
				schemaID, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
				manager := schema.GetManager()
				schema, ok := manager.Schema(schemaID)
				if !ok {
					ThrowOttoException(&call, unknownSchemaErrorMesssageFormat, schemaID)
				}
				value, _ := vm.ToValue(schema.URL)
				return value
			},
			"gohan_policies": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_policies", 0)
				manager := schema.GetManager()
				response := []interface{}{}
				for _, policy := range manager.Policies() {
					response = append(response, policy.RawData)
				}
				value, _ := vm.ToValue(response)
				return value
			},
		}

		for name, object := range builtins {
			vm.Set(name, object)
		}

		err := env.Load("<Gohan built-in exceptions>", `
		function BaseException() {
		  this.fields = ["name", "message"]
		  this.message = "";
		  this.name = "BaseException";

		  this.toDict = function () {
		    return _.reduce(this.fields, function(resp, field) {
		      resp[field] = this[field];
		      return resp;
		    }, {}, this);
		  };

		  this.toString = function () {
		    return this.name.concat("(").concat(this.message).concat(")");
		  };
		}

		function CustomException(msg, code) {
		  BaseException.call(this);
		  this.message = msg;
		  this.name = "CustomException";
		  this.code = code;
		  this.fields.push("code");
		}
		CustomException.prototype = Object.create(BaseException.prototype);

		function ResourceException(msg, problem) {
		  BaseException.call(this);
		  this.message = msg;
		  this.name = "ResourceException";
		  this.problem = problem;
		  this.fields.push("problem");
		}
		ResourceException.prototype = Object.create(BaseException.prototype);

		function ExtensionException(msg, inner_exception) {
		  BaseException.call(this);
		  this.message = msg;
		  this.name = "ExtensionException";
		  this.inner_exception = inner_exception;
		  this.fields.push("inner_exception");
		}
		ExtensionException.prototype = Object.create(BaseException.prototype);
		`)
		if err != nil {
			log.Fatal(err)
		}

		err = env.Load("<Gohan built-ins>", `
		var gohan_handler = {}
		function gohan_register_handler(event_type, func){
		  if(_.isUndefined(gohan_handler[event_type])){
		    gohan_handler[event_type] = [];
		  }
		  gohan_handler[event_type].push(func)
		}

		function gohan_handle_event(event_type, context){
		  if(_.isUndefined(gohan_handler[event_type])){
		    return;
		  }

		  for (var i = 0; i < gohan_handler[event_type].length; ++i) {
		    try {
		      gohan_handler[event_type][i](context);
		      //backwards compatibility
		      if (!_.isUndefined(context.response_code)) {
		        throw new CustomException(context.response, context.response_code);
		      }
		    } catch(e) {
		      if (e instanceof BaseException) {
		        context.exception = e.toDict();
		        context.exception_message = event_type.concat(": ").concat(e.toString());
		      } else {
		        throw e;
		      }
		    }
		  }
		}
		`)
		if err != nil {
			log.Fatal(err)
		}
	}
	RegisterInit(gohanInit)
}
