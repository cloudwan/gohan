package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"net/http/httptest"

	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {
	gohanscript.RegisterStmtParser("get_test_server_url",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				server, _ := stmt.Arg("server", context).(*httptest.Server)
				result1 := lib.GetTestServerURL(server)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GetTestServerURL",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			server, _ := args[i].(*httptest.Server)
			i++
			result1 := lib.GetTestServerURL(server)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("stop_test_server",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				server, _ := stmt.Arg("server", context).(*httptest.Server)
				lib.StopTestServer(server)
				return nil, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("StopTestServer",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			server, _ := args[i].(*httptest.Server)
			i++
			lib.StopTestServer(server)
			return []interface{}{}
		})
}
