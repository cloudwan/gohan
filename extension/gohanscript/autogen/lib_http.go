package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {
	gohanscript.RegisterStmtParser("http_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				url, _ := stmt.Arg("url", context).(string)
				headers, _ := stmt.Arg("headers", context).(map[string]interface{})
				var err error
				result1, err := lib.HTTPGet(url, headers)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPGet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			url, _ := args[i].(string)
			i++
			headers, _ := args[i].(map[string]interface{})
			i++
			result1, result2 := lib.HTTPGet(url, headers)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("http_post",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				url, _ := stmt.Arg("url", context).(string)
				headers, _ := stmt.Arg("headers", context).(map[string]interface{})
				postData, _ := stmt.Arg("post_data", context).(map[string]interface{})
				var err error
				result1, err := lib.HTTPPost(url, headers, postData)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPPost",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			url, _ := args[i].(string)
			i++
			headers, _ := args[i].(map[string]interface{})
			i++
			postData, _ := args[i].(map[string]interface{})
			i++
			result1, result2 := lib.HTTPPost(url, headers, postData)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("http_put",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				url, _ := stmt.Arg("url", context).(string)
				headers, _ := stmt.Arg("headers", context).(map[string]interface{})
				postData, _ := stmt.Arg("post_data", context).(map[string]interface{})
				var err error
				result1, err := lib.HTTPPut(url, headers, postData)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPPut",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			url, _ := args[i].(string)
			i++
			headers, _ := args[i].(map[string]interface{})
			i++
			postData, _ := args[i].(map[string]interface{})
			i++
			result1, result2 := lib.HTTPPut(url, headers, postData)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("http_patch",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				url, _ := stmt.Arg("url", context).(string)
				headers, _ := stmt.Arg("headers", context).(map[string]interface{})
				postData, _ := stmt.Arg("post_data", context).(map[string]interface{})
				var err error
				result1, err := lib.HTTPPatch(url, headers, postData)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPPatch",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			url, _ := args[i].(string)
			i++
			headers, _ := args[i].(map[string]interface{})
			i++
			postData, _ := args[i].(map[string]interface{})
			i++
			result1, result2 := lib.HTTPPatch(url, headers, postData)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("http_delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				url, _ := stmt.Arg("url", context).(string)
				headers, _ := stmt.Arg("headers", context).(map[string]interface{})
				var err error
				result1, err := lib.HTTPDelete(url, headers)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPDelete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			url, _ := args[i].(string)
			i++
			headers, _ := args[i].(map[string]interface{})
			i++
			result1, result2 := lib.HTTPDelete(url, headers)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("http_request",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				url, _ := stmt.Arg("url", context).(string)
				method, _ := stmt.Arg("method", context).(string)
				headers, _ := stmt.Arg("headers", context).(map[string]interface{})
				postData, _ := stmt.Arg("post_data", context).(map[string]interface{})
				var err error
				result1, err := lib.HTTPRequest(url, method, headers, postData)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("HTTPRequest",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			url, _ := args[i].(string)
			i++
			method, _ := args[i].(string)
			i++
			headers, _ := args[i].(map[string]interface{})
			i++
			postData, _ := args[i].(map[string]interface{})
			i++
			result1, result2 := lib.HTTPRequest(url, method, headers, postData)
			return []interface{}{result1, result2}
		})
}
