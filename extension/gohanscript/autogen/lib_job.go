package autogen

// AUTO GENERATED CODE DO NOT MODIFY MANUALLY
import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/extension/gohanscript/lib"
	"github.com/cloudwan/gohan/job"
)

func init() {

	gohanscript.RegisterStmtParser("make_queue",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"workers")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				workers := stmt.Arg("workers", context).(int)

				result1 :=
					lib.MakeQueue(
						workers)

				return result1, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MakeQueue",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			workers := args[0].(int)

			result1 :=
				lib.MakeQueue(
					workers)
			return []interface{}{
				result1}

		})

	gohanscript.RegisterStmtParser("wait_queue",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"queue")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				queue := stmt.Arg("queue", context).(*job.Queue)

				lib.WaitQueue(
					queue)
				return nil, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("WaitQueue",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			queue := args[0].(*job.Queue)

			lib.WaitQueue(
				queue)
			return nil

		})

	gohanscript.RegisterStmtParser("stop",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"queue")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				queue := stmt.Arg("queue", context).(*job.Queue)

				lib.Stop(
					queue)
				return nil, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Stop",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			queue := args[0].(*job.Queue)

			lib.Stop(
				queue)
			return nil

		})

	gohanscript.RegisterStmtParser("force_stop",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			stmtErr := stmt.HasArgs(
				"queue")
			if stmtErr != nil {
				return nil, stmtErr
			}
			return func(context *gohanscript.Context) (interface{}, error) {

				queue := stmt.Arg("queue", context).(*job.Queue)

				lib.ForceStop(
					queue)
				return nil, nil

			}, nil
		})
	gohanscript.RegisterMiniGoFunc("ForceStop",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {

			queue := args[0].(*job.Queue)

			lib.ForceStop(
				queue)
			return nil

		})

}
