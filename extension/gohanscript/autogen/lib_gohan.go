package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
)

func init() {
	gohanscript.RegisterStmtParser("gohan_schema",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
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
}
