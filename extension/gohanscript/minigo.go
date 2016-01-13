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
)

//MiniGo is tiny interpreter for evaluating small expression in the gohan scirpt
//MiniGo uses go parser, and implements subset of functions
type MiniGo struct {
	ops      []*Op
	floats   []float64
	imags    []complex128
	chars    []rune
	strings  []string
	idents   []string
	funcAddr map[string]int
	err      error
}

//MiniGoFunc represents function which can be used in minigo
type MiniGoFunc func(vm *VM, args []interface{}) []interface{}

//miniGoFuncs is a register for minigo func
var miniGoFuncs map[string]MiniGoFunc

//RegisterMiniGoFunc register minigo func
func RegisterMiniGoFunc(name string, f MiniGoFunc) {
	miniGoFuncs[name] = f
}

func init() {
	miniGoFuncs = map[string]MiniGoFunc{
		"len": func(vm *VM, args []interface{}) []interface{} {
			return []interface{}{len(args[0].([]interface{}))}
		},
		"print": func(vm *VM, args []interface{}) []interface{} {
			fmt.Println(args)
			return nil
		},
	}
}

func newMiniGo() *MiniGo {
	return &MiniGo{
		ops:     []*Op{},
		floats:  []float64{},
		imags:   []complex128{},
		chars:   []rune{},
		strings: []string{},
		idents:  []string{},
		err:     nil,
	}
}

func (code *MiniGo) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("# Code\n")
	for index, op := range code.ops {
		buffer.WriteString(fmt.Sprintf("%d: %s\n", index, op.String()))
	}

	buffer.WriteString("# Consts\n")

	buffer.WriteString("## floats\n")
	for index, i := range code.floats {
		buffer.WriteString(fmt.Sprintf("%d: %f\n", index, i))
	}
	buffer.WriteString("## imags\n")
	for index, i := range code.imags {
		buffer.WriteString(fmt.Sprintf("%d: %v\n", index, i))
	}
	buffer.WriteString("## chars\n")
	for index, i := range code.chars {
		buffer.WriteString(fmt.Sprintf("%d: %v\n", index, i))
	}
	buffer.WriteString("## strings\n")
	for index, i := range code.strings {
		buffer.WriteString(fmt.Sprintf("%d: %s\n", index, i))
	}
	buffer.WriteString("## idents\n")
	for index, i := range code.idents {
		buffer.WriteString(fmt.Sprintf("%d: %s\n", index, i))
	}
	buffer.WriteString("## functions\n")
	for name, addr := range code.funcAddr {
		buffer.WriteString(fmt.Sprintf("%s: %d\n", name, addr))
	}
	if code.err != nil {
		buffer.WriteString("# error \n")
		buffer.WriteString(code.err.Error())
	}
	return buffer.String()
}

func (code *MiniGo) addIdent(ident string) int {
	index := 0
	for index = range code.idents {
		if code.idents[index] == ident {
			return index
		}
	}
	code.idents = append(code.idents, ident)
	return len(code.idents) - 1
}

func (code *MiniGo) addFloat(i float64) int {
	index := 0
	for index = range code.floats {
		if code.floats[index] == i {
			return index
		}
	}
	code.floats = append(code.floats, i)
	return len(code.floats) - 1
}

func (code *MiniGo) addImag(i complex128) int {
	index := 0
	for index = range code.imags {
		if code.imags[index] == i {
			return index
		}
	}
	code.imags = append(code.imags, i)
	return len(code.imags) - 1
}

func (code *MiniGo) addChar(i rune) int {
	index := 0
	for index = range code.chars {
		if code.chars[index] == i {
			return index
		}
	}
	code.chars = append(code.chars, i)
	return len(code.chars) - 1
}

func (code *MiniGo) addString(i string) int {
	index := 0
	for index = range code.chars {
		if code.strings[index] == i {
			return index
		}
	}
	code.strings = append(code.strings, i)
	return len(code.strings) - 1
}

func (code *MiniGo) addOp(op OpCode, x int, y int) {
	code.ops = append(code.ops, &Op{
		code: op,
		x:    x,
		y:    y,
	})
}

func (code *MiniGo) len() int {
	return len(code.ops)
}

func (code *MiniGo) error(err error) {
	code.err = err
}
