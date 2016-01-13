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

//OpCode ...
type OpCode int

//Op ...
type Op struct {
	code OpCode
	x    int
	y    int
}

const (
	// ILLEGAL tokens
	ILLEGAL OpCode = iota

	POP
	PUSH
	CALL
	GET
	SET

	// Identifiers and basic type literals
	// (these tokens stand for classes of literals)
	IDENT  // main
	INT    // 12345
	FLOAT  // 123.45
	IMAG   // 123.45i
	CHAR   // 'a'
	STRING // "abc"

	// Operators and delimiters
	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // %

	AND     // &
	OR      // |
	XOR     // ^
	SHL     // <<
	SHR     // >>
	AND_NOT // &^

	ADD_ASSIGN // +=
	SUB_ASSIGN // -=
	MUL_ASSIGN // *=
	QUO_ASSIGN // /=
	REM_ASSIGN // %=

	AND_ASSIGN     // &=
	OR_ASSIGN      // |=
	XOR_ASSIGN     // ^=
	SHL_ASSIGN     // <<=
	SHR_ASSIGN     // >>=
	AND_NOT_ASSIGN // &^=

	LAND  // &&
	LOR   // ||
	ARROW // <-
	INC   // ++
	DEC   // --

	EQL    // ==
	LSS    // <
	GTR    // >
	ASSIGN // =
	NOT    // !

	NEQ      // !=
	LEQ      // <=
	GEQ      // >=
	DEFINE   // :=
	ELLIPSIS // ...

	GETPROP
	SETPROP
	INCPROP
	DECPROP

	GETINDEX
	SETINDEX
	INCINDEX
	DECINDEX

	GOTO
	GOTOIF

	RANGE

	RET
	SUB_UNARY
)

func (i OpCode) String() string {
	switch i {
	case ILLEGAL:
		return "ILLEGAL"
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case IMAG:
		return "IMAG"
	case CHAR:
		return "CHAR"
	case STRING:
		return "STRING"
	case ADD:
		return "ADD"
	case SUB:
		return "SUB"
	case MUL:
		return "MUL"
	case QUO:
		return "QUO"
	case REM:
		return "REM"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case XOR:
		return "XOR"
	case SHL:
		return "SHL"
	case SHR:
		return "SHR"
	case AND_NOT:
		return "AND_NOT"
	case ADD_ASSIGN:
		return "ADD_ASSIGN"
	case SUB_ASSIGN:
		return "SUB_ASSIGN"
	case MUL_ASSIGN:
		return "MUL_ASSIGN"
	case QUO_ASSIGN:
		return "QUO_ASSIGN"
	case REM_ASSIGN:
		return "REM_ASSIGN"
	case AND_ASSIGN:
		return "AND_ASSIGN"
	case OR_ASSIGN:
		return "OR_ASSIGN"
	case XOR_ASSIGN:
		return "XOR_ASSIGN"
	case SHL_ASSIGN:
		return "SHL_ASSIGN"
	case SHR_ASSIGN:
		return "SHR_ASSIGN"
	case AND_NOT_ASSIGN:
		return "AND_NOT_ASSIGN"
	case LAND:
		return "LAND"
	case LOR:
		return "LOR"
	case ARROW:
		return "ARROW"
	case INC:
		return "INC"
	case DEC:
		return "DEC"
	case EQL:
		return "EQL"
	case LSS:
		return "LSS"
	case GTR:
		return "GTR"
	case ASSIGN:
		return "ASSIGN"
	case NOT:
		return "NOT"
	case NEQ:
		return "NEQ"
	case LEQ:
		return "LEQ"
	case GEQ:
		return "GEQ"
	case DEFINE:
		return "DEFINE"
	case ELLIPSIS:
		return "ELLIPSIS"
	case POP:
		return "POP"
	case PUSH:
		return "PUSH"
	case CALL:
		return "CALL"
	case GET:
		return "GET"
	case SET:
		return "SET"
	case GETPROP:
		return "GETPROP"
	case SETPROP:
		return "SETPROP"
	case INCPROP:
		return "INCPROP"
	case DECPROP:
		return "DECPROP"
	case GETINDEX:
		return "GETINDEX"
	case SETINDEX:
		return "SETINDEX"
	case INCINDEX:
		return "INCINDEX"
	case DECINDEX:
		return "DECINDEX"
	case GOTO:
		return "GOTO"
	case GOTOIF:
		return "GOTOIF"
	case RET:
		return "RET"
	case RANGE:
		return "RANGE"
	case SUB_UNARY:
		return "SUB_UNARY"
	}
	return "Not defined"
}

func (op Op) String() string {
	return fmt.Sprintf("%s (%d,%d)", op.code.String(), op.x, op.y)
}
