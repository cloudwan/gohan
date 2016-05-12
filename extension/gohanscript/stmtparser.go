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
	"fmt"
	"path/filepath"
	"time"

	"github.com/cloudwan/gohan/util"
	"github.com/kr/pretty"
	"github.com/nati/yaml"
	"reflect"
	"strings"
)

//StmtParser converts gohan script statement for golang function.
//You can register your parser using RegisterStmtParser call, so that
// you can have new gohan script function implemented by go
type StmtParser func(stmt *Stmt) (func(*Context) (interface{}, error), error)

//Parsers converts Stmts for functions
var parsers map[string]StmtParser

func init() {
	parsers = map[string]StmtParser{}
	RegisterStmtParser("blocks", blocks)
	RegisterStmtParser("background", background)
	RegisterStmtParser("tasks", blocks)
	RegisterStmtParser("define", define)
	RegisterStmtParser("vars", vars)
	RegisterStmtParser("return", returnCode)
	RegisterStmtParser("fail", fail)
	RegisterStmtParser("include", include)
	RegisterStmtParser("debugger", debugger)
	RegisterStmtParser("debug", debug)
	RegisterStmtParser("log_warn", logWarn)
	RegisterStmtParser("log_info", logInfo)
	RegisterStmtParser("log_error", logError)
	RegisterStmtParser("test_suite", testSuite)
	RegisterStmtParser("assert", assert)
	RegisterStmtParser("sleep", sleep)
	RegisterStmtParser("yaml", yamlTemplate)
	RegisterStmtParser("minigo", minigo)
	RegisterStmtParser("minigo_func", minigoFunc)
}

//RegisterStmtParser registers new parser per function
func RegisterStmtParser(funcName string, callback StmtParser) {
	parsers[funcName] = callback
}

//GetStmtParser returns parser for specific per function
func GetStmtParser(funcName string) StmtParser {
	return parsers[funcName]
}

type breakCode struct {
}

func (b breakCode) Error() string {
	return "break"
}

func blocks(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	stmts, err := MakeStmts(stmt.File, stmt.RawNode[stmt.funcName])
	if err != nil {
		return nil, err
	}
	return StmtsToFunc(stmt.funcName, stmts)
}

func include(stmt *Stmt) (func(*Context) (interface{}, error), error) {

	source := util.MaybeString(stmt.RawData["include"])
	if !filepath.IsAbs(source) {
		currentSource := stmt.File
		source = filepath.Clean(filepath.Join(filepath.Dir(currentSource), source))
	}
	importedCode, err := LoadYAMLFile(source)
	if err != nil {
		return nil, stmt.Errorf("yaml parse error: %s", err)
	}
	stmt.Args = map[string]Value{}
	for key, value := range stmt.RawData {
		stmt.Args[key] = NewValue(value)
	}
	stmt.File = source
	stmts, err := MakeStmts(stmt.File, importedCode)
	if err != nil {
		return nil, stmt.Errorf("import code parse error: %s", err)
	}
	stmt.Vars = MapToValue(util.MaybeMap(stmt.RawData["vars"]))
	return StmtsToFunc(stmt.funcName, stmts)
}

func background(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	stmts, err := MakeStmts(stmt.File, stmt.RawNode["background"])
	if err != nil {
		return nil, stmt.Errorf("background code error: %s", err)
	}
	f, err := StmtsToFunc("background", stmts)
	if err != nil {
		return nil, err
	}
	return func(context *Context) (interface{}, error) {
		newContext := context.Extend(map[string]interface{}{})
		go func() {
			defer func() {
				if err := recover(); err != nil {
					log.Error(fmt.Sprintf("panic: %s", err))
				}
			}()
			f(newContext)
			newContext.VM.Stop()
		}()
		return nil, nil
	}, nil
}

func vars(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		return nil, nil
	}, nil
}

func returnCode(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	retValue := NewValue(stmt.RawData["return"])
	return func(context *Context) (interface{}, error) {
		val := retValue.Value(context)
		return val, breakCode{}
	}, nil
}

func fail(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		return nil, fmt.Errorf("%v", stmt.Arg("msg", context))
	}, nil
}

func define(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	var err error
	funcName := util.MaybeString(stmt.Args["name"].Value(nil))
	defineNode := MappingNodeToMap(stmt.RawNode["define"])
	funcArgs := util.MaybeMap(stmt.Args["args"].Value(nil))
	stmts, err := MakeStmts(stmt.File, defineNode["body"])
	if err != nil {
		return nil, stmt.Errorf("error in define body: %s", err)
	}
	var body func(*Context) (interface{}, error)
	RegisterStmtParser(
		funcName,
		func(aStmt *Stmt) (func(*Context) (interface{}, error), error) {
			for key := range funcArgs {
				_, ok := aStmt.Args[key]
				if !ok {
					return nil, fmt.Errorf("missing augument %s", key)
				}
			}
			return func(context *Context) (value interface{}, err error) {
				vm := context.VM
				newContext := NewContext(vm)
				for key := range funcArgs {
					newContext.Set(key, aStmt.Arg(key, context))
				}
				value, err = body(newContext)

				if vm.debugReturn {
					vm.debugReturn = false
					vm.debug = true
				}
				return
			}, nil
		})
	body, err = StmtsToFunc("funcName", stmts)
	if err != nil {
		return nil, err
	}
	return func(context *Context) (interface{}, error) {
		return nil, nil
	}, nil
}

