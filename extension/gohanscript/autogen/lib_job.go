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
			return func(context *gohanscript.Context) (interface{}, error) {
				workers, _ := stmt.Arg("workers", context).(int)
				result1 := lib.MakeQueue(workers)
				return result1, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("MakeQueue",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			workers, _ := args[i].(int)
			i++
			result1 := lib.MakeQueue(workers)
			return []interface{}{result1}
		})
	gohanscript.RegisterStmtParser("wait_queue",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				queue, _ := stmt.Arg("queue", context).(*job.Queue)
				lib.WaitQueue(queue)
				return nil, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("WaitQueue",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			queue, _ := args[i].(*job.Queue)
			i++
			lib.WaitQueue(queue)
			return []interface{}{}
		})
	gohanscript.RegisterStmtParser("stop",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				queue, _ := stmt.Arg("queue", context).(*job.Queue)
				lib.Stop(queue)
				return nil, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("Stop",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			queue, _ := args[i].(*job.Queue)
			i++
			lib.Stop(queue)
			return []interface{}{}
		})
	gohanscript.RegisterStmtParser("force_stop",
		func(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
			return func(context *gohanscript.Context) (interface{}, error) {
				queue, _ := stmt.Arg("queue", context).(*job.Queue)
				lib.ForceStop(queue)
				return nil, nil
			}, nil
		})
	gohanscript.RegisterMiniGoFunc("ForceStop",
		func(vm *gohanscript.VM, args []interface{}) []interface{} {
			i := 0
			queue, _ := args[i].(*job.Queue)
			i++
			lib.ForceStop(queue)
			return []interface{}{}
		})
}
