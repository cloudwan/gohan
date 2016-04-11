package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {

	gohanscript.RegisterStmtParser("append",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list", "value")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})
				value := stmt.Arg("value", context).(interface{})

				result1 :=
					lib.Append(
						list, value)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Append",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})
			value := args[0].(interface{})

			result1 :=
				lib.Append(
					list, value)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("contains",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list", "value")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})
				value := stmt.Arg("value", context).(interface{})

				result1 :=
					lib.Contains(
						list, value)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Contains",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})
			value := args[0].(interface{})

			result1 :=
				lib.Contains(
					list, value)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("size",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})

				result1 :=
					lib.Size(
						list)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Size",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})

			result1 :=
				lib.Size(
					list)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("shift",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})

				result1,
					result2 :=
					lib.Shift(
						list)

				return []interface{}{
					result1,
					result2}, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Shift",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})

			result1,
				result2 :=
				lib.Shift(
					list)
			return []interface{}{
				result1,
				result2}

		})

	gohanscript.RegisterStmtParser("unshift",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list", "value")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})
				value := stmt.Arg("value", context).(interface{})

				result1 :=
					lib.Unshift(
						list, value)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Unshift",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})
			value := args[0].(interface{})

			result1 :=
				lib.Unshift(
					list, value)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("copy",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})

				result1 :=
					lib.Copy(
						list)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Copy",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})

			result1 :=
				lib.Copy(
					list)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list", "index")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})
				index := stmt.Arg("index", context).(int)

				result1 :=
					lib.Delete(
						list, index)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Delete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})
			index := args[0].(int)

			result1 :=
				lib.Delete(
					list, index)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("first",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})

				result1 :=
					lib.First(
						list)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("First",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})

			result1 :=
				lib.First(
					list)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("last",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"list")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				list := stmt.Arg("list", context).([]interface{})

				result1 :=
					lib.Last(
						list)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Last",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			list := args[0].([]interface{})

			result1 :=
				lib.Last(
					list)
			return []interface{}{
				result1}

		})

}
