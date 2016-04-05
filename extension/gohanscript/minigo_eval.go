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
)

//Stack express stack in running context
type Stack struct {
	stack []interface{}
}

//NewStack makes stack
func NewStack() *Stack {
	return &Stack{}
}

func (stack *Stack) push(val interface{}) {
	stack.stack = append(stack.stack, val)
}

func (stack *Stack) pop() interface{} {
	item := stack.stack[len(stack.stack)-1]
	stack.stack = stack.stack[:len(stack.stack)-1]
	return item
}

//Run code with given context.
func (code *MiniGo) Run(context *Context) (interface{}, error) {
	results, err := code.Eval(context, 0, NewStack())
	if err != nil {
		return nil, err
	}
	if len(results) < 1 {
		return nil, nil
	}
	return results[0], nil
}

//Eval executes byte code.
func (code *MiniGo) Eval(context *Context, offset int, stack *Stack) ([]interface{}, error) {
	vm := context.VM
	for ; offset < code.len(); offset++ {
		select {
		case f := <-vm.StopChan:
			f()
		default:
			op := code.ops[offset]
			switch op.code {
			case IDENT:
				stack.push(code.idents[op.x])
			case INT:
				stack.push(op.x)
			case FLOAT:
				stack.push(code.floats[op.x])
			case CHAR:
				stack.push(code.chars[op.x])
			case STRING:
				stack.push(code.strings[op.x])
			case ADD:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(a + ib.(int))
				case float64:
					stack.push(a + ib.(float64))
				case string:
					stack.push(ib.(string) + a)
				default:
					return nil, fmt.Errorf("operator + not defined for type")
				}
			case SUB:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(ib.(int) - a)
				case float64:
					stack.push(ib.(float64) - a)
				default:
					return nil, fmt.Errorf("operator - not defined for type")
				}
			case MUL:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(ib.(int) * a)
				case float64:
					stack.push(ib.(float64) * a)
				default:
					return nil, fmt.Errorf("operator * not defined for type")
				}
			case QUO:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(ib.(int) / a)
				case float64:
					stack.push(ib.(float64) / a)
				default:
					return nil, fmt.Errorf("operator / not defined for type")
				}
			case REM:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(ib.(int) % a)
				default:
					return nil, fmt.Errorf("operator is not defined for type")
				}
			case AND:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(ib.(int) & a)
				default:
					return nil, fmt.Errorf("operator & not defined for type")
				}
			case OR:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(ib.(int) | a)
				default:
					return nil, fmt.Errorf("operator | not defined for type")
				}
			case XOR:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(ib.(int) ^ a)
				default:
					return nil, fmt.Errorf("operator ^ not defined for type")
				}
			case SHL:
				return nil, fmt.Errorf("operator << not defined for type")
			case SHR:
				return nil, fmt.Errorf("operator >> not defined for type")
			case AND_NOT:
				ia := stack.pop()
				ib := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(ib.(int) &^ a)
				default:
					return nil, fmt.Errorf("operator &^ not defined for type")
				}
			case INC:
				ident := stack.pop()
				i := context.MaybeInt(ident.(string))
				context.Set(ident.(string), i+1)
			case DEC:
				ident := stack.pop()
				i := context.MaybeInt(ident.(string))
				context.Set(ident.(string), i-1)
			case EQL:
				ir := stack.pop()
				il := stack.pop()
				stack.push(il == ir)
			case LSS:
				ir := stack.pop()
				il := stack.pop()
				switch r := ir.(type) {
				case int:
					stack.push(il.(int) < r)
				case float64:
					stack.push(il.(float64) < r)
				default:
					return nil, fmt.Errorf("operator < not defined for type")
				}
			case GTR:
				ir := stack.pop()
				il := stack.pop()
				switch r := ir.(type) {
				case int:
					stack.push(il.(int) > r)
				case float64:
					stack.push(il.(float64) > r)
				default:
					return nil, fmt.Errorf("operator > not defined for type")
				}
			case NOT:
				ir := stack.pop()
				switch r := ir.(type) {
				case bool:
					stack.push(!r)
				default:
					return nil, fmt.Errorf("operator ! not defined for type")
				}
			case NEQ:
				ir := stack.pop()
				il := stack.pop()
				stack.push(il != ir)
			case LEQ:
				ir := stack.pop()
				il := stack.pop()
				switch r := ir.(type) {
				case int:
					stack.push(il.(int) <= r)
				case float64:
					stack.push(il.(float64) <= r)
				default:
					return nil, fmt.Errorf("operator <= not defined for type")
				}
			case GEQ:
				ir := stack.pop()
				il := stack.pop()
				switch r := ir.(type) {
				case int:
					stack.push(il.(int) >= r)
				case float64:
					stack.push(il.(float64) >= r)
				default:
					return nil, fmt.Errorf("operator >= not defined for type")
				}
			case LAND:
				ir := stack.pop()
				il := stack.pop()
				switch r := ir.(type) {
				case bool:
					stack.push(il.(bool) && r)
				default:
					return nil, fmt.Errorf("operator <= not defined for type")
				}
			case LOR:
				ir := stack.pop()
				il := stack.pop()
				switch r := ir.(type) {
				case bool:
					stack.push(il.(bool) || r)
				default:
					return nil, fmt.Errorf("operator <= not defined for type")
				}
			case CALL:
				f := stack.pop().(MiniGoFunc)
				args := make([]interface{}, op.x)
				for i := 0; i < op.x; i++ {
					args[i] = stack.pop()
				}
				results := f(vm, args)
				for _, val := range results {
					stack.push(val)
				}
			case GET:
				ident := stack.pop()
				value, err := context.Get(ident.(string))
				if err != nil {
					return nil, err
				}
				stack.push(value)
			case GETPROP:
				ident := stack.pop()
				object := stack.pop()
				stack.push(object.(map[string]interface{})[ident.(string)])
			case SETPROP:
				object := stack.pop()
				ident := stack.pop()
				value := stack.pop()
				object.(map[string]interface{})[ident.(string)] = value
			case GETINDEX:
				index := stack.pop()
				object := stack.pop()
				switch t := index.(type) {
				case int:
					stack.push(object.([]interface{})[t])
				case string:
					stack.push(object.(map[string]interface{})[t])
				}
			case SETINDEX:
				index := stack.pop()
				object := stack.pop()
				value := stack.pop()
				switch t := index.(type) {
				case int:
					object.([]interface{})[t] = value
				case string:
					object.(map[string]interface{})[t] = value
				}
			case SET:
				ident := stack.pop()
				value := stack.pop()
				context.Set(ident.(string), value)
			case GOTO:
				offset = op.x - 1
			case GOTOIF:
				flag := stack.pop()
				if flag.(bool) {
					offset = op.x - 1
				} else {
					offset = op.y - 1
				}
			case RET:
				return stack.stack, nil
			case RANGE:
				object := stack.pop()
				switch t := object.(type) {
				case []interface{}:
					for i, value := range t {
						if op.y == 2 {
							stack.push(value)
						}
						stack.push(i)
						code.Eval(context, offset+1, stack)
					}
				case map[string]interface{}:
					for key, value := range t {
						if op.y == 2 {
							stack.push(value)
						}
						stack.push(key)
						code.Eval(context, offset+1, stack)
					}
				}
				offset = op.x - 1
			case SUB_UNARY:
				ia := stack.pop()
				switch a := ia.(type) {
				case int:
					stack.push(-a)
				case float64:
					stack.push(-a)
				default:
					return nil, fmt.Errorf("operator - not defined for type")
				}
			default:
				return nil, fmt.Errorf("Op not supported")
			}
		}
	}
	return stack.stack, nil
}
