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

				var server *httptest.Server
				iserver := stmt.Arg("server", context)
				if iserver != nil {
					server = iserver.(*httptest.Server)
				}

				result1 :=
					lib.GetTestServerURL(
						server)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GetTestServerURL",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			server, _ := args[0].(*httptest.Server)

			result1 :=
				lib.GetTestServerURL(
					server)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("stop_test_server",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var server *httptest.Server
				iserver := stmt.Arg("server", context)
				if iserver != nil {
					server = iserver.(*httptest.Server)
				}

				lib.StopTestServer(
					server)
				return nil, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("StopTestServer",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			server, _ := args[0].(*httptest.Server)

			lib.StopTestServer(
				server)
			return nil

		})

	gohanscript.RegisterStmtParser("gohan_server",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {

				var configFile string
				iconfigFile := stmt.Arg("config_file", context)
				if iconfigFile != nil {
					configFile = iconfigFile.(string)
				}
				var test bool
				itest := stmt.Arg("test", context)
				if itest != nil {
					test = itest.(bool)
				}

				result1,
					err :=
					lib.GohanServer(
						configFile, test)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanServer",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			configFile, _ := args[0].(string)
			test, _ := args[0].(bool)

			result1,
				err :=
				lib.GohanServer(
					configFile, test)
			return []interface{}{
				result1,
				err}

		})

}
