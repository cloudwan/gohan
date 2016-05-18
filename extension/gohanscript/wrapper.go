package gohanscript

import "time"

//Funcwrapper addes extra process on function call
var funcWrappers []func(stmt *Stmt, callback func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error)

func init() {
	funcWrappers = []func(stmt *Stmt, callback func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error){
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

func applyWrappers(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	var err error
	for _, wrapper := range funcWrappers {
		f, err = wrapper(stmt, f)

		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

func retryWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	if stmt.Retry == 0 {
		return f, nil
	}
	return func(context *Context) (value interface{}, err error) {
		for i := 0; i < stmt.Retry; i++ {
			value, err = f(context)
			if err == nil {
				if stmt.Until != nil {
					r, err := stmt.Until.Run(context)
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

func conditionalWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	if stmt.When == nil {
		return f, nil
	}
	elseRunners, err := StmtsToFunc(stmt.funcName+".else", stmt.ElseStmt)
	if err != nil {
		return nil, err
	}
	return func(context *Context) (value interface{}, err error) {
		r, err := stmt.When.Run(context)
		if err != nil {
			return nil, err
		}
		if r != true {
			if elseRunners != nil {
				value, err = elseRunners(context)
			}
			return
		}
		return f(context)
	}, nil
}

func rescueWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	if stmt.Rescue == nil {
		return f, nil
	}
	rescueRunners, err := StmtsToFunc(stmt.funcName+".rescue", stmt.Rescue)
	if err != nil {
		return nil, err
	}
	return func(context *Context) (value interface{}, err error) {
		value, err = f(context)
		if err != nil {
			context.Set("error", err.Error())
			if rescueRunners != nil {
				value, err = rescueRunners(context)
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

func alwaysWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	if stmt.Always == nil {
		return f, nil
	}
	alwaysRunners, err := StmtsToFunc(stmt.funcName+".always", stmt.Always)
	if err != nil {
		return nil, err
	}
	return func(context *Context) (value interface{}, err error) {
		value, err = f(context)
		value, err = alwaysRunners(context)
		return
	}, nil
}

func registerWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	if stmt.Register == "" {
		return f, nil
	}
	return func(context *Context) (value interface{}, err error) {
		value, err = f(context)
		context.Set(stmt.Register, value)
		return
	}, nil
}

func loopWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	if stmt.WithDict == nil && stmt.WithItems == nil {
		return f, nil
	}
	return func(context *Context) (value interface{}, err error) {
		var items []interface{}
		vm := context.VM
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
			worker := stmt.Worker
			err = forEachList(vm, items, worker, func(item interface{}) error {
				loopContext := context
				if stmt.Worker > 0 {
					loopContext = context.Extend(nil)
				}
				loopContext.Set(stmt.LoopVar, item)
				_, err := f(loopContext)
				if err != nil {
					loopContext.Set("error", err.Error())
					return err
				}
				if stmt.Worker > 0 {
					loopContext.VM.Stop()
				}
				return nil
			})
			if err != nil {
				return
			}
		}
		return
	}, nil
}

func contextWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (value interface{}, err error) {
		context.Set("__file__", stmt.File)
		context.Set("__dir__", stmt.Dir)
		for key, value := range stmt.Vars {
			if value == nil {
				context.SetByKeys(key, nil)
			} else {
				context.SetByKeys(key, value.Value(context))
			}
		}
		return f(context)
	}, nil
}

func panicWrapper(stmt *Stmt, f func(*Context) (interface{}, error)) (func(*Context) (interface{}, error), error) {
	return func(context *Context) (value interface{}, err error) {
		defer func() {
			if caught := recover(); caught != nil {
				if caught == context.VM.timeoutError {
					panic(caught)
				}
				if err, ok := caught.(breakCode); ok {
					panic(err)
				}
				if err, ok := caught.(error); ok {
					panic(stmt.Error(err))
				}
				panic(caught)
			}
		}()
		return f(context)
	}, nil
}
