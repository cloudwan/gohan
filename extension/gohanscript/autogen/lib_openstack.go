package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
	"github.com/rackspace/gophercloud"
)

func init() {
	gohanscript.RegisterStmtParser("get_openstack_client",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				authURL, _ := stmt.Arg("auth_url", context).(string)
				userName, _ := stmt.Arg("user_name", context).(string)
				password, _ := stmt.Arg("password", context).(string)
				domainName, _ := stmt.Arg("domain_name", context).(string)
				tenantName, _ := stmt.Arg("tenant_name", context).(string)
				version, _ := stmt.Arg("version", context).(string)
				var err error
				result1, err := lib.GetOpenstackClient(authURL, userName, password, domainName, tenantName, version)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GetOpenstackClient",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			authURL, _ := args[i].(string)
			i++
			userName, _ := args[i].(string)
			i++
			password, _ := args[i].(string)
			i++
			domainName, _ := args[i].(string)
			i++
			tenantName, _ := args[i].(string)
			i++
			version, _ := args[i].(string)
			i++
			result1, result2 := lib.GetOpenstackClient(authURL, userName, password, domainName, tenantName, version)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("openstack_token",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				client, _ := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				result1 := lib.OpenstackToken(client)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackToken",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			client, _ := args[i].(*gophercloud.ServiceClient)
			i++
			result1 := lib.OpenstackToken(client)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("openstack_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				client, _ := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				url, _ := stmt.Arg("url", context).(string)
				var err error
				result1, err := lib.OpenstackGet(client, url)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackGet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			client, _ := args[i].(*gophercloud.ServiceClient)
			i++
			url, _ := args[i].(string)
			i++
			result1, result2 := lib.OpenstackGet(client, url)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("openstack_put",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				client, _ := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				url, _ := stmt.Arg("url", context).(string)
				data, _ := stmt.Arg("data", context).(interface{})
				var err error
				result1, err := lib.OpenstackPut(client, url, data)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackPut",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			client, _ := args[i].(*gophercloud.ServiceClient)
			i++
			url, _ := args[i].(string)
			i++
			data, _ := args[i].(interface{})
			i++
			result1, result2 := lib.OpenstackPut(client, url, data)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("openstack_post",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				client, _ := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				url, _ := stmt.Arg("url", context).(string)
				data, _ := stmt.Arg("data", context).(interface{})
				var err error
				result1, err := lib.OpenstackPost(client, url, data)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackPost",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			client, _ := args[i].(*gophercloud.ServiceClient)
			i++
			url, _ := args[i].(string)
			i++
			data, _ := args[i].(interface{})
			i++
			result1, result2 := lib.OpenstackPost(client, url, data)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("openstack_delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				client, _ := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				url, _ := stmt.Arg("url", context).(string)
				var err error
				result1, err := lib.OpenstackDelete(client, url)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackDelete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			client, _ := args[i].(*gophercloud.ServiceClient)
			i++
			url, _ := args[i].(string)
			i++
			result1, result2 := lib.OpenstackDelete(client, url)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("openstack_endpoint",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				client, _ := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				endpointType, _ := stmt.Arg("endpoint_type", context).(string)
				name, _ := stmt.Arg("name", context).(string)
				region, _ := stmt.Arg("region", context).(string)
				availability, _ := stmt.Arg("availability", context).(string)
				var err error
				result1, err := lib.OpenstackEndpoint(client, endpointType, name, region, availability)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackEndpoint",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			client, _ := args[i].(*gophercloud.ServiceClient)
			i++
			endpointType, _ := args[i].(string)
			i++
			name, _ := args[i].(string)
			i++
			region, _ := args[i].(string)
			i++
			availability, _ := args[i].(string)
			i++
			result1, result2 := lib.OpenstackEndpoint(client, endpointType, name, region, availability)
			return []interface{}{result1, result2}
		})
}
