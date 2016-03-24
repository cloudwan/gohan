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
	WithDict  Value
	Register  string
	RawData   map[string]interface{}
	RawNode   map[string]*yaml.Node
}

//MakeStmts parses multiple statemantes
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
	return fmt.Errorf("[%s:%d] %s", stmt.File, stmt.Line, fmt.Sprintf(msg, args...))
}

//NewStmt makes gohan statement from yaml node
func NewStmt(FileName string, node *yaml.Node) (stmt *Stmt, err error) {
	if node == nil {
		return nil, nil
	}
	stmt = &Stmt{}
	stmt.RawNode = MappingNodeToMap(node)
	stmt.Line = node.Line
	stmt.Column = node.Column
	var rawData interface{}
	yaml.UnmarshalNode(node, &rawData)
	stmt.RawData = util.MaybeMap(convertMapformat(rawData))
	stmt.Name = util.MaybeString(stmt.RawData["name"])
	stmt.File = FileName
	stmt.Dir = filepath.Dir(stmt.File)
	stmt.Always, err = MakeStmts(FileName, stmt.RawNode["always"])
	if err != nil {
		return nil, stmt.Errorf("always path parse error: %s", err)
	}
	stmt.Rescue, err = MakeStmts(FileName, stmt.RawNode["rescue"])
	if err != nil {
		return nil, stmt.Errorf("rescue path parse error: %s", err)
	}
	stmt.ElseStmt, err = MakeStmts(FileName, stmt.RawNode["else"])
	if err != nil {
		return nil, stmt.Errorf("else path parse error: %s", err)
	}
	stmt.Retry = util.MaybeInt(stmt.RawData["retries"])
	stmt.Worker = util.MaybeInt(stmt.RawData["worker"])
	if stmt.Retry == 0 {
		stmt.Retry = 1
	}
	stmt.Delay = util.MaybeInt(stmt.RawData["delay"])
	stmt.WithItems = NewValue(stmt.RawData["with_items"])
	stmt.WithDict = NewValue(stmt.RawData["with_dict"])
	if stmt.RawData["when"] != nil {
		stmt.When, err = CompileExpr(util.MaybeString(stmt.RawData["when"]))
		if err != nil {
			return nil, stmt.Errorf("when code parse error: %s", err)
		}
	}
	if stmt.RawData["until"] != nil {
		stmt.Until, err = CompileExpr(util.MaybeString(stmt.RawData["until"]))
		if err != nil {
			return nil, stmt.Errorf("until code parse error: %s", err)
		}
	}
	stmt.Register = util.MaybeString(stmt.RawData["register"])
	stmt.Vars = MapToValue(util.MaybeMap(stmt.RawData["vars"]))
	return stmt, nil
}

func (stmt *Stmt) parser() StmtParser {
	for key, value := range stmt.RawData {
		parser := GetStmtParser(key)
		if parser != nil {
			stmt.funcName = key
			stmt.Args = MapToValue(parseCode(key, value))
			return parser
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
func (stmt *Stmt) Func() (func(vm *VM, context *Context) (interface{}, error), error) {
	stmtParser := stmt.parser()
	if stmtParser == nil {
		stmtParser = vars
	}
	f, err := stmtParser(stmt)
	if err != nil {
		return nil, stmt.Errorf("StmtParser failed: %s", err)
	}
	return applyWrappers(stmt, f)
}

//StmtsToFunc creates list of func from stmts
func StmtsToFunc(funcName string, stmts []*Stmt) (func(*VM, *Context) (interface{}, error), error) {
	runners := []func(*VM, *Context) (interface{}, error){}
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
	return func(vm *VM, context *Context) (value interface{}, err error) {
		for _, f := range runners {
			select {
			case f := <-vm.StopChan:
				f()
			default:
				if value, err = f(vm, context); err != nil {
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
func MapToValue(a map[string]interface{}) map[string]Value {
	result := map[string]Value{}
	if a == nil {
		return result
	}
	for key, value := range a {
		result[key] = NewValue(value)
	}
	return result
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
