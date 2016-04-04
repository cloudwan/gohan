package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {
	gohanscript.RegisterStmtParser("append",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				value, _ := stmt.Arg("value", context).(interface{})
				result1 := lib.Append(list, value)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Append",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			value, _ := args[i].(interface{})
			i++
			result1 := lib.Append(list, value)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("contains",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				value, _ := stmt.Arg("value", context).(interface{})
				result1 := lib.Contains(list, value)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Contains",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			value, _ := args[i].(interface{})
			i++
			result1 := lib.Contains(list, value)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("size",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				result1 := lib.Size(list)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Size",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1 := lib.Size(list)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("shift",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				var err error
				result1, result2 := lib.Shift(list)
				return []interface{}{result1, result2}, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Shift",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1, result2 := lib.Shift(list)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("unshift",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				value, _ := stmt.Arg("value", context).(interface{})
				result1 := lib.Unshift(list, value)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Unshift",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			value, _ := args[i].(interface{})
			i++
			result1 := lib.Unshift(list, value)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("copy",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				result1 := lib.Copy(list)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Copy",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1 := lib.Copy(list)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				index, _ := stmt.Arg("index", context).(int)
				result1 := lib.Delete(list, index)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Delete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			index, _ := args[i].(int)
			i++
			result1 := lib.Delete(list, index)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("first",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				result1 := lib.First(list)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("First",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1 := lib.First(list)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("last",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				result1 := lib.Last(list)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Last",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1 := lib.Last(list)
			return []interface{}{result1}
		})
}
