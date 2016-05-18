// Copyright (C) 2016  Juniper Networks, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"github.com/cloudwan/gohan/extension/gohanscript"
	"github.com/cloudwan/gohan/job"
)

func init() {
	gohanscript.RegisterStmtParser("job", backgroundJob)
}

func backgroundJob(stmt *gohanscript.Stmt) (func(*gohanscript.Context) (interface{}, error), error) {
	stmts, err := gohanscript.MakeStmts(stmt.File, stmt.RawNode["job"])
	if err != nil {
		return nil, stmt.Errorf("background code error: %s", err)
	}
	queueArg, err := gohanscript.NewValue(stmt.RawData["queue"])
	if err != nil {
		return nil, stmt.Errorf("background code error: %s", err)
	}
	f, err := gohanscript.StmtsToFunc("job", stmts)
	if err != nil {
		return nil, err
	}
	return func(context *gohanscript.Context) (interface{}, error) {
		queue := queueArg.Value(context).(*job.Queue)
		newContext := context.Extend(map[string]interface{}{})
		queue.Add(
			job.NewJob(func() {
				f(newContext)
				newContext.VM.Stop()
			}))
		return nil, nil
	}, nil
}

//MakeQueue makes new worker queue with specfied workers
func MakeQueue(workers int) *job.Queue {
	return job.NewQueue(uint(workers))
}

//WaitQueue waits queue get empty
func WaitQueue(queue *job.Queue) {
	queue.Wait()
}

//Stop stops work queue
func Stop(queue *job.Queue) {
	queue.Stop()
}

//ForceStop force stop queue
func ForceStop(queue *job.Queue) {
	queue.ForceStop()
}
