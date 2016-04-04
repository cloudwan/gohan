// Copyright (C) 2016  Juniper Networks, Inc.
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

package gohanscript

import "fmt"

//VM is a struct for Gohan script.
type VM struct {
	Context      *Context
	debug        bool
	debugReturn  bool
	timeoutError error
	StopChan     chan func()
	funcs        []func(*Context) (interface{}, error)
}

func (vm *VM) String() string {
	return "[Gohan VM]"
}

//NewVM creates a VM
func NewVM() *VM {
	vm := &VM{
		timeoutError: fmt.Errorf("exceed timeout for extention execution"),
		StopChan:     make(chan func(), 1),
		funcs:        []func(*Context) (interface{}, error){},
	}
	vm.Context = NewContext(vm)
	return vm
}

//Clone clones a VM
func (vm *VM) Clone() *VM {
	newVM := NewVM()
	newVM.funcs = vm.funcs
	return newVM
}

//RunFile load data from file and execute it
func (vm *VM) RunFile(file string) (interface{}, error) {
	code, err := LoadYAMLFile(file)
	if err != nil {
		return nil, err
	}
	main, err := NewStmt(file, code)
	if err != nil {
		return nil, err
	}
	runner, err := main.Func()
	if err != nil {
		return nil, err
	}
	return runner(vm.Context)
}

//LoadFile loads gohan script code from file and make func
func (vm *VM) LoadFile(file string) error {
	code, err := LoadYAMLFile(file)
	if err != nil {
		return err
	}
	main, err := NewStmt(file, code)
	if err != nil {
		return err
	}
	fun, err := main.Func()
	if err != nil {
		return err
	}
	vm.funcs = append(vm.funcs, fun)
	return nil
}

//LoadString loads gohan script code from string and make func
func (vm *VM) LoadString(fileName, yamlString string) error {
	code := LoadYAML([]byte(yamlString))
	main, err := NewStmt(fileName, code)
	if err != nil {
		return err
	}
	fun, err := main.Func()
	if err != nil {
		return err
	}
	vm.funcs = append(vm.funcs, fun)
	return nil
}

//Run executes loaded funcs
func (vm *VM) Run(data map[string]interface{}) error {
	for _, fun := range vm.funcs {
		context := NewContext(vm)
		context.SetMap(data)
		_, err := fun(context)
		if err != nil {
			return err
		}
	}
	return nil
}