func debugger(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		context.VM.debug = true
		return nil, nil
	}, nil
}

func debug(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		if msg, ok := stmt.Args["msg"]; ok {
			log.Debug("%s:%d %s",
				filepath.Base(stmt.File),
				stmt.Line,
				msg.Value(context))
		} else if id, ok := stmt.Args["var"]; ok {
			log.Debug("%s:%d %v",
				filepath.Base(stmt.File),
				stmt.Line,
				id.Value(context))
		} else {
			log.Debug("%s:%d Dump vars",
				filepath.Base(stmt.File),
				stmt.Line)
			for key, value := range context.data {
				log.Debug("    %s: %v", key, value)
			}
		}
		return nil, nil
	}, nil
}

func logWarn(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		log.Warning("%s:%d %s",
			stmt.File,
			stmt.Line,
			stmt.Arg("msg", context))
		return nil, nil
	}, nil
}

func logInfo(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		log.Info("%s:%d %s",
			stmt.File,
			stmt.Line,
			stmt.Arg("msg", context))
		return nil, nil
	}, nil
}

func logError(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		log.Error(fmt.Sprintf("%s:%d %s",
			stmt.File,
			stmt.Line,
			stmt.Arg("msg", context)))
		return nil, nil
	}, nil
}

func yamlTemplate(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		if y, ok := stmt.Args["yaml"]; ok {
			yamlString := y.Value(context)
			var data interface{}
			err := yaml.Unmarshal([]byte(yamlString.(string)), &data)
			return data, err
		}
		return nil, nil
	}, nil
}

func sleep(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		duration := time.Duration(util.MaybeInt(stmt.Arg("sleep", context))) * time.Millisecond
		timer := time.NewTimer(duration)
		for {
			select {
			case <-timer.C:
				return nil, nil
			case f := <-context.VM.StopChan:
				timer.Stop()
				f()
				return nil, nil
			}
		}
	}, nil
}

func testSuite(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	var beforeEach func(*Context) (interface{}, error)
	testNode := MappingNodeToMap(stmt.RawNode["test_suite"])
	tests := testNode["tests"]
	if testNode["before_each"] != nil {
		beforeStmts, err := MakeStmts(stmt.File, testNode["before_each"])
		if err != nil {
			return nil, err
		}
		beforeEach, err = StmtsToFunc("before_each", beforeStmts)
		if err != nil {
			return nil, err
		}
	}
	return func(context *Context) (interface{}, error) {
		var successCount, failedCount int
		for _, test := range tests.Children {
			testMap := MappingNodeToMap(test)
			name := testMap["name"].Value
			testContext := context.Extend(nil)
			if beforeEach != nil {
				_, err := beforeEach(testContext)
				if err != nil {
					log.Error(fmt.Sprintf("%s ... failed", name))
					log.Error(err.Error())
					failedCount++
					continue
				}
			}
			testsStmt, err := MakeStmts(stmt.File, testMap["test"])
			if err != nil {
				log.Error(fmt.Sprintf("%s ... failed", name))
				log.Error(err.Error())
				failedCount++
				continue
			}
			testRunner, err := StmtsToFunc("test", testsStmt)
			if err != nil {
				log.Error(fmt.Sprintf("%s ... failed", name))
				log.Error(err.Error())
				failedCount++
				continue
			}
			_, err = testRunner(testContext)
			if err != nil {
				log.Error(fmt.Sprintf("%s ... failed", name))
				log.Error(err.Error())
				failedCount++
			} else {
				log.Debug("%s ... ok", name)
				successCount++
			}
		}
		return map[string]interface{}{
			"count":   len(tests.Children),
			"success": successCount,
			"failed":  failedCount,
		}, nil
	}, nil
}

func assert(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (interface{}, error) {
		expect := stmt.Arg("expect", context)
		actual := stmt.Arg("actual", context)
		if !reflect.DeepEqual(expect, actual) {
			return nil, fmt.Errorf("%s:%d error expected: '%v', actual: '%v' \n%v",
				stmt.File,
				stmt.Line,
				expect, actual,
				strings.Join(pretty.Diff(expect, actual), "\n"),
			)
		}
		return nil, nil
	}, nil
}

func minigo(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	code := stmt.Args["minigo"].Value(nil)
	minigo, err := CompileGoStmt(code.(string))
	if err != nil {
		return nil, err
	}
	return func(context *Context) (interface{}, error) {
		return minigo.Run(context)
	}, nil
}

func minigoFunc(stmt *Stmt) (func(*Context) (interface{}, error), error) {
	code := stmt.Args["minigo_func"].Value(nil)
	err := CompileFile(code.(string))
	if err != nil {
		return nil, err
	}
	return func(context *Context) (interface{}, error) {
		return nil, nil
	}, nil
}
