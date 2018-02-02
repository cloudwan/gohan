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
	//TODO(nati) upstream local change
	"github.com/cloudwan/gohan/util"
	"github.com/nati/yaml"
)

//Error represent error for gohanscript
type Error struct {
	error
}

//Stmt represents gohan script statement.
//Stmt is responseible to generate go function from
//YAML definitions.
type Stmt struct {
	Name      string
	When      *MiniGo
	Until     *MiniGo
	File      string
	Dir       string
	Line      int
	Column    int
	Vars      map[string]Value
	Rescue    []*Stmt
	Always    []*Stmt
	ElseStmt  []*Stmt
	funcName  string
	Retry     int
	Delay     int
	Worker    int
	Args      map[string]Value
	WithItems Value
	LoopVar   string
	WithDict  Value
	Register  string
	RawData   map[string]interface{}
	RawNode   map[string]*yaml.Node
}

//MakeStmts parses multiple statesmantes
func MakeStmts(FileName string, node *yaml.Node) ([]*Stmt, error) {
	if node == nil {
		return nil, nil
	}
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("Expected list: %s line: %d", FileName, node.Line)
	}
	result := []*Stmt{}
	for _, n := range node.Children {
		stmt, err := NewStmt(FileName, n)
		if err != nil {
			return nil, err
		}
		result = append(result, stmt)
	}
	return result, nil
}

//Errorf makes err with stmt information
func (stmt *Stmt) Errorf(msg string, args ...interface{}) error {
	return &Error{fmt.Errorf("%s:%d error %s", stmt.File, stmt.Line, fmt.Sprintf(msg, args...))}
}

//Error makes err with stmt information
func (stmt *Stmt) Error(err error) error {
	if _, ok := err.(*Error); ok {
		return err
	}
	return &Error{fmt.Errorf("%s:%d: error %s", stmt.File, stmt.Line, err)}
}

//NewStmt makes gohan statement from yaml node
func NewStmt(FileName string, node *yaml.Node) (stmt *Stmt, err error) {
	if node == nil {
		return nil, nil
	}
	stmt = &Stmt{}
	stmt.RawNode = MappingNodeToMap(node)
	stmt.Line = node.Line + 1
	stmt.Column = node.Column
	var rawData interface{}
	yaml.UnmarshalNode(node, &rawData)
	stmt.RawData = util.MaybeMap(convertMapformat(rawData))
	stmt.Name = util.MaybeString(stmt.RawData["name"])
	stmt.File, _ = filepath.Abs(FileName)
	stmt.Dir = filepath.Dir(stmt.File)
	stmt.Always, err = MakeStmts(FileName, stmt.RawNode["always"])
	if err != nil {
		return nil, stmt.Error(err)
	}
	stmt.Rescue, err = MakeStmts(FileName, stmt.RawNode["rescue"])
	if err != nil {
		return nil, stmt.Error(err)
	}
	stmt.ElseStmt, err = MakeStmts(FileName, stmt.RawNode["else"])
	if err != nil {
		return nil, stmt.Error(err)
	}
	stmt.Retry = util.MaybeInt(stmt.RawData["retries"])
	stmt.Worker = util.MaybeInt(stmt.RawData["worker"])
	stmt.LoopVar = util.MaybeString(stmt.RawData["loop_var"])
	if stmt.LoopVar == "" {
		stmt.LoopVar = "item"
	}
	if stmt.Retry == 0 {
		stmt.Retry = 1
	}
	stmt.Delay = util.MaybeInt(stmt.RawData["delay"])
	stmt.WithItems, err = NewValue(stmt.RawData["with_items"])
	if err != nil {
		return nil, stmt.Error(err)
	}
	stmt.WithDict, err = NewValue(stmt.RawData["with_dict"])
	if err != nil {
		return nil, stmt.Error(err)
	}
	if stmt.RawData["when"] != nil {
		stmt.When, err = CompileExpr(util.MaybeString(stmt.RawData["when"]))
		if err != nil {
			return nil, stmt.Error(err)
		}
	}
	if stmt.RawData["until"] != nil {
		stmt.Until, err = CompileExpr(util.MaybeString(stmt.RawData["until"]))
		if err != nil {
			return nil, stmt.Error(err)
		}
	}
	stmt.Register = util.MaybeString(stmt.RawData["register"])
	stmt.Vars, err = MapToValue(util.MaybeMap(stmt.RawData["vars"]))
	if err != nil {
		return nil, stmt.Error(err)
	}
	return stmt, nil
}

func (stmt *Stmt) parser() (parser StmtParser, err error) {
	for key, value := range stmt.RawData {
		parser = GetStmtParser(key)
		if parser != nil {
			stmt.funcName = key
			stmt.Args, err = MapToValue(parseCode(key, value))
			return
		}
	}
	return nil, nil
}

//HasArgs checks if statement has functions
func (stmt *Stmt) HasArgs(keys ...string) error {
	for _, key := range keys {
		_, ok := stmt.Args[key]
		if !ok {
			return fmt.Errorf("missing argument %s", key)
		}
	}
	return nil
}

//Arg get augument data using key and context
func (stmt *Stmt) Arg(key string, context *Context) interface{} {
	arg := stmt.Args[key]
	if arg == nil {
		return &Nil{}
	}
	return arg.Value(context)
}

//Func generates function from stmt
func (stmt *Stmt) Func() (func(context *Context) (interface{}, error), error) {
	stmtParser, err := stmt.parser()
	if err != nil {
		return nil, stmt.Error(err)
	}
	if stmtParser == nil {
		yamlCode, _ := yaml.Marshal(&stmt.RawData)
		return nil, stmt.Errorf("Undefined function, %s", yamlCode)
	}
	f, err := stmtParser(stmt)
	if err != nil {
		return nil, stmt.Error(err)
	}
	return applyWrappers(stmt, f)
}

//StmtsToFunc creates list of func from stmts
func StmtsToFunc(funcName string, stmts []*Stmt) (func(*Context) (interface{}, error), error) {
	runners := []func(*Context) (interface{}, error){}
	if stmts == nil {
		return nil, nil
	}
	if len(stmts) == 0 {
		return nil, nil
	}
	for _, s := range stmts {
		runner, err := s.Func()
		if err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	return func(context *Context) (value interface{}, err error) {
		vm := context.VM
		for _, f := range runners {
			select {
			case f := <-vm.StopChan:
				f()
			default:
				if value, err = f(context); err != nil {
					if _, ok := err.(breakCode); ok {
						return value, nil
					}
					return nil, err
				}
			}
		}
		return
	}, nil
}

//MapToValue converts interface map to value map
func MapToValue(a map[string]interface{}) (result map[string]Value, err error) {
	result = map[string]Value{}
	if a == nil {
		return
	}
	for key, value := range a {
		result[key], err = NewValue(value)
	}
	return
}

//MappingNodeToMap convert yaml node to map
func MappingNodeToMap(node *yaml.Node) (result map[string]*yaml.Node) {
	result = map[string]*yaml.Node{}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Children); i += 2 {
		result[node.Children[i].Value] = node.Children[i+1]
	}
	return
}
