// Copyright 2014 dong<ddliuhb@gmail.com>.
// Licensed under the MIT license.
//
// Motto - Modular Javascript environment.
package motto

import (
	"errors"
	"github.com/xyproto/otto"
	"io/ioutil"
	"path/filepath"
)

// ModuleLoader is declared to load a module.
type ModuleLoader func(*Motto) (otto.Value, error)

// Create module loader from javascript source code.
//
// When the loader is called, the javascript source is executed in Motto.
//
// "pwd" indicates current working directory, which might be used to search for
// modules.
func CreateLoaderFromSource(source, pwd string) ModuleLoader {
	return func(vm *Motto) (otto.Value, error) {
		// Wraps the source to create a module environment
		source = "(function(module) {var require = module.require;var exports = module.exports;\n" + source + "\n})"

		// Provide the "require" method in the module scope.
		jsRequire := func(call otto.FunctionCall) otto.Value {
			jsModuleName := call.Argument(0).String()

			moduleValue, err := vm.Require(jsModuleName, pwd)
			if err != nil {
				jsException(vm, "Error", "motto: "+err.Error())
			}

			return moduleValue
		}

		jsModule, _ := vm.Object(`({exports: {}})`)
		jsModule.Set("require", jsRequire)
		jsExports, _ := jsModule.Get("exports")

		// Run the module source, with "jsModule" as the "module" variable, "jsExports" as "this"(Nodejs capable).
		moduleReturn, err := vm.Call(source, jsExports, jsModule)
		if err != nil {
			return otto.UndefinedValue(), err
		}

		var moduleValue otto.Value
		if !moduleReturn.IsUndefined() {
			moduleValue = moduleReturn
			jsModule.Set("exports", moduleValue)
		} else {
			moduleValue, _ = jsModule.Get("exports")
		}

		return moduleValue, nil
	}
}

// Create module loader from javascript file.
//
// Filename can be a javascript file or a json file.
func CreateLoaderFromFile(filename string) ModuleLoader {
	return func(vm *Motto) (otto.Value, error) {
		source, err := ioutil.ReadFile(filename)

		if err != nil {
			return otto.UndefinedValue(), err
		}

		// load json
		if filepath.Ext(filename) == ".json" {
			return vm.Call("JSON.parse", nil, string(source))
		}

		pwd := filepath.Dir(filename)

		return CreateLoaderFromSource(string(source), pwd)(vm)
	}
}

// Find a file module by name.
//
// If name starts with "." or "/", we search the module in the according locations
// (name and name.js and name.json).
//
// Otherwise we search the module in the "node_modules" sub-directory of "pwd" and
// "paths"
//
// It basicly follows the rules of Node.js module api: http://nodejs.org/api/modules.html
func FindFileModule(name, pwd string, paths []string) (string, error) {
	if len(name) == 0 {
		return "", errors.New("Empty module name")
	}

	var choices []string
	if name[0] == '.' || filepath.IsAbs(name) {
		if name[0] == '.' {
			name = filepath.Join(pwd, name)
		}

		choices = append(choices, name)
		ext := filepath.Ext(name)
		if ext != ".js" && ext != ".json" {
			choices = append(choices, name+".js", name+".json")
		}
	} else {
		if pwd != "" {
			choices = append(choices, filepath.Join(pwd, "node_modules", name))
		}

		for _, v := range paths {
			choices = append(choices, filepath.Join(v), name)
		}
	}

	for _, v := range choices {
		ok, err := isDir(v)
		if err != nil {
			return "", err
		}

		if ok {
			packageJsonFilename := filepath.Join(v, "package.json")
			ok, err := isFile(packageJsonFilename)
			if err != nil {
				return "", err
			}

			var entryPoint string
			if ok {
				entryPoint, err = parsePackageEntryPoint(packageJsonFilename)
				if err != nil {
					return "", err
				}
			} else {
				entryPoint = "./index.js"
			}

			return filepath.Abs(filepath.Join(v, entryPoint))
		}

		ok, err = isFile(v)
		if err != nil {
			return "", err
		}

		if ok {
			return filepath.Abs(v)
		}
	}

	return "", errors.New("Module not found: " + name)
}
