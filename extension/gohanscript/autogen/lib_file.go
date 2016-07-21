package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {

	gohanscript.RegisterStmtParser("fetch_content",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var path string
				ipath := stmt.Arg("path", context)
				if ipath != nil {
					path = ipath.(string)
				}

				result1,
					err :=
					lib.FetchContent(
						path)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("FetchContent",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			path, _ := args[0].(string)

			result1,
				err :=
				lib.FetchContent(
					path)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("save_content",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var path string
				ipath := stmt.Arg("path", context)
				if ipath != nil {
					path = ipath.(string)
				}
				var data interface{}
				idata := stmt.Arg("data", context)
				if idata != nil {
					data = idata.(interface{})
				}

				err :=
					lib.SaveContent(
						path, data)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("SaveContent",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			path, _ := args[0].(string)
			data, _ := args[0].(interface{})

			err :=
				lib.SaveContent(
					path, data)
			return []interface{}{
				err}

		})

}
