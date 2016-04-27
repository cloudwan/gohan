package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {

	gohanscript.RegisterStmtParser("uuid",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs()
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				result1 :=
					lib.UUID()

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("UUID",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			result1 :=
				lib.UUID()
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("env",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs()
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				result1 :=
					lib.Env()

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Env",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			result1 :=
				lib.Env()
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("normalize_map",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var data map[string]interface{}
				idata := stmt.Arg("data", context)
				if idata != nil {
					data = idata.(map[string]interface{})
				}

				result1 :=
					lib.NormalizeMap(
						data)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("NormalizeMap",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			data := args[0].(map[string]interface{})

			result1 :=
				lib.NormalizeMap(
					data)
			return []interface{}{
				result1}

		})

}
