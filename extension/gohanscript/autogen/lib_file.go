package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {
	gohanscript.RegisterStmtParser("fetch_content",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				path, _ := stmt.Arg("path", context).(string)
				var err error
				result1, err := lib.FetchContent(path)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("FetchContent",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			path, _ := args[i].(string)
			i++
			result1, result2 := lib.FetchContent(path)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("save_content",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				path, _ := stmt.Arg("path", context).(string)
				data, _ := stmt.Arg("data", context).(interface{})
				err := lib.SaveContent(path, data)
				return nil, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("SaveContent",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			path, _ := args[i].(string)
			i++
			data, _ := args[i].(interface{})
			i++
			result1 := lib.SaveContent(path, data)
			return []interface{}{result1}
		})
}
