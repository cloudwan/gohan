package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {
	gohanscript.RegisterStmtParser("uuid",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				result1 := lib.UUID()
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("UUID",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			result1 := lib.UUID()
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("env",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				result1 := lib.Env()
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Env",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			result1 := lib.Env()
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("normalize_map",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				data, _ := stmt.Arg("data", context).(map[string]interface{})
				result1 := lib.NormalizeMap(data)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("NormalizeMap",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			data, _ := args[i].(map[string]interface{})
			i++
			result1 := lib.NormalizeMap(data)
			return []interface{}{result1}
		})
}
