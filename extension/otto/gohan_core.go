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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ddliu/motto"

	"github.com/cloudwan/gohan/extension"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	"github.com/xyproto/otto"
)

var log = l.NewLogger()

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
				value, err := require(moduleName, vm)
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}
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
			"gohan_trigger_event": func(call otto.FunctionCall) otto.Value {
				VerifyCallArguments(&call, "gohan_trigger_event", 2)

				event, err := GetString(call.Argument(0))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				context, err := GetMap(call.Argument(1))
				if err != nil {
					ThrowOttoException(&call, err.Error())
				}

				schemaID := ""

				if s, ok := context["schema"]; ok {
					schemaID = s.(*schema.Schema).ID
				} else {
					log.Panic("gohan_trigger_event: schema not found")
				}

				envManager := extension.GetManager()
				envManager.HandleEventInAllEnvironments(context, event, schemaID)

				value, _ := vm.ToValue(nil)

				return value
			},
			"gohan_closers": []io.Closer{},
		}

		for name, object := range builtins {
			vm.Set(name, object)
		}
		loadNPMModules()
		registerBuiltinModules(vm)

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
		function TimeOutException(msg){
		  BaseException.call(this);
		  this.message = msg;
		  this.name = "TimeOutException";
		  this.code = 503;
		  this.fields.push("code");
		}
		TimeOutException.prototype = Object.create(BaseException.prototype);

		function NotFoundException(msg) {
		   BaseException.call(this);
		   this.message = msg;
		   this.name = "NotFoundException";
		   this.code = 404;
 		   this.fields.push("code");
		}
		NotFoundException.prototype = Object.create(BaseException.prototype);

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
		var gohan_handler = {};
		var gohan_caller = "";
		function gohan_add_dots(str, lim){
  		  if(str.length > lim){
		    str = str.substring(0,lim) + "...";
		  }
		  return str;
		}
		function gohan_register_handler(event_type, func){
		  if(_.isUndefined(gohan_handler[event_type])){
		    gohan_handler[event_type] = [];
		  }
		  var handlerUUID = gohan_uuid();
		  gohan_handler[event_type].push({fn:func,uuid:handlerUUID})
		  gohan_log_debug("REG: id=" + handlerUUID + ", type=" + event_type.toString() +
				", index=" + (gohan_handler[event_type].length - 1).toString())
		}
		function gohan_handle_event(event_type, context){
		  if(_.isUndefined(gohan_handler[event_type])){
		    return;
		  }
		  gohan_caller = gohan_uuid();
		  for (var i = 0; i < gohan_handler[event_type].length; ++i) {
		    try {
		      var old_module = gohan_log_module_push(event_type);
		      var handlerUUID = gohan_handler[event_type][i].uuid;
		      gohan_log_debug("BEGIN: id=" + handlerUUID + ", type=" + event_type.toString())
		      var timeStart = new Date().getTime();
		      gohan_handler[event_type][i].fn(context);
		      var timeStop = new Date().getTime();
		      gohan_log_debug("END: id=" + handlerUUID + ", type=" + event_type.toString() + ", time=" + (timeStop - timeStart) + " [ms]")
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
		    } finally {
		      gohan_log_module_restore(old_module);
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

func requireFromOtto(moduleName string, vm *otto.Otto) (otto.Value, error) {
	log.Debug(fmt.Sprintf("Loading module %s from otto", moduleName))
	rawModule, errRequire := RequireModule(moduleName)
	if errRequire != nil {
		return otto.UndefinedValue(), errRequire
	}

	module, errConvert := vm.ToValue(rawModule)
	if errConvert != nil {
		return otto.UndefinedValue(), errConvert
	}

	return module, nil
}

func requireFromMotto(moduleName string, vm *motto.Motto) (otto.Value, error) {
	log.Debug(fmt.Sprintf("Loading module %s from motto", moduleName))
	v, err := vm.Require(moduleName, "")
	if err != nil {
		log.Error("Cannot load module %s in Motto, err:%s", moduleName, err.Error())
	}
	return v, err
}

func require(moduleName string, vm *motto.Motto) (otto.Value, error) {
	value, err := requireFromMotto(moduleName, vm) // NPM
	if err != nil {
		log.Error(fmt.Sprintf("Loading module %s from motto failed: %s, trying to load from otto",
			moduleName, err))
		value, err = requireFromOtto(moduleName, vm.Otto) // Go extensions
	}

	return value, err
}

func loadNPMModules() {
	config := util.GetConfig()
	npmPath := config.GetString("extension/npm_path", ".")
	files, _ := ioutil.ReadDir(npmPath + "/node_modules/")
	for _, f := range files {
		if f.IsDir() && !strings.HasPrefix(f.Name(), ".") {
			module, err := motto.FindFileModule(f.Name(), npmPath, nil)
			if err != nil {
				log.Error("Finding module failed %s in %s", err, f.Name())
				break
			}

			var entryPoint string
			entryPointCandidates := []string{module, module + ".js", module + "/index.js"}

			for _, candidate := range entryPointCandidates {
				if candidateFile, err := os.Stat(candidate); err == nil && !candidateFile.IsDir() {
					entryPoint = candidate
					break
				}
			}

			if entryPoint == "" {
				log.Error("Cannot find entry point of %s module", module)
				break
			}

			loader := motto.CreateLoaderFromFile(entryPoint)
			motto.AddModule(f.Name(), loader)
		}
	}
}

func registerBuiltinModules(vm *motto.Motto) {
	vm.AddModule("fs", fsModule)
	vm.AddModule("vm", vmModule)
	vm.AddModule("crypto", cryptoModule)

	vm.AddModule("os", emptyModule)
	vm.AddModule("assert", emptyModule)
	vm.AddModule("glob", emptyModule)
}

func emptyModule(vm *motto.Motto) (otto.Value, error) {
	module, _ := vm.Object(`({})`)
	return vm.ToValue(module)
}
