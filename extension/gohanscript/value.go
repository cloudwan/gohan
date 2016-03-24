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
	"github.com/flosch/pongo2"
	"strconv"
	"strings"
)

//Value represents gohan script value interface
type Value interface {
	Value(context *Context) interface{}
}

//Nil reprensents nil
type Nil struct {
}

//Value returns nil
func (i *Nil) Value(context *Context) interface{} {
	return nil
}

//Ident reprensents identifire in gohan script
type Ident struct {
	keys []string
}

//Value returns correcponding value from context
func (i *Ident) Value(context *Context) interface{} {
	value, _ := GetByKeys(i.keys, context.data)
	return value
}

//NewIdent makes new identifier
func NewIdent(key string) *Ident {
	keys := KeyToList(key[1:]) //omit $
	return &Ident{
		keys: keys,
	}
}

//Constant represents donburi consntant.
type Constant struct {
	value interface{}
}

//Value returns constant value
func (c *Constant) Value(context *Context) interface{} {
	return c.value
}

//Template reprensents a value includes templates
type Template struct {
	templates map[string]*pongo2.Template
	param     interface{}
}

//NewTemplate makes a tempalte value
func NewTemplate(param interface{}) *Template {
	t := &Template{
		param:     param,
		templates: map[string]*pongo2.Template{}}
	CacheTemplate(t.param, t.templates)
	return t
}

//Value applies new template value
func (t *Template) Value(context *Context) interface{} {
	if context == nil {
		return t.param
	}
	return ApplyTemplate(t, t.param, context)
}

//NewValue makes gohan script value from an yaml data
func NewValue(a interface{}) Value {
	if a == nil {
		return nil
	}
	if arg, ok := a.(string); ok {
		if strings.HasPrefix(arg, "$") {
			return NewIdent(arg)
		}
	}
	if ContainsTemplate(a) {
		return NewTemplate(a)
	}
	return &Constant{
		value: a,
	}
}

//ApplyTemplate apply template code for input params.
func ApplyTemplate(template *Template, param interface{}, context *Context) (result interface{}) {
	switch p := param.(type) {
	case string:
		if strings.HasPrefix(p, "$") {
			result, _ := GetByKeys(KeyToList(p[1:]), context.data)
			return result
		}
		tpl := template.templates[p]
		if tpl == nil {
			return param
		}
		result, err := tpl.Execute(context.data)
		if err != nil {
			return param
		}
		i, err := strconv.Atoi(result)
		if err == nil {
			return i
		}
		return result

	case map[string]interface{}:
		result := map[string]interface{}{}
		for key, value := range p {
			result[ApplyTemplate(template, key, context).(string)] = ApplyTemplate(template, value, context)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(p))
		for index, value := range p {
			result[index] = ApplyTemplate(template, value, context)
		}
		return result
	}
	return param
}

//CacheTemplate caches template to the statement
func CacheTemplate(arg interface{}, templates map[string]*pongo2.Template) error {
	switch a := arg.(type) {
	case string:
		if !strings.HasPrefix(a, "$") {
			tpl, err := pongo2.FromString(a)
			if err != nil {
				return err
			}
			templates[a] = tpl
		}
	case []interface{}:
		for _, item := range a {
			err := CacheTemplate(item, templates)
			if err != nil {
				return err
			}
		}
	case map[string]interface{}:
		for key, item := range a {
			err := CacheTemplate(key, templates)
			if err != nil {
				return err
			}
			err = CacheTemplate(item, templates)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//ContainsTemplate checks an object contains template or not
func ContainsTemplate(arg interface{}) bool {
	switch a := arg.(type) {
	case string:
		return strings.HasPrefix(a, "$") || strings.Contains(a, "{{")
	case []interface{}:
		for _, item := range a {
			if ContainsTemplate(item) {
				return true
			}
		}
	case map[string]interface{}:
		for key, item := range a {
			if ContainsTemplate(key) {
				return true
			}
			if ContainsTemplate(item) {
				return true
			}
		}
	}
	return false
}
