package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
	"github.com/rackspace/gophercloud"
)

func init() {

	gohanscript.RegisterStmtParser("get_openstack_client",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"auth_url", "user_name", "password", "domain_name", "tenant_name", "version")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				authURL := stmt.Arg("auth_url", context).(string)
				userName := stmt.Arg("user_name", context).(string)
				password := stmt.Arg("password", context).(string)
				domainName := stmt.Arg("domain_name", context).(string)
				tenantName := stmt.Arg("tenant_name", context).(string)
				version := stmt.Arg("version", context).(string)

				result1,
					err :=
					lib.GetOpenstackClient(
						authURL, userName, password, domainName, tenantName, version)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GetOpenstackClient",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			authURL := args[0].(string)
			userName := args[1].(string)
			password := args[2].(string)
			domainName := args[3].(string)
			tenantName := args[4].(string)
			version := args[5].(string)

			result1,
				err :=
				lib.GetOpenstackClient(
					authURL, userName, password, domainName, tenantName, version)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("openstack_token",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"client")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				client := stmt.Arg("client", context).(*gophercloud.ServiceClient)

				result1 :=
					lib.OpenstackToken(
						client)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackToken",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			client := args[0].(*gophercloud.ServiceClient)

			result1 :=
				lib.OpenstackToken(
					client)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("openstack_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"client", "url")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				client := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				url := stmt.Arg("url", context).(string)

				result1,
					err :=
					lib.OpenstackGet(
						client, url)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackGet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			client := args[0].(*gophercloud.ServiceClient)
			url := args[0].(string)

			result1,
				err :=
				lib.OpenstackGet(
					client, url)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("openstack_put",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"client", "url", "data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				client := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				url := stmt.Arg("url", context).(string)
				data := stmt.Arg("data", context).(interface{})

				result1,
					err :=
					lib.OpenstackPut(
						client, url, data)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackPut",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			client := args[0].(*gophercloud.ServiceClient)
			url := args[0].(string)
			data := args[0].(interface{})

			result1,
				err :=
				lib.OpenstackPut(
					client, url, data)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("openstack_post",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"client", "url", "data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				client := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				url := stmt.Arg("url", context).(string)
				data := stmt.Arg("data", context).(interface{})

				result1,
					err :=
					lib.OpenstackPost(
						client, url, data)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackPost",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			client := args[0].(*gophercloud.ServiceClient)
			url := args[0].(string)
			data := args[0].(interface{})

			result1,
				err :=
				lib.OpenstackPost(
					client, url, data)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("openstack_delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"client", "url")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				client := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				url := stmt.Arg("url", context).(string)

				result1,
					err :=
					lib.OpenstackDelete(
						client, url)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackDelete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			client := args[0].(*gophercloud.ServiceClient)
			url := args[0].(string)

			result1,
				err :=
				lib.OpenstackDelete(
					client, url)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("openstack_endpoint",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"client", "endpoint_type", "name", "region", "availability")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				client := stmt.Arg("client", context).(*gophercloud.ServiceClient)
				endpointType := stmt.Arg("endpoint_type", context).(string)
				name := stmt.Arg("name", context).(string)
				region := stmt.Arg("region", context).(string)
				availability := stmt.Arg("availability", context).(string)

				result1,
					err :=
					lib.OpenstackEndpoint(
						client, endpointType, name, region, availability)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("OpenstackEndpoint",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			client := args[0].(*gophercloud.ServiceClient)
			endpointType := args[0].(string)
			name := args[1].(string)
			region := args[2].(string)
			availability := args[3].(string)

			result1,
				err :=
				lib.OpenstackEndpoint(
					client, endpointType, name, region, availability)
			return []interface{}{
				result1,
				err}

		})

}
