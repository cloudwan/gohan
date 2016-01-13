package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
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
	gohanscript.RegisterStmtParser("split",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				value, _ := stmt.Arg("value", context).(string)
				sep, _ := stmt.Arg("sep", context).(string)
				var err error
				result1, err := lib.Split(value, sep)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Split",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			value, _ := args[i].(string)
			i++
			sep, _ := args[i].(string)
			i++
			result1, result2 := lib.Split(value, sep)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("join",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				value, _ := stmt.Arg("value", context).([]interface{})
				sep, _ := stmt.Arg("sep", context).(string)
				var err error
				result1, err := lib.Join(value, sep)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Join",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			value, _ := args[i].([]interface{})
			i++
			sep, _ := args[i].(string)
			i++
			result1, result2 := lib.Join(value, sep)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("uuid",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				result1 := lib.UUID()
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("UUID",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			result1 := lib.UUID()
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("env",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				result1 := lib.Env()
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Env",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			result1 := lib.Env()
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("normalize_map",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				data, _ := stmt.Arg("data", context).(map[string]interface{})
				result1 := lib.NormalizeMap(data)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("NormalizeMap",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			data, _ := args[i].(map[string]interface{})
			i++
			result1 := lib.NormalizeMap(data)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("gohan_schema",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				schemaID, _ := stmt.Arg("schema_id", context).(string)
				var err error
				result1, err := lib.GohanSchema(schemaID)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanSchema",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			schemaID, _ := args[i].(string)
			i++
			result1, result2 := lib.GohanSchema(schemaID)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("gohan_schemas",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				result1 := lib.GohanSchemas()
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanSchemas",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			result1 := lib.GohanSchemas()
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("gohan_policies",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				result1 := lib.GohanPolicies()
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanPolicies",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			result1 := lib.GohanPolicies()
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("read_config",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				path, _ := stmt.Arg("path", context).(string)
				err := lib.ReadConfig(path)
				return nil, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("ReadConfig",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			path, _ := args[i].(string)
			i++
			result1 := lib.ReadConfig(path)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("get_config",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				key, _ := stmt.Arg("key", context).(string)
				defaultValue, _ := stmt.Arg("default_value", context).(interface{})
				result1 := lib.GetConfig(key, defaultValue)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GetConfig",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			key, _ := args[i].(string)
			i++
			defaultValue, _ := args[i].(interface{})
			i++
			result1 := lib.GetConfig(key, defaultValue)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("gohan_load_schema",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				src, _ := stmt.Arg("src", context).(string)
				var err error
				result1, err := lib.GohanLoadSchema(src)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanLoadSchema",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			src, _ := args[i].(string)
			i++
			result1, result2 := lib.GohanLoadSchema(src)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("connect_db",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				dbType, _ := stmt.Arg("db_type", context).(string)
				connection, _ := stmt.Arg("connection", context).(string)
				var err error
				result1, err := lib.ConnectDB(dbType, connection)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("ConnectDB",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			dbType, _ := args[i].(string)
			i++
			connection, _ := args[i].(string)
			i++
			result1, result2 := lib.ConnectDB(dbType, connection)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("init_db",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				dbType, _ := stmt.Arg("db_type", context).(string)
				connection, _ := stmt.Arg("connection", context).(string)
				dropOnCreate, _ := stmt.Arg("drop_on_create", context).(bool)
				cascade, _ := stmt.Arg("cascade", context).(bool)
				err := lib.InitDB(dbType, connection, dropOnCreate, cascade)
				return nil, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("InitDB",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			dbType, _ := args[i].(string)
			i++
			connection, _ := args[i].(string)
			i++
			dropOnCreate, _ := args[i].(bool)
			i++
			cascade, _ := args[i].(bool)
			i++
			result1 := lib.InitDB(dbType, connection, dropOnCreate, cascade)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("db_begin",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				connection, _ := stmt.Arg("connection", context).(db.DB)
				var err error
				result1, err := lib.DBBegin(connection)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBBegin",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			connection, _ := args[i].(db.DB)
			i++
			result1, result2 := lib.DBBegin(connection)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("db_commit",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				tx, _ := stmt.Arg("tx", context).(transaction.Transaction)
				err := lib.DBCommit(tx)
				return nil, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBCommit",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			tx, _ := args[i].(transaction.Transaction)
			i++
			result1 := lib.DBCommit(tx)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("db_close",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				tx, _ := stmt.Arg("tx", context).(transaction.Transaction)
				err := lib.DBClose(tx)
				return nil, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBClose",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			tx, _ := args[i].(transaction.Transaction)
			i++
			result1 := lib.DBClose(tx)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("db_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				tx, _ := stmt.Arg("tx", context).(transaction.Transaction)
				schemaID, _ := stmt.Arg("schema_id", context).(string)
				id, _ := stmt.Arg("id", context).(string)
				tenantID, _ := stmt.Arg("tenant_id", context).(string)
				var err error
				result1, err := lib.DBGet(tx, schemaID, id, tenantID)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBGet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			tx, _ := args[i].(transaction.Transaction)
			i++
			schemaID, _ := args[i].(string)
			i++
			id, _ := args[i].(string)
			i++
			tenantID, _ := args[i].(string)
			i++
			result1, result2 := lib.DBGet(tx, schemaID, id, tenantID)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("db_create",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				tx, _ := stmt.Arg("tx", context).(transaction.Transaction)
				schemaID, _ := stmt.Arg("schema_id", context).(string)
				data, _ := stmt.Arg("data", context).(map[string]interface{})
				err := lib.DBCreate(tx, schemaID, data)
				return nil, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBCreate",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			tx, _ := args[i].(transaction.Transaction)
			i++
			schemaID, _ := args[i].(string)
			i++
			data, _ := args[i].(map[string]interface{})
			i++
			result1 := lib.DBCreate(tx, schemaID, data)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("db_list",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				tx, _ := stmt.Arg("tx", context).(transaction.Transaction)
				schemaID, _ := stmt.Arg("schema_id", context).(string)
				filter, _ := stmt.Arg("filter", context).(map[string]interface{})
				var err error
				result1, err := lib.DBList(tx, schemaID, filter)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBList",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			tx, _ := args[i].(transaction.Transaction)
			i++
			schemaID, _ := args[i].(string)
			i++
			filter, _ := args[i].(map[string]interface{})
			i++
			result1, result2 := lib.DBList(tx, schemaID, filter)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("db_update",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				tx, _ := stmt.Arg("tx", context).(transaction.Transaction)
				schemaID, _ := stmt.Arg("schema_id", context).(string)
				data, _ := stmt.Arg("data", context).(map[string]interface{})
				err := lib.DBUpdate(tx, schemaID, data)
				return nil, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBUpdate",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			tx, _ := args[i].(transaction.Transaction)
			i++
			schemaID, _ := args[i].(string)
			i++
			data, _ := args[i].(map[string]interface{})
			i++
			result1 := lib.DBUpdate(tx, schemaID, data)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("db_delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				tx, _ := stmt.Arg("tx", context).(transaction.Transaction)
				schemaID, _ := stmt.Arg("schema_id", context).(string)
				id, _ := stmt.Arg("id", context).(string)
				err := lib.DBDelete(tx, schemaID, id)
				return nil, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBDelete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			tx, _ := args[i].(transaction.Transaction)
			i++
			schemaID, _ := args[i].(string)
			i++
			id, _ := args[i].(string)
			i++
			result1 := lib.DBDelete(tx, schemaID, id)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("db_query",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				tx, _ := stmt.Arg("tx", context).(transaction.Transaction)
				schemaID, _ := stmt.Arg("schema_id", context).(string)
				sql, _ := stmt.Arg("sql", context).(string)
				arguments, _ := stmt.Arg("arguments", context).([]interface{})
				var err error
				result1, err := lib.DBQuery(tx, schemaID, sql, arguments)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBQuery",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			tx, _ := args[i].(transaction.Transaction)
			i++
			schemaID, _ := args[i].(string)
			i++
			sql, _ := args[i].(string)
			i++
			arguments, _ := args[i].([]interface{})
			i++
			result1, result2 := lib.DBQuery(tx, schemaID, sql, arguments)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("db_column",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				schemaID, _ := stmt.Arg("schema_id", context).(string)
				join, _ := stmt.Arg("join", context).(bool)
				var err error
				result1, err := lib.DBColumn(schemaID, join)
				return result1, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBColumn",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			schemaID, _ := args[i].(string)
			i++
			join, _ := args[i].(bool)
			i++
			result1, result2 := lib.DBColumn(schemaID, join)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("http_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
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
	gohanscript.RegisterStmtParser("append",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				value, _ := stmt.Arg("value", context).(interface{})
				result1 := lib.Append(list, value)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Append",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			value, _ := args[i].(interface{})
			i++
			result1 := lib.Append(list, value)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("contains",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				value, _ := stmt.Arg("value", context).(interface{})
				result1 := lib.Contains(list, value)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Contains",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			value, _ := args[i].(interface{})
			i++
			result1 := lib.Contains(list, value)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("size",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				result1 := lib.Size(list)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Size",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1 := lib.Size(list)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("shift",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				var err error
				result1, result2 := lib.Shift(list)
				return []interface{}{result1, result2}, err
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Shift",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1, result2 := lib.Shift(list)
			return []interface{}{result1, result2}
		})
	gohanscript.RegisterStmtParser("unshift",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				value, _ := stmt.Arg("value", context).(interface{})
				result1 := lib.Unshift(list, value)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Unshift",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			value, _ := args[i].(interface{})
			i++
			result1 := lib.Unshift(list, value)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("copy",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				result1 := lib.Copy(list)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Copy",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1 := lib.Copy(list)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				index, _ := stmt.Arg("index", context).(int)
				result1 := lib.Delete(list, index)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Delete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			index, _ := args[i].(int)
			i++
			result1 := lib.Delete(list, index)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("first",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				result1 := lib.First(list)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("First",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1 := lib.First(list)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("last",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				list, _ := stmt.Arg("list", context).([]interface{})
				result1 := lib.Last(list)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Last",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			list, _ := args[i].([]interface{})
			i++
			result1 := lib.Last(list)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("add_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				a, _ := stmt.Arg("a", context).(int)
				b, _ := stmt.Arg("b", context).(int)
				result1 := lib.AddInt(a, b)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("AddInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			a, _ := args[i].(int)
			i++
			b, _ := args[i].(int)
			i++
			result1 := lib.AddInt(a, b)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("sub_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				a, _ := stmt.Arg("a", context).(int)
				b, _ := stmt.Arg("b", context).(int)
				result1 := lib.SubInt(a, b)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("SubInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			a, _ := args[i].(int)
			i++
			b, _ := args[i].(int)
			i++
			result1 := lib.SubInt(a, b)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("mul_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				a, _ := stmt.Arg("a", context).(int)
				b, _ := stmt.Arg("b", context).(int)
				result1 := lib.MulInt(a, b)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MulInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			a, _ := args[i].(int)
			i++
			b, _ := args[i].(int)
			i++
			result1 := lib.MulInt(a, b)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("div_int",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.VM, *gohanscript.Context) (interface{}, error), error) {
			return func(vm *gohanscript.VM, context *gohanscript.Context) (interface{}, error) {
				a, _ := stmt.Arg("a", context).(int)
				b, _ := stmt.Arg("b", context).(int)
				result1 := lib.DivInt(a, b)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DivInt",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			a, _ := args[i].(int)
			i++
			b, _ := args[i].(int)
			i++
			result1 := lib.DivInt(a, b)
			return []interface{}{result1}
		})
}
