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
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

//Minigo compiler

//CompileExpr compiles single line of code
func CompileExpr(expr string) (*MiniGo, error) {
	f, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, err
	}
	code := newMiniGo()
	code.compileExpr(f)
	return code, nil
}

//CompileFile compiles miniGo code and register func
func CompileFile(file string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		return err
	}
	compileFile(f)
	return nil
}

func compileFile(file *ast.File) (err error) {
	for _, decl := range file.Decls {
		switch decl.(type) {
		case *ast.FuncDecl:
			compileFunc(decl.(*ast.FuncDecl))
		}
	}
	return nil
}

func compileFunc(funcDecl *ast.FuncDecl) {
	code := newMiniGo()
	for _, args := range funcDecl.Type.Params.List {
		index := code.addIdent(args.Names[0].Name)
		code.addOp(IDENT, index, 0)
		code.addOp(SET, 0, 0)
	}
	code.compileBlockStmt(funcDecl.Body)
	RegisterMiniGoFunc(funcDecl.Name.Name, func(vm *VM, args []interface{}) []interface{} {
		stack := NewStack()
		stack.stack = args
		context := NewContext(vm)
		code.Eval(context, 0, stack)
		return stack.stack
	})
}

//CompileGoStmt compiles miniGo code and register func
func CompileGoStmt(file string) (*MiniGo, error) {
	fset := token.NewFileSet()
	code := fmt.Sprintf(`
        package main
        func main(){
            %s
        }`, file)
	f, err := parser.ParseFile(fset, "<build-in>", code, 0)
	if err != nil {
		return nil, err
	}
	return compileGoStmt(f)
}

func compileGoStmt(file *ast.File) (*MiniGo, error) {
	for _, decl := range file.Decls {
		switch decl.(type) {
		case *ast.FuncDecl:
			return compileGoStmtFunc(decl.(*ast.FuncDecl))
		}
	}
	return nil, nil
}

func compileGoStmtFunc(funcDecl *ast.FuncDecl) (*MiniGo, error) {
	code := newMiniGo()
	for _, args := range funcDecl.Type.Params.List {
		index := code.addIdent(args.Names[0].Name)
		code.addOp(IDENT, index, 0)
		code.addOp(SET, 0, 0)
	}
	code.compileBlockStmt(funcDecl.Body)
	return code, code.err
}

func (code *MiniGo) compileBlockStmt(block *ast.BlockStmt) {
	for _, stmt := range block.List {
		code.compileStmt(stmt)
		if code.err != nil {
			return
		}
	}
	return
}

func (code *MiniGo) compileStmt(stmt ast.Stmt) {
	if code.err != nil {
		return
	}
	switch stmt.(type) {
	case (*ast.BlockStmt):
		code.compileBlockStmt(stmt.(*ast.BlockStmt))
	case (*ast.ExprStmt):
		code.compileExprStmt(stmt.(*ast.ExprStmt))
	case (*ast.AssignStmt):
		code.compileAssignStmt(stmt.(*ast.AssignStmt))
	case (*ast.ForStmt):
		code.compileForStmt(stmt.(*ast.ForStmt))
	case (*ast.IfStmt):
		code.compileIfStmt(stmt.(*ast.IfStmt))
	case (*ast.ReturnStmt):
		code.compileReturnStmt(stmt.(*ast.ReturnStmt))
	case (*ast.IncDecStmt):
		code.compileIncDecStmt(stmt.(*ast.IncDecStmt))
	case (*ast.RangeStmt):
		code.compileRangeStmt(stmt.(*ast.RangeStmt))
	}
}

func (code *MiniGo) compileExprStmt(exprStmt *ast.ExprStmt) {
	code.compileExpr(exprStmt.X)
}

func (code *MiniGo) compileReturnStmt(returnStmt *ast.ReturnStmt) {
	for _, result := range returnStmt.Results {
		code.compileExpr(result)
	}
	code.addOp(RET, 0, 0)
}

func (code *MiniGo) compileExpr(expr ast.Expr) {
	if code.err != nil {
		return
	}
	switch e := expr.(type) {
	case *ast.CallExpr:
		code.compileCallExpr(e)
	case *ast.SelectorExpr:
		code.compileSelectorExpr(e)
	case *ast.IndexExpr:
		code.compileIndexExpr(e)
	case *ast.BinaryExpr:
		code.compileBinaryExpr(e)
	case *ast.Ident:
		code.compileIdent(e)
	case *ast.BasicLit:
		code.compileBasicLit(e)
	case *ast.ParenExpr:
		code.compileParenExpr(e)
	case *ast.UnaryExpr:
		code.compileUnaryExpr(e)
	}
}

func (code *MiniGo) compileBasicLit(basicLit *ast.BasicLit) {
	switch basicLit.Kind {
	case token.INT:
		i, err := strconv.ParseInt(basicLit.Value, 10, 64)
		if err != nil {
			code.error(err)
			return
		}
		code.addOp(INT, int(i), 0)
	case token.FLOAT:
		f, err := strconv.ParseFloat(basicLit.Value, 64)
		if err != nil {
			code.error(err)
			return
		}
		index := code.addFloat(f)
		code.addOp(FLOAT, index, 0)
	case token.CHAR:
		value := basicLit.Value[0]
		index := code.addChar(rune(value))
		code.addOp(CHAR, index, 0)
	case token.STRING:
		value, err := strconv.Unquote(basicLit.Value)
		if err != nil {
			code.err = err
			return
		}
		index := code.addString(value)
		code.addOp(STRING, index, 0)
	}
}

func (code *MiniGo) compileIdent(ident *ast.Ident) {
	index := code.addIdent(ident.Name)
	code.addOp(IDENT, index, 0)
	code.addOp(GET, 0, 0)
}

