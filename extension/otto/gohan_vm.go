// Copyright (C) 2016 NTT Innovation Institute, Inc.
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
	"github.com/ddliu/motto"
	"github.com/robertkrimen/otto"
)

func vmModule(vm *motto.Motto) (otto.Value, error) {
	mod, _ := vm.Object(`({})`)

	emptyFunctions := []string{
		"createScript",
	}
	for _, funName := range emptyFunctions {
		mod.Set(funName, func(call otto.FunctionCall) otto.Value {
			return otto.UndefinedValue()
		})
	}

	mod.Set("runInThisContext", func(call otto.FunctionCall) otto.Value {
		VerifyCallArguments(&call, "runInThisContext", 2)
		source, err := GetString(call.Argument(0))
		ThrowWithMessageIfHappened(&call, err,
			"Source code: %v", err)
		filename, err := GetString(call.Argument(1))
		ThrowWithMessageIfHappened(&call, err,
			"Filename: %v", err)

		script, err := vm.Compile(filename, source)
		ThrowWithMessageIfHappened(&call, err,
			"Failed to compile %s err: %v", filename, err)
		result, err := vm.Otto.Run(script)
		ThrowIfHappened(&call, err)
		v, _ := vm.ToValue(result)
		return v
	})

	return vm.ToValue(mod)
}
