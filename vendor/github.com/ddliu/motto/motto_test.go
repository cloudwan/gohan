// Copyright 2014 dong<ddliuhb@gmail.com>.
// Licensed under the MIT license.
// 
// Motto - Modular Javascript environment.
package motto

import (
    "testing"
    "github.com/robertkrimen/otto"
    "io/ioutil"
)

func TestModule(t *testing.T) {
    _, v, err := Run("tests/index.js")
    if err != nil {
        t.Error(err)
    }

    i, _ := v.ToString()
    if i != "rat" {
        t.Error("testing result: ", i , "!=", "rat")
    }
}

func TestNpmModule(t *testing.T) {
    _, v, err := Run("tests/npm/index.js")

    if err != nil {
        t.Error(err)
    }

    i, _ := v.ToInteger()

    if i != 1 {
        t.Error("npm test failed: ", i , "!=", 1)
    }
}

func TestCoreModule(t *testing.T) {
    vm := New()
    vm.AddModule("fs", fsModuleLoader)

    v, err := vm.Run("tests/core_module_test.js")
    if err != nil {
        t.Error(err)
    }

    s, _ := v.ToString()
    if s != "cat" {
        t.Error("core module test failed: ", s, "!=", "cat")
    }
}

func fsModuleLoader(vm *Motto) (otto.Value, error) {
    fs, _ := vm.Object(`({})`)
    fs.Set("readFileSync", func(call otto.FunctionCall) otto.Value {
        filename, _ := call.Argument(0).ToString()
        bytes, err := ioutil.ReadFile(filename)
        if err != nil {
            return otto.UndefinedValue()
        }

        v, _ := call.Otto.ToValue(string(bytes))
        return v
    })

    return vm.ToValue(fs)
}