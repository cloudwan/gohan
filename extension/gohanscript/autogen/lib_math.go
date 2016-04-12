package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {

	gohanscript.RegisterStmtParser("add_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"a", "b")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				a := stmt.Arg("a", context).(int)
				b := stmt.Arg("b", context).(int)

				result1 :=
					lib.AddInt(
						a, b)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("AddInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			a := args[0].(int)
			b := args[1].(int)

			result1 :=
				lib.AddInt(
					a, b)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("sub_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"a", "b")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				a := stmt.Arg("a", context).(int)
				b := stmt.Arg("b", context).(int)

				result1 :=
					lib.SubInt(
						a, b)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("SubInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			a := args[0].(int)
			b := args[1].(int)

			result1 :=
				lib.SubInt(
					a, b)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("mul_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"a", "b")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				a := stmt.Arg("a", context).(int)
				b := stmt.Arg("b", context).(int)

				result1 :=
					lib.MulInt(
						a, b)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MulInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			a := args[0].(int)
			b := args[1].(int)

			result1 :=
				lib.MulInt(
					a, b)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("div_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"a", "b")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				a := stmt.Arg("a", context).(int)
				b := stmt.Arg("b", context).(int)

				result1 :=
					lib.DivInt(
						a, b)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DivInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			a := args[0].(int)
			b := args[1].(int)

			result1 :=
				lib.DivInt(
					a, b)
			return []interface{}{
				result1}

		})

}
