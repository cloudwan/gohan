package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {

	gohanscript.RegisterStmtParser("http_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"url", "headers")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				url := stmt.Arg("url", context).(string)
				headers := stmt.Arg("headers", context).(map[string]interface{})

				result1,
					err :=
					lib.HTTPGet(
						url, headers)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPGet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			url := args[0].(string)
			headers := args[0].(map[string]interface{})

			result1,
				err :=
				lib.HTTPGet(
					url, headers)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("http_post",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"url", "headers", "post_data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				url := stmt.Arg("url", context).(string)
				headers := stmt.Arg("headers", context).(map[string]interface{})
				postData := stmt.Arg("post_data", context).(map[string]interface{})

				result1,
					err :=
					lib.HTTPPost(
						url, headers, postData)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPPost",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			url := args[0].(string)
			headers := args[0].(map[string]interface{})
			postData := args[0].(map[string]interface{})

			result1,
				err :=
				lib.HTTPPost(
					url, headers, postData)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("http_put",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"url", "headers", "post_data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				url := stmt.Arg("url", context).(string)
				headers := stmt.Arg("headers", context).(map[string]interface{})
				postData := stmt.Arg("post_data", context).(map[string]interface{})

				result1,
					err :=
					lib.HTTPPut(
						url, headers, postData)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPPut",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			url := args[0].(string)
			headers := args[0].(map[string]interface{})
			postData := args[0].(map[string]interface{})

			result1,
				err :=
				lib.HTTPPut(
					url, headers, postData)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("http_patch",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"url", "headers", "post_data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				url := stmt.Arg("url", context).(string)
				headers := stmt.Arg("headers", context).(map[string]interface{})
				postData := stmt.Arg("post_data", context).(map[string]interface{})

				result1,
					err :=
					lib.HTTPPatch(
						url, headers, postData)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPPatch",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			url := args[0].(string)
			headers := args[0].(map[string]interface{})
			postData := args[0].(map[string]interface{})

			result1,
				err :=
				lib.HTTPPatch(
					url, headers, postData)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("http_delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"url", "headers")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				url := stmt.Arg("url", context).(string)
				headers := stmt.Arg("headers", context).(map[string]interface{})

				result1,
					err :=
					lib.HTTPDelete(
						url, headers)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPDelete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			url := args[0].(string)
			headers := args[0].(map[string]interface{})

			result1,
				err :=
				lib.HTTPDelete(
					url, headers)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("http_request",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"url", "method", "headers", "post_data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				url := stmt.Arg("url", context).(string)
				method := stmt.Arg("method", context).(string)
				headers := stmt.Arg("headers", context).(map[string]interface{})
				postData := stmt.Arg("post_data", context).(map[string]interface{})

				result1,
					err :=
					lib.HTTPRequest(
						url, method, headers, postData)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPRequest",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			url := args[0].(string)
			method := args[0].(string)
			headers := args[0].(map[string]interface{})
			postData := args[0].(map[string]interface{})

			result1,
				err :=
				lib.HTTPRequest(
					url, method, headers, postData)
			return []interface{}{
				result1,
				err}

		})

}
