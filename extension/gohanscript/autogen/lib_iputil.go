package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {

	gohanscript.RegisterStmtParser("ip_to_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var ip string
				iip := stmt.Arg("ip", context)
				if iip != nil {
					ip = iip.(string)
				}

				result1 :=
					lib.IPToInt(
						ip)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("IPToInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			ip, _ := args[0].(string)

			result1 :=
				lib.IPToInt(
					ip)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("int_to_ip",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var value int
				ivalue := stmt.Arg("value", context)
				if ivalue != nil {
					value = ivalue.(int)
				}

				result1 :=
					lib.IntToIP(
						value)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("IntToIP",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			value, _ := args[0].(int)

			result1 :=
				lib.IntToIP(
					value)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("ip_add",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var ip string
				iip := stmt.Arg("ip", context)
				if iip != nil {
					ip = iip.(string)
				}
				var value int
				ivalue := stmt.Arg("value", context)
				if ivalue != nil {
					value = ivalue.(int)
				}

				result1 :=
					lib.IPAdd(
						ip, value)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("IPAdd",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			ip, _ := args[0].(string)
			value, _ := args[0].(int)

			result1 :=
				lib.IPAdd(
					ip, value)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("parse_cidr",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var cidr string
				icidr := stmt.Arg("cidr", context)
				if icidr != nil {
					cidr = icidr.(string)
				}

				result1,
					result2,
					result3 :=
					lib.ParseCidr(
						cidr)

				return []interface{}{
					result1,
					result2,
					result3}, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("ParseCidr",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			cidr, _ := args[0].(string)

			result1,
				result2,
				result3 :=
				lib.ParseCidr(
					cidr)
			return []interface{}{
				result1,
				result2,
				result3}

		})

	gohanscript.RegisterStmtParser("float_to_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var value float64
				ivalue := stmt.Arg("value", context)
				if ivalue != nil {
					value = ivalue.(float64)
				}

				result1 :=
					lib.FloatToInt(
						value)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("FloatToInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			value, _ := args[0].(float64)

			result1 :=
				lib.FloatToInt(
					value)
			return []interface{}{
				result1}

		})

}
