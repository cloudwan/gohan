package gohanscript

import (
	"fmt"
	"time"

	"github.com/k0kubun/pp"
	"gopkg.in/yaml.v2"
)

//Funcwrapper addes extra process on function call
var funcWrappers []func(stmt *Stmt, callback func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error)

func init() {
	funcWrappers = []func(stmt *Stmt, callback func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error){
		contextWrapper,
		retryWrapper,
		debugWrapper,
		loopWrapper,
		registerWrapper,
		conditionalWrapper,
		rescueWrapper,
		alwaysWrapper,
		panicWrapper,
	}
}

func applyWrappers(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	var err error
	for _, wrapper := range funcWrappers {
		f, err = wrapper(stmt, f)

		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

func retryWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	if stmt.Retry == 0 {
		return f, nil
	}
	return func(vm *VM, context *Context) (value interface{}, err error) {
		for i := 0; i < stmt.Retry; i++ {
			value, err = f(vm, context)
			if err == nil {
				if stmt.Until != nil {
					r, err := stmt.Until.Run(vm, context)
					if err != nil {
						return nil, err
					}
					if r == true {
						return value, nil
					}
				} else {
					return
				}
			}
			time.Sleep(time.Duration(stmt.Delay) * time.Second)
		}
		return
	}, nil
}

func debugWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	return func(vm *VM, context *Context) (value interface{}, err error) {
		debugNext := false
		if vm.debug {
			var command string
		DEBUG_LOOP:
			for {
				fmt.Printf("%s:%d %s > ", stmt.File, stmt.Line, stmt.Name)
				//TODO(nati) support multi thread
				fmt.Scanf("%s", &command)
				switch command {
				case "s":
					break DEBUG_LOOP
				case "n":
					debugNext = true
					vm.debug = false
					break DEBUG_LOOP
				case "r":
					vm.debugReturn = true
					vm.debug = false
					break DEBUG_LOOP
				case "c":
					vm.debug = false
					break DEBUG_LOOP
				case "p":
					pp.Print(context)
				case "l":
					yamlCode, _ := yaml.Marshal(&stmt.RawData)
					fmt.Println(string(yamlCode))
				default:
					fmt.Println("s: step, n: next, r: return, c: continue, p: print context, l: print current line")
				}
			}
		}
		value, err = f(vm, context)
		if debugNext {
			vm.debug = true
		}
		return
	}, nil
}

func conditionalWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	if stmt.When == nil {
		return f, nil
	}
	elseRunners, err := StmtsToFunc(stmt.funcName+".else", stmt.ElseStmt)
	if err != nil {
		return nil, err
	}
	return func(vm *VM, context *Context) (value interface{}, err error) {
		r, err := stmt.When.Run(vm, context)
		if err != nil {
			return nil, err
		}
		if r != true {
			if elseRunners != nil {
				value, err = elseRunners(vm, context)
			}
			return
		}
		return f(vm, context)
	}, nil
}

func rescueWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	if stmt.Rescue == nil {
		return f, nil
	}
	rescueRunners, err := StmtsToFunc(stmt.funcName+".rescue", stmt.Rescue)
	if err != nil {
		return nil, err
	}
	return func(vm *VM, context *Context) (value interface{}, err error) {
		value, err = f(vm, context)
		if err != nil {
			context.Set("error", err.Error())
			if rescueRunners != nil {
				value, err = rescueRunners(vm, context)
				if err == nil {
					context.Set("error", nil)
				} else {
					context.Set("error", err.Error())
				}
			} else {
				err = nil
				context.Set("error", nil)
			}
		}
		return
	}, nil
}

func alwaysWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	if stmt.Always == nil {
		return f, nil
	}
	alwaysRunners, err := StmtsToFunc(stmt.funcName+".always", stmt.Always)
	if err != nil {
		return nil, err
	}
	return func(vm *VM, context *Context) (value interface{}, err error) {
		value, err = f(vm, context)
		value, err = alwaysRunners(vm, context)
		return
	}, nil
}

func registerWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	if stmt.Register == "" {
		return f, nil
	}
	return func(vm *VM, context *Context) (value interface{}, err error) {
		value, err = f(vm, context)
		context.Set(stmt.Register, value)
		return
	}, nil
}

func loopWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	if stmt.WithDict == nil && stmt.WithItems == nil {
		return f, nil
	}
	return func(vm *VM, context *Context) (value interface{}, err error) {
		var items []interface{}
		if stmt.WithDict != nil {
			if mapItems, ok := stmt.WithDict.Value(context).(map[string]interface{}); ok {
				for key, value := range mapItems {
					items = append(items, map[string]interface{}{
						"key":   key,
						"value": value,
					})
				}
			}
		} else if stmt.WithItems != nil {
			rawItem := stmt.WithItems.Value(context)
			items, _ = rawItem.([]interface{})
		}

		if len(items) > 0 {
			results := []interface{}{}
			worker := stmt.Worker
			if vm.debug {
				worker = 0
			}
			forEachList(vm, items, worker, func(item interface{}) {
				loopContext := context
				if stmt.Worker > 0 {
					loopContext = context.Extend(nil)
				}
				loopContext.Set(stmt.LoopVar, item)
				value, err = f(vm, loopContext)
				if err != nil {
					context.Set("error", err.Error())
					return
				}
				results = append(results, value)
			})
			if err != nil {
				return
			}
			value = results
		}
		return
	}, nil
}

func contextWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	return func(vm *VM, context *Context) (value interface{}, err error) {
		context.Set("__file__", stmt.File)
		context.Set("__dir__", stmt.Dir)
		for key, value := range stmt.Vars {
			if value == nil {
				context.SetByKeys(key, nil)
			} else {
				context.SetByKeys(key, value.Value(context))
			}
		}
		return f(vm, context)
	}, nil
}

func panicWrapper(stmt *Stmt, f func(*VM, *Context) (interface{}, error)) (func(*VM, *Context) (interface{}, error), error) {
	return func(vm *VM, context *Context) (value interface{}, err error) {
		defer func() {
			if caught := recover(); caught != nil {
				if caught == vm.timeoutError {
					panic(caught)
				}
				panic(stmt.Errorf("%s", caught))
			}
		}()
		return f(vm, context)
	}, nil
}
