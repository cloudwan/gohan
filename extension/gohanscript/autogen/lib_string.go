package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {

	gohanscript.RegisterStmtParser("split",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"value", "sep")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var value string
				ivalue := stmt.Arg("value", context)
				if ivalue != nil {
					value = ivalue.(string)
				}
				var sep string
				isep := stmt.Arg("sep", context)
				if isep != nil {
					sep = isep.(string)
				}

				result1,
					err :=
					lib.Split(
						value, sep)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Split",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			value := args[0].(string)
			sep := args[1].(string)

			result1,
				err :=
				lib.Split(
					value, sep)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("join",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"value", "sep")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var value []interface{}
				ivalue := stmt.Arg("value", context)
				if ivalue != nil {
					value = ivalue.([]interface{})
				}
				var sep string
				isep := stmt.Arg("sep", context)
				if isep != nil {
					sep = isep.(string)
				}

				result1,
					err :=
					lib.Join(
						value, sep)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Join",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			value := args[0].([]interface{})
			sep := args[0].(string)

			result1,
				err :=
				lib.Join(
					value, sep)
			return []interface{}{
				result1,
				err}

		})

}