func (code *MiniGo) compileParenExpr(parenExpr *ast.ParenExpr) {
	code.compileExpr(parenExpr.X)
}

func (code *MiniGo) compileUnaryExpr(unaryExpr *ast.UnaryExpr) {
	code.compileExpr(unaryExpr.X)
	if OpCode(unaryExpr.Op) == SUB {
		code.addOp(SUB_UNARY, 0, 0)
	} else {
		code.addOp(OpCode(unaryExpr.Op), 0, 0)
	}
}

func (code *MiniGo) compileCallExpr(callExpr *ast.CallExpr) {
	for i := len(callExpr.Args) - 1; i >= 0; i-- {
		expr := callExpr.Args[i]
		code.compileExpr(expr)
	}
	code.compileExpr(callExpr.Fun)
	code.addOp(CALL, len(callExpr.Args), 0)
}

func (code *MiniGo) compileSelectorExpr(selectorExpr *ast.SelectorExpr) {
	code.compileExpr(selectorExpr.X)
	index := code.addIdent(selectorExpr.Sel.Name)
	code.addOp(IDENT, index, 0)
	code.addOp(GETPROP, 0, 0)
}

func (code *MiniGo) compileIndexExpr(indexExpr *ast.IndexExpr) {
	code.compileExpr(indexExpr.X)
	code.compileExpr(indexExpr.Index)

	code.addOp(GETINDEX, 0, 0)
}

func (code *MiniGo) compileBinaryExpr(binaryExpr *ast.BinaryExpr) {
	code.compileExpr(binaryExpr.X)
	code.compileExpr(binaryExpr.Y)
	code.addOp(OpCode(binaryExpr.Op), 0, 0)
}

func (code *MiniGo) compileAssignStmt(assignStmt *ast.AssignStmt) {
	for _, rhs := range assignStmt.Rhs {
		code.compileExpr(rhs)
	}
	for _, lhs := range assignStmt.Lhs {
		switch l := lhs.(type) {
		case (*ast.SelectorExpr):
			code.compileExpr(l.X)
			index := code.addIdent(l.Sel.Name)
			code.addOp(IDENT, index, 0)
			code.addOp(SETPROP, 0, 0)
		case (*ast.IndexExpr):
			code.compileExpr(l.X)
			code.compileExpr(l.Index)
			code.addOp(SETINDEX, 0, 0)
		case (*ast.Ident):
			ident := lhs.(*ast.Ident)
			index := code.addIdent(ident.Name)
			code.addOp(IDENT, index, 0)
			code.addOp(SET, 0, 0)
		default:
			code.error(fmt.Errorf("lhs should be ident or selector"))
		}
	}
}

func (code *MiniGo) compileIncDecStmt(incDecStmt *ast.IncDecStmt) {
	x := incDecStmt.X
	switch e := x.(type) {
	case *ast.Ident:
		index := code.addIdent(e.Name)
		code.addOp(IDENT, index, 0)
		code.addOp(OpCode(incDecStmt.Tok), 0, 0)
	case *ast.SelectorExpr:
		code.compileExpr(e.X)
		index := code.addIdent(e.Sel.Name)
		code.addOp(IDENT, index, 0)
		switch incDecStmt.Tok {
		case token.INC:
			code.addOp(INCPROP, 0, 0)
		case token.DEC:
			code.addOp(DECPROP, 0, 0)
		}
	case *ast.IndexExpr:
		code.compileExpr(e.X)
		code.compileExpr(e.Index)
		switch incDecStmt.Tok {
		case token.INC:
			code.addOp(INCINDEX, 0, 0)
		case token.DEC:
			code.addOp(DECINDEX, 0, 0)
		}
	}
}

func (code *MiniGo) compileForStmt(forStmt *ast.ForStmt) {
	code.compileStmt(forStmt.Init)
	index := code.len()
	code.compileExpr(forStmt.Cond)
	gotoif := &Op{
		code: GOTOIF,
		x:    code.len() + 1,
		y:    0,
	}
	if forStmt.Cond != nil {
		code.ops = append(code.ops, gotoif)
	}
	code.compileStmt(forStmt.Body)
	code.compileStmt(forStmt.Post)
	code.addOp(GOTO, index, 0)
	breakIndex := code.len()
	gotoif.y = breakIndex
}

func (code *MiniGo) compileRangeStmt(rangeStmt *ast.RangeStmt) {
	key := rangeStmt.Key.(*ast.Ident)
	assignStmt := key.Obj.Decl.(*ast.AssignStmt)
	rhs := assignStmt.Rhs[0].(*ast.UnaryExpr)
	code.compileExpr(rhs.X)
	rangeOp := &Op{
		code: RANGE,
		x:    0,
		y:    len(assignStmt.Lhs),
	}
	code.ops = append(code.ops, rangeOp)
	for _, lhs := range assignStmt.Lhs {
		ident := lhs.(*ast.Ident)
		index := code.addIdent(ident.Name)
		code.addOp(IDENT, index, 0)
		code.addOp(SET, 0, 0)
	}
	code.compileStmt(rangeStmt.Body)
	code.addOp(RET, 0, 0)
	rangeOp.x = code.len()
}

func (code *MiniGo) compileIfStmt(ifStmt *ast.IfStmt) {
	code.compileExpr(ifStmt.Cond)
	gotoif := &Op{
		code: GOTOIF,
		x:    code.len() + 1,
		y:    0,
	}
	code.ops = append(code.ops, gotoif)
	code.compileStmt(ifStmt.Body)
	gotoEnd := &Op{
		code: GOTO,
		x:    0,
		y:    0,
	}
	code.ops = append(code.ops, gotoEnd)
	gotoif.y = code.len()
	code.compileStmt(ifStmt.Else)
	gotoEnd.x = code.len()
}
