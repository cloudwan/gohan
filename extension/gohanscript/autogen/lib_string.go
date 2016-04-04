package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {
	gohanscript.RegisterStmtParser("split",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				value, _ := stmt.Arg("value", context).(string)
				sep, _ := stmt.Arg("sep", context).(string)
				var err error
				result1, err := lib.Split(value, sep)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Split",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			value, _ := args[i].(string)
			i++
			sep, _ := args[i].(string)
			i++
			result1, result2 := lib.Split(value, sep)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("join",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				value, _ := stmt.Arg("value", context).([]interface{})
				sep, _ := stmt.Arg("sep", context).(string)
				var err error
				result1, err := lib.Join(value, sep)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Join",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			value, _ := args[i].([]interface{})
			i++
			sep, _ := args[i].(string)
			i++
			result1, result2 := lib.Join(value, sep)
			return []interface{}{result1, result2}
		})
}
