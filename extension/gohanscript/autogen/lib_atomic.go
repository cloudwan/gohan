package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
	"github.com/cloudwan/gohan/util"
	"github.com/streamrail/concurrent-map"
)

func init() {
	gohanscript.RegisterStmtParser("make_map",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				result1 := lib.MakeMap()
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MakeMap",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			result1 := lib.MakeMap()
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("map_set",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				m, _ := stmt.Arg("m", context).(cmap.ConcurrentMap)
				key, _ := stmt.Arg("key", context).(string)
				value, _ := stmt.Arg("value", context).(interface{})
				lib.MapSet(m, key, value)
				return nil, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MapSet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			m, _ := args[i].(cmap.ConcurrentMap)
			i++
			key, _ := args[i].(string)
			i++
			value, _ := args[i].(interface{})
			i++
			lib.MapSet(m, key, value)
			return []interface{}{}
		})
	gohanscript.RegisterStmtParser("map_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				m, _ := stmt.Arg("m", context).(cmap.ConcurrentMap)
				key, _ := stmt.Arg("key", context).(string)
				result1 := lib.MapGet(m, key)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MapGet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			m, _ := args[i].(cmap.ConcurrentMap)
			i++
			key, _ := args[i].(string)
			i++
			result1 := lib.MapGet(m, key)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("map_has",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				m, _ := stmt.Arg("m", context).(cmap.ConcurrentMap)
				key, _ := stmt.Arg("key", context).(string)
				result1 := lib.MapHas(m, key)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MapHas",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			m, _ := args[i].(cmap.ConcurrentMap)
			i++
			key, _ := args[i].(string)
			i++
			result1 := lib.MapHas(m, key)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("map_remove",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				m, _ := stmt.Arg("m", context).(cmap.ConcurrentMap)
				key, _ := stmt.Arg("key", context).(string)
				lib.MapRemove(m, key)
				return nil, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MapRemove",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			m, _ := args[i].(cmap.ConcurrentMap)
			i++
			key, _ := args[i].(string)
			i++
			lib.MapRemove(m, key)
			return []interface{}{}
		})
	gohanscript.RegisterStmtParser("make_counter",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				value, _ := stmt.Arg("value", context).(int)
				result1 := lib.MakeCounter(value)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MakeCounter",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			value, _ := args[i].(int)
			i++
			result1 := lib.MakeCounter(value)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("counter_add",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				counter, _ := stmt.Arg("counter", context).(*util.Counter)
				value, _ := stmt.Arg("value", context).(int)
				lib.CounterAdd(counter, value)
				return nil, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("CounterAdd",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			counter, _ := args[i].(*util.Counter)
			i++
			value, _ := args[i].(int)
			i++
			lib.CounterAdd(counter, value)
			return []interface{}{}
		})
	gohanscript.RegisterStmtParser("counter_value",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				counter, _ := stmt.Arg("counter", context).(*util.Counter)
				result1 := lib.CounterValue(counter)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("CounterValue",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			counter, _ := args[i].(*util.Counter)
			i++
			result1 := lib.CounterValue(counter)
			return []interface{}{result1}
		})
}
