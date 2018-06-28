// Copyright 2014 dong<ddliuhb@gmail.com>.
// Licensed under the MIT license.
// 
// Motto - Modular Javascript environment.
package motto

import (
    "github.com/robertkrimen/otto"
    "path/filepath"
    // "fmt"
)

// Globally registered modules
var globalModules map[string]ModuleLoader = make(map[string]ModuleLoader)

// Globally registered paths (paths to search for modules)
var globalPaths []string

// The modular vm environment
type Motto struct {
    // Motto is based on otto
    *otto.Otto

    // Modules that registered for current vm
    modules map[string]ModuleLoader

    // Location to search for modules
    paths []string

    // Onece a module is required by vm, the exported value is cached for further
    // use.
    moduleCache map[string]otto.Value
}

// Run a module or file
func (this *Motto) Run(name string) (otto.Value, error) {
    if ok, _ := isFile(name); ok {
        name, _ = filepath.Abs(name)
    }

    return this.Require(name, ".")
}

// Require a module with cache
func (this *Motto) Require(id, pwd string) (otto.Value, error) {
    if cache, ok := this.moduleCache[id]; ok {
        return cache, nil
    }

    loader, ok := this.modules[id]
    if !ok {
        loader, ok = globalModules[id]
    }

    if loader != nil {
        value, err := loader(this)
        if err != nil {
            return otto.UndefinedValue(), err
        }

        this.moduleCache[id] = value
        return value, nil
    }

    filename, err := FindFileModule(id, pwd, append(this.paths, globalPaths...))
    if err != nil {
        return otto.UndefinedValue(), err
    }

    // resove id
    id = filename

    if cache, ok := this.moduleCache[id]; ok {
        return cache, nil
    }

    v, err := CreateLoaderFromFile(id)(this)

    if err != nil {
        return otto.UndefinedValue(), err
    }

    // cache
    this.moduleCache[id] = v

    return v, nil
}

// Register a new module to current vm.
func (this *Motto) AddModule(id string, loader ModuleLoader) {
    this.modules[id] = loader
}


// Add paths to search for modules.
func (this *Motto) AddPath(paths ...string) {
    this.paths = append(this.paths, paths...)
}

// Register a global module
func AddModule(id string, m ModuleLoader) {
    globalModules[id] = m
}

// Register global path.
func AddPath(paths ...string) {
    globalPaths = append(globalPaths, paths...)
}

// Run module by name in the motto module environment.
func Run(name string) (*Motto, otto.Value, error) {
    vm := New()
    v, err := vm.Run(name)

    return vm, v, err
}

// Create a motto vm instance.
func New() *Motto {
    return &Motto{otto.New(), make(map[string]ModuleLoader), nil, make(map[string]otto.Value)}
}