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

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/cloudwan/gohan/util"
)

// Context is execution context in gohan script.
// Variables are stored in this context object.
type Context struct {
	VM   *VM
	data map[string]interface{}
}

//NewContext makes data context
func NewContext(vm *VM) *Context {
	return &Context{
		VM: vm,
		data: map[string]interface{}{
			"true":  true,
			"false": false,
			"nil":   nil,
			"null":  nil,
		},
	}
}

// SetMap set entire data
func (context *Context) SetMap(data map[string]interface{}) {
	context.data = data
}

// Set set data in the context
func (context *Context) Set(key string, value interface{}) {
	context.data[key] = value
}

// SetByKeys set values in keys
func (context *Context) SetByKeys(key string, value interface{}) {
	keys := KeyToList(key)
	if len(keys) == 1 {
		context.data[key] = value
		return
	}

	data, err := GetByKeys(keys[:len(keys)-1], context.data)
	if err != nil {
		return
	}
	m, ok := data.(map[string]interface{})
	if ok {
		m[keys[len(keys)-1]] = value
	}

}

// Get returns data from key
func (context *Context) Get(key string) (interface{}, error) {
	v, ok := context.data[key]
	if !ok {
		if f, ok := miniGoFuncs[key]; ok {
			return f, nil
		}
		return nil, fmt.Errorf("key %s not found", key)
	}
	return v, nil
}

// MayGet returns data from key if it exists
func (context *Context) MayGet(key string) interface{} {
	v, _ := context.Get(key)
	return v
}

//KeyToList converts keys to string array
func KeyToList(keys string) []string {
	var buf bytes.Buffer
	result := []string{}
	for _, c := range keys {
		switch c {
		case '[', '.':
			result = append(result, buf.String())
			buf.Reset()
		case ']':
		default:
			buf.WriteRune(c)
		}
	}
	result = append(result, buf.String())
	return result
}

//GetByKeys get value from list of key
func GetByKeys(keys []string, value interface{}) (interface{}, error) {
	if keys == nil {
		return nil, fmt.Errorf("Key is nil")
	}
	if len(keys) < 1 {
		return value, nil
	}
	key := keys[0]
	switch v := value.(type) {
	case []interface{}:
		index, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("Irregular index")
		}
		if len(v) <= index {
			return nil, fmt.Errorf("Index out of range")
		}
		return GetByKeys(keys[1:], v[index])
	case map[string]interface{}:
		if result, ok := v[key]; ok {
			return GetByKeys(keys[1:], result)
		}
		return nil, fmt.Errorf("key not found")
	case map[interface{}]interface{}:
		if result, ok := v[key]; ok {
			return GetByKeys(keys[1:], result)
		}
		return nil, fmt.Errorf("key not found")
	}
	return nil, fmt.Errorf("key not found")
}

// Clear clears key
func (context *Context) Clear(key string) {
	delete(context.data, key)
}

// MaybeList resturns a List or nil
func (context *Context) MaybeList(key string) []interface{} {
	return util.MaybeList(context.MayGet(key))
}

// MaybeMap resturns a Map or nil
func (context *Context) MaybeMap(key string) map[string]interface{} {
	return util.MaybeMap(context.MayGet(key))
}

// MaybeString returns a string
func (context *Context) MaybeString(key string) string {
	return util.MaybeString(context.MayGet(key))
}

// MaybeInt returns a int
func (context *Context) MaybeInt(key string) int {
	return util.MaybeInt(context.MayGet(key))
}

// HasKey checks if we have key
func (context *Context) HasKey(key string) bool {
	_, ok := context.data[key]
	return ok
}

// Extend clones context and extend it
func (context *Context) Extend(values map[string]interface{}) *Context {
	newContext := NewContext(context.VM.Clone())
	for key, value := range context.data {
		newContext.Set(key, value)
	}
	for key, value := range values {
		newContext.Set(util.MaybeString(key), value)
	}
	return newContext
}
