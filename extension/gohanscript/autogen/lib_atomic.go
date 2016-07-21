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

				result1 :=
					lib.MakeMap()

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MakeMap",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			result1 :=
				lib.MakeMap()
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("map_set",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var m cmap.ConcurrentMap
				im := stmt.Arg("m", context)
				if im != nil {
					m = im.(cmap.ConcurrentMap)
				}
				var key string
				ikey := stmt.Arg("key", context)
				if ikey != nil {
					key = ikey.(string)
				}
				var value interface{}
				ivalue := stmt.Arg("value", context)
				if ivalue != nil {
					value = ivalue.(interface{})
				}

				lib.MapSet(
					m, key, value)
				return nil, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MapSet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			m, _ := args[0].(cmap.ConcurrentMap)
			key, _ := args[0].(string)
			value, _ := args[0].(interface{})

			lib.MapSet(
				m, key, value)
			return nil

		})

	gohanscript.RegisterStmtParser("map_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var m cmap.ConcurrentMap
				im := stmt.Arg("m", context)
				if im != nil {
					m = im.(cmap.ConcurrentMap)
				}
				var key string
				ikey := stmt.Arg("key", context)
				if ikey != nil {
					key = ikey.(string)
				}

				result1 :=
					lib.MapGet(
						m, key)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MapGet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			m, _ := args[0].(cmap.ConcurrentMap)
			key, _ := args[0].(string)

			result1 :=
				lib.MapGet(
					m, key)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("map_has",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var m cmap.ConcurrentMap
				im := stmt.Arg("m", context)
				if im != nil {
					m = im.(cmap.ConcurrentMap)
				}
				var key string
				ikey := stmt.Arg("key", context)
				if ikey != nil {
					key = ikey.(string)
				}

				result1 :=
					lib.MapHas(
						m, key)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MapHas",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			m, _ := args[0].(cmap.ConcurrentMap)
			key, _ := args[0].(string)

			result1 :=
				lib.MapHas(
					m, key)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("map_remove",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var m cmap.ConcurrentMap
				im := stmt.Arg("m", context)
				if im != nil {
					m = im.(cmap.ConcurrentMap)
				}
				var key string
				ikey := stmt.Arg("key", context)
				if ikey != nil {
					key = ikey.(string)
				}

				lib.MapRemove(
					m, key)
				return nil, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MapRemove",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			m, _ := args[0].(cmap.ConcurrentMap)
			key, _ := args[0].(string)

			lib.MapRemove(
				m, key)
			return nil

		})

	gohanscript.RegisterStmtParser("make_counter",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var value int
				ivalue := stmt.Arg("value", context)
				if ivalue != nil {
					value = ivalue.(int)
				}

				result1 :=
					lib.MakeCounter(
						value)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MakeCounter",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			value, _ := args[0].(int)

			result1 :=
				lib.MakeCounter(
					value)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("counter_add",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var counter *util.Counter
				icounter := stmt.Arg("counter", context)
				if icounter != nil {
					counter = icounter.(*util.Counter)
				}
				var value int
				ivalue := stmt.Arg("value", context)
				if ivalue != nil {
					value = ivalue.(int)
				}

				lib.CounterAdd(
					counter, value)
				return nil, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("CounterAdd",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			counter, _ := args[0].(*util.Counter)
			value, _ := args[0].(int)

			lib.CounterAdd(
				counter, value)
			return nil

		})

	gohanscript.RegisterStmtParser("counter_value",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var counter *util.Counter
				icounter := stmt.Arg("counter", context)
				if icounter != nil {
					counter = icounter.(*util.Counter)
				}

				result1 :=
					lib.CounterValue(
						counter)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("CounterValue",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			counter, _ := args[0].(*util.Counter)

			result1 :=
				lib.CounterValue(
					counter)
			return []interface{}{
				result1}

		})

}
