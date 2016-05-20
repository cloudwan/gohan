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
			stmtErr := stmt.HasArgs(
				"schema_id")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var schemaID string
				ischemaID := stmt.Arg("schema_id", context)
				if ischemaID != nil {
					schemaID = ischemaID.(string)
				}

				result1,
					err :=
					lib.GohanSchema(
						schemaID)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanSchema",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			schemaID := args[0].(string)

			result1,
				err :=
				lib.GohanSchema(
					schemaID)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("gohan_schemas",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs()
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				result1 :=
					lib.GohanSchemas()

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanSchemas",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			result1 :=
				lib.GohanSchemas()
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("gohan_policies",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs()
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				result1 :=
					lib.GohanPolicies()

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanPolicies",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			result1 :=
				lib.GohanPolicies()
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("read_config",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"path")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var path string
				ipath := stmt.Arg("path", context)
				if ipath != nil {
					path = ipath.(string)
				}

				err :=
					lib.ReadConfig(
						path)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("ReadConfig",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			path := args[0].(string)

			err :=
				lib.ReadConfig(
					path)
			return []interface{}{
				err}

		})

	gohanscript.RegisterStmtParser("get_config",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"key", "default_value")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var key string
				ikey := stmt.Arg("key", context)
				if ikey != nil {
					key = ikey.(string)
				}
				var defaultValue interface{}
				idefaultValue := stmt.Arg("default_value", context)
				if idefaultValue != nil {
					defaultValue = idefaultValue.(interface{})
				}

				result1 :=
					lib.GetConfig(
						key, defaultValue)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GetConfig",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			key := args[0].(string)
			defaultValue := args[0].(interface{})

			result1 :=
				lib.GetConfig(
					key, defaultValue)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("gohan_load_schema",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"src")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var src string
				isrc := stmt.Arg("src", context)
				if isrc != nil {
					src = isrc.(string)
				}

				result1,
					err :=
					lib.GohanLoadSchema(
						src)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("GohanLoadSchema",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			src := args[0].(string)

			result1,
				err :=
				lib.GohanLoadSchema(
					src)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("connect_db",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"db_type", "connection", "max_open_conn")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var dbType string
				idbType := stmt.Arg("db_type", context)
				if idbType != nil {
					dbType = idbType.(string)
				}
				var connection string
				iconnection := stmt.Arg("connection", context)
				if iconnection != nil {
					connection = iconnection.(string)
				}
				var maxOpenConn int
				imaxOpenConn := stmt.Arg("max_open_conn", context)
				if imaxOpenConn != nil {
					maxOpenConn = imaxOpenConn.(int)
				}

				result1,
					err :=
					lib.ConnectDB(
						dbType, connection, maxOpenConn)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("ConnectDB",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			dbType := args[0].(string)
			connection := args[0].(string)
			maxOpenConn := args[0].(int)

			result1,
				err :=
				lib.ConnectDB(
					dbType, connection, maxOpenConn)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("init_db",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"db_type", "connection", "drop_on_create", "cascade")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var dbType string
				idbType := stmt.Arg("db_type", context)
				if idbType != nil {
					dbType = idbType.(string)
				}
				var connection string
				iconnection := stmt.Arg("connection", context)
				if iconnection != nil {
					connection = iconnection.(string)
				}
				var dropOnCreate bool
				idropOnCreate := stmt.Arg("drop_on_create", context)
				if idropOnCreate != nil {
					dropOnCreate = idropOnCreate.(bool)
				}
				var cascade bool
				icascade := stmt.Arg("cascade", context)
				if icascade != nil {
					cascade = icascade.(bool)
				}

				err :=
					lib.InitDB(
						dbType, connection, dropOnCreate, cascade)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("InitDB",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			dbType := args[0].(string)
			connection := args[0].(string)
			dropOnCreate := args[0].(bool)
			cascade := args[0].(bool)

			err :=
				lib.InitDB(
					dbType, connection, dropOnCreate, cascade)
			return []interface{}{
				err}

		})

	gohanscript.RegisterStmtParser("db_begin",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"connection")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var connection db.DB
				iconnection := stmt.Arg("connection", context)
				if iconnection != nil {
					connection = iconnection.(db.DB)
				}

				result1,
					err :=
					lib.DBBegin(
						connection)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBBegin",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			connection := args[0].(db.DB)

			result1,
				err :=
				lib.DBBegin(
					connection)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("db_commit",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}

				err :=
					lib.DBCommit(
						tx)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBCommit",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)

			err :=
				lib.DBCommit(
					tx)
			return []interface{}{
				err}

		})

	gohanscript.RegisterStmtParser("db_close",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}

				err :=
					lib.DBClose(
						tx)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBClose",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)

			err :=
				lib.DBClose(
					tx)
			return []interface{}{
				err}

		})

	gohanscript.RegisterStmtParser("db_get",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx", "schema_id", "id", "tenant_id")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}
				var schemaID string
				ischemaID := stmt.Arg("schema_id", context)
				if ischemaID != nil {
					schemaID = ischemaID.(string)
				}
				var id string
				iid := stmt.Arg("id", context)
				if iid != nil {
					id = iid.(string)
				}
				var tenantID string
				itenantID := stmt.Arg("tenant_id", context)
				if itenantID != nil {
					tenantID = itenantID.(string)
				}

				result1,
					err :=
					lib.DBGet(
						tx, schemaID, id, tenantID)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBGet",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)
			schemaID := args[0].(string)
			id := args[0].(string)
			tenantID := args[0].(string)

			result1,
				err :=
				lib.DBGet(
					tx, schemaID, id, tenantID)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("db_create",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx", "schema_id", "data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}
				var schemaID string
				ischemaID := stmt.Arg("schema_id", context)
				if ischemaID != nil {
					schemaID = ischemaID.(string)
				}
				var data map[string]interface{}
				idata := stmt.Arg("data", context)
				if idata != nil {
					data = idata.(map[string]interface{})
				}

				err :=
					lib.DBCreate(
						tx, schemaID, data)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBCreate",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)
			schemaID := args[0].(string)
			data := args[0].(map[string]interface{})

			err :=
				lib.DBCreate(
					tx, schemaID, data)
			return []interface{}{
				err}

		})

	gohanscript.RegisterStmtParser("db_list",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx", "schema_id", "filter")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}
				var schemaID string
				ischemaID := stmt.Arg("schema_id", context)
				if ischemaID != nil {
					schemaID = ischemaID.(string)
				}
				var filter map[string]interface{}
				ifilter := stmt.Arg("filter", context)
				if ifilter != nil {
					filter = ifilter.(map[string]interface{})
				}

				result1,
					err :=
					lib.DBList(
						tx, schemaID, filter)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBList",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)
			schemaID := args[0].(string)
			filter := args[0].(map[string]interface{})

			result1,
				err :=
				lib.DBList(
					tx, schemaID, filter)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("db_update",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx", "schema_id", "data")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}
				var schemaID string
				ischemaID := stmt.Arg("schema_id", context)
				if ischemaID != nil {
					schemaID = ischemaID.(string)
				}
				var data map[string]interface{}
				idata := stmt.Arg("data", context)
				if idata != nil {
					data = idata.(map[string]interface{})
				}

				err :=
					lib.DBUpdate(
						tx, schemaID, data)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBUpdate",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)
			schemaID := args[0].(string)
			data := args[0].(map[string]interface{})

			err :=
				lib.DBUpdate(
					tx, schemaID, data)
			return []interface{}{
				err}

		})

	gohanscript.RegisterStmtParser("db_delete",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx", "schema_id", "id")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}
				var schemaID string
				ischemaID := stmt.Arg("schema_id", context)
				if ischemaID != nil {
					schemaID = ischemaID.(string)
				}
				var id string
				iid := stmt.Arg("id", context)
				if iid != nil {
					id = iid.(string)
				}

				err :=
					lib.DBDelete(
						tx, schemaID, id)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBDelete",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)
			schemaID := args[0].(string)
			id := args[0].(string)

			err :=
				lib.DBDelete(
					tx, schemaID, id)
			return []interface{}{
				err}

		})

	gohanscript.RegisterStmtParser("db_query",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx", "schema_id", "sql", "arguments")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}
				var schemaID string
				ischemaID := stmt.Arg("schema_id", context)
				if ischemaID != nil {
					schemaID = ischemaID.(string)
				}
				var sql string
				isql := stmt.Arg("sql", context)
				if isql != nil {
					sql = isql.(string)
				}
				var arguments []interface{}
				iarguments := stmt.Arg("arguments", context)
				if iarguments != nil {
					arguments = iarguments.([]interface{})
				}

				result1,
					err :=
					lib.DBQuery(
						tx, schemaID, sql, arguments)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBQuery",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)
			schemaID := args[0].(string)
			sql := args[0].(string)
			arguments := args[0].([]interface{})

			result1,
				err :=
				lib.DBQuery(
					tx, schemaID, sql, arguments)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("db_exec",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"tx", "sql", "arguments")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var tx transaction.Transaction
				itx := stmt.Arg("tx", context)
				if itx != nil {
					tx = itx.(transaction.Transaction)
				}
				var sql string
				isql := stmt.Arg("sql", context)
				if isql != nil {
					sql = isql.(string)
				}
				var arguments []interface{}
				iarguments := stmt.Arg("arguments", context)
				if iarguments != nil {
					arguments = iarguments.([]interface{})
				}

				err :=
					lib.DBExec(
						tx, sql, arguments)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBExec",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			tx := args[0].(transaction.Transaction)
			sql := args[0].(string)
			arguments := args[0].([]interface{})

			err :=
				lib.DBExec(
					tx, sql, arguments)
			return []interface{}{
				err}

		})

	gohanscript.RegisterStmtParser("db_column",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"schema_id", "join")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var schemaID string
				ischemaID := stmt.Arg("schema_id", context)
				if ischemaID != nil {
					schemaID = ischemaID.(string)
				}
				var join bool
				ijoin := stmt.Arg("join", context)
				if ijoin != nil {
					join = ijoin.(bool)
				}

				result1,
					err :=
					lib.DBColumn(
						schemaID, join)

				return result1, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("DBColumn",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			schemaID := args[0].(string)
			join := args[0].(bool)

			result1,
				err :=
				lib.DBColumn(
					schemaID, join)
			return []interface{}{
				result1,
				err}

		})

	gohanscript.RegisterStmtParser("error",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"code", "name", "message")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				var code int
				icode := stmt.Arg("code", context)
				if icode != nil {
					code = icode.(int)
				}
				var name string
				iname := stmt.Arg("name", context)
				if iname != nil {
					name = iname.(string)
				}
				var message string
				imessage := stmt.Arg("message", context)
				if imessage != nil {
					message = imessage.(string)
				}

				err :=
					lib.Error(
						code, name, message)

				return nil, err

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Error",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			code := args[0].(int)
			name := args[0].(string)
			message := args[1].(string)

			err :=
				lib.Error(
					code, name, message)
			return []interface{}{
				err}

		})

}
