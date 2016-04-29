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

//MiniGoValue reprensents minigo value
type MiniGoValue struct {
	minigo *MiniGo
}

//Value returns nil
func (i *MiniGoValue) Value(context *Context) interface{} {
	value, _ := i.minigo.Run(context)
	return value
}

//NewMiniGoValue makes a MiniGoValue value
func NewMiniGoValue(param interface{}) *MiniGoValue {
	code, ok := param.(string)
	if !ok {
		return nil
	}
	if code == "" {
		return nil
	}
	if code[0] != '$' {
		return nil
	}
	minigoCode := code[1:]
	if minigoCode[0] == '{' {
		minigoCode = minigoCode[1:]
	}
	if minigoCode[len(minigoCode)-1] == '}' {
		minigoCode = minigoCode[:len(minigoCode)-1]
	}
	minigo, err := CompileExpr(minigoCode)
	if err != nil {
		return nil
	}
	t := &MiniGoValue{
		minigo: minigo,
	}
	return t
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
	minigos   map[string]*MiniGo
}

//NewTemplate makes a template value
func NewTemplate(param interface{}) *Template {
	t := &Template{
		param:     param,
		templates: map[string]*pongo2.Template{},
		minigos:   map[string]*MiniGo{},
	}
	CacheTemplate(t.param, t.templates, t.minigos)
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
	miniGoValue := NewMiniGoValue(a)
	if miniGoValue != nil {
		return miniGoValue
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
		if p == "" {
			return nil
		}
		if p[0] == '$' {
			minigo := template.minigos[p]
			result, _ := minigo.Run(context)
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
func CacheTemplate(arg interface{}, templates map[string]*pongo2.Template, minigos map[string]*MiniGo) error {
	switch a := arg.(type) {
	case string:
		if a == "" {
			return nil
		}
		if a[0] == '$' {
			miniGoValue := NewMiniGoValue(a)
			minigos[a] = miniGoValue.minigo
		} else {
			tpl, err := pongo2.FromString(a)
			if err != nil {
				return err
			}
			templates[a] = tpl
		}
	case []interface{}:
		for _, item := range a {
			err := CacheTemplate(item, templates, minigos)
			if err != nil {
				return err
			}
		}
	case map[string]interface{}:
		for key, item := range a {
			err := CacheTemplate(key, templates, minigos)
			if err != nil {
				return err
			}
			err = CacheTemplate(item, templates, minigos)
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
