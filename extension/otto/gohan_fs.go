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
	"io/ioutil"
	"os"

	"github.com/ddliu/motto"
	"github.com/robertkrimen/otto"
)

func fsModule(vm *motto.Motto) (otto.Value, error) {
	fs, _ := vm.Object(`({})`)

	// Modified from motto documentation
	fs.Set("readFileSync", func(call otto.FunctionCall) otto.Value {
		filename, err := GetString(call.Argument(0))
		ThrowWithMessageIfHappened(&call, err,
			"Filename: %v", err)

		bytes, err := ioutil.ReadFile(filename)
		ThrowWithMessageIfHappened(&call, err,
			"Failed to read file '%s': %v", filename, err)

		// Note: according to the specification,
		// we should return a Buffer instead
		// if the 'encoding' argument isn't provided.
		// But we always return a string here.
		v, _ := call.Otto.ToValue(string(bytes))
		return v
	})

	fs.Set("existsSync", func(call otto.FunctionCall) otto.Value {
		filename, err := GetString(call.Argument(0))
		ThrowWithMessageIfHappened(&call, err,
			"Filename: %v", err)

		_, err = os.Stat(filename)

		v, _ := call.Otto.ToValue(err == nil)
		return v
	})

	fs.Set("writeFileSync", func(call otto.FunctionCall) otto.Value {
		filename, err := GetString(call.Argument(0))
		ThrowWithMessageIfHappened(&call, err,
			"Filename: %v", err)
		data, err := GetString(call.Argument(1))
		ThrowWithMessageIfHappened(&call, err,
			"File data: %v", err)

		err = ioutil.WriteFile(filename, []byte(data), 0666)
		ThrowWithMessageIfHappened(&call, err,
			"Failed to write file '%s': %v", filename, err)
		return otto.UndefinedValue()
	})

	fs.Set("readdirSync", func(call otto.FunctionCall) otto.Value {
		dirname, err := GetString(call.Argument(0))
		ThrowIfHappened(&call, err)

		files, err := ioutil.ReadDir(dirname)
		ThrowIfHappened(&call, err)

		var result []string
		for _, file := range files {
			result = append(result, file.Name())
		}

		v, _ := call.Otto.ToValue(result)
		return v
	})

	fs.Set("mkdirSync", func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) < 2 {
			defaultMode, _ := otto.ToValue(0777)
			call.ArgumentList = append(call.ArgumentList, defaultMode)
		}
		VerifyCallArguments(&call, "mkdirSync", 2)
		dirname, err := GetString(call.Argument(0))
		ThrowWithMessageIfHappened(&call, err,
			"Directory name: %v", err)
		mode, err := GetInt64(call.Argument(1))
		ThrowWithMessageIfHappened(&call, err,
			"Permission mode: %v", err)

		// TODO properly handle "file exists" error instead
		err = os.MkdirAll(dirname, os.FileMode(int32(mode)))
		ThrowWithMessageIfHappened(&call, err,
			"Failed to create directory '%s': %v", dirname, err)

		return otto.UndefinedValue()
	})

	return vm.ToValue(fs)
}
