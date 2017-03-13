// Copyright (C) 2017 NTT Innovation Institute, Inc.
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
	"github.com/xyproto/otto"
)

func init() {
	gohanGlobalInit := func(env *Environment) {
		vm := env.VM

		builtins := map[string]interface{}{
			"gohan_global":         gohanGlobalBuiltin("gohan_global", env.GohanGlobal),
			"gohan_process_global": gohanGlobalBuiltin("gohan_process_global", GohanProcessGlobal),
		}
		for name, object := range builtins {
			vm.Set(name, object)
		}
	}
	RegisterInit(gohanGlobalInit)
}

func gohanGlobalBuiltin(name string, gohanGlobal func(name string) map[string]interface{}) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		VerifyCallArguments(&call, "gohan_global", 1)

		name, err := GetString(call.Argument(0))
		if err != nil {
			log.Error("%s failed with error: %s", name, err.Error())
		}
		result := gohanGlobal(name)

		value, _ := call.Otto.ToValue(result)
		return value
	}
}

// GohanGlobal returns an object global for the Core
// this environment was created for -
// changes to this object will be seen in other such environments
// (but not in environments for other Core's,
//  e.g. not for other test suites)
// Example:
//   (test_suite1.env1)     var a = gohan_global("a");
//   (test_suite1.env2)     var a = gohan_global("a");
//   (other_test_suite.env) var a = gohan_global("a");
//   (test_suite1.env1)     a.test = 2;
//   (test_suite1.env2)     a.test; // => 2
//   (other_test_suite.env) a.test; // => undefined
func (env *Environment) GohanGlobal(name string) map[string]interface{} {
	return env.globalStore.Get(name)
}

var globalObjects = NewGlobalStore()

// GohanProcessGlobal returns an object global for the whole Gohan process -
// changes to this object will be seen in all environments.
// Example:
//   (test_suite1.env1)     var a = gohan_process_global("a");
//   (test_suite1.env2)     var a = gohan_process_global("a");
//   (other_test_suite.env) var a = gohan_process_global("a");
//   (test_suite1.env1)     a.test = 2;
//   (test_suite1.env2)     a.test; // => 2
//   (other_test_suite.env) a.test; // => 2
func GohanProcessGlobal(name string) map[string]interface{} {
	return globalObjects.Get(name)
}
