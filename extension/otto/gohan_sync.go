// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package otto

import (
	"fmt"
	"github.com/cloudwan/gohan/sync"
	"github.com/xyproto/otto"
	"golang.org/x/net/context"
	"time"
)

func convertSyncEvent(event *sync.Event) map[string]interface{} {
	jsEvent := map[string]interface{}{}

	jsEvent["action"] = event.Action
	jsEvent["key"] = event.Key
	jsEvent["data"] = event.Data
	jsEvent["revision"] = event.Revision

	return jsEvent
}

func convertSyncNode(node *sync.Node) map[string]interface{} {
	jsNode := map[string]interface{}{}

	jsNode["key"] = node.Key
	jsNode["value"] = node.Value
	jsNode["revision"] = node.Revision
	jsNode["children"] = convertSyncNodes(node.Children)

	return jsNode
}

func convertSyncNodes(nodes []*sync.Node) []map[string]interface{} {
	jsNodes := []map[string]interface{}{}

	for _, node := range nodes {
		jsNodes = append(jsNodes, convertSyncNode(node))
	}

	return jsNodes
}

//init sets up vm to with environment
func init() {
	gohanSyncInit := func(env *Environment) {
		vm := env.VM

		builtins := map[string]interface{}{
			"gohan_sync_fetch": func(call otto.FunctionCall) otto.Value {
				var path string
				var err error
				var node *sync.Node
				var value otto.Value

				VerifyCallArguments(&call, "gohan_sync_fetch", 1)

				if path, err = GetString(call.Argument(0)); err != nil {
					ThrowOttoException(&call, "Invalid type of first argument: expected a string")
					return otto.NullValue()
				}

				errCh := make(chan error)
				defer close(errCh)
				go func() {
					node, err = env.Sync.Fetch(path)
					errCh <- err
				}()

				select {
				case interrupt := <-call.Otto.Interrupt:
					log.Debug("Received otto interrupt in gohan_sync_fetch")
					interrupt()
				case err = <-errCh:
					if err == context.DeadlineExceeded {
						ThrowOtto(&call, "TimeOutException", "Context Deadline Exceeded")
						return otto.NullValue()
					} else if err != nil {
						ThrowOttoException(&call, "Failed to fetch sync: "+err.Error())
						return otto.NullValue()
					}

					if value, err = vm.ToValue(convertSyncNode(node)); err == nil {
						return value
					}
				}

				return otto.NullValue()
			},
			"gohan_sync_delete": func(call otto.FunctionCall) otto.Value {
				var path string
				var prefix bool
				var err error

				if len(call.ArgumentList) == 1 {
					defaultPrefix, _ := otto.ToValue(false)
					call.ArgumentList = append(call.ArgumentList, defaultPrefix)
				}
				VerifyCallArguments(&call, "gohan_sync_delete", 2)

				if path, err = GetString(call.Argument(0)); err != nil {
					ThrowOttoException(&call, "Invalid type of first argument: expected a string")
					return otto.NullValue()
				}
				if prefix, err = GetBool(call.Argument(1)); err != nil {
					ThrowOttoException(&call, "Invalid type of second argument: expected a bool")
					return otto.NullValue()
				}

				errCh := make(chan error)
				defer close(errCh)
				go func() {
					err = env.Sync.Delete(path, prefix)
					errCh <- err
				}()

				select {
				case interrupt := <-call.Otto.Interrupt:
					log.Debug("Received otto interrupt in gohan_sync_delete")
					interrupt()
				case err = <-errCh:
					if err != nil {
						ThrowOttoException(&call, "Failed to delete sync: "+err.Error())
						return otto.NullValue()
					}
				}
				return otto.NullValue()
			},
			"gohan_sync_update": func(call otto.FunctionCall) otto.Value {
				var path string
				var value string
				var err error

				VerifyCallArguments(&call, "gohan_sync_update", 2)

				if path, err = GetString(call.Argument(0)); err != nil {
					ThrowOttoException(&call, "Invalid type of first argument: expected a string")
					return otto.NullValue()
				}
				if value, err = GetString(call.Argument(1)); err != nil {
					ThrowOttoException(&call, "Invalid type of second argument: expected a string")
					return otto.NullValue()
				}

				errCh := make(chan error)
				defer close(errCh)
				go func() {
					err = env.Sync.Update(path, value)
					errCh <- err
				}()

				select {
				case interrupt := <-call.Otto.Interrupt:
					log.Debug("Received otto interrupt in gohan_sync_update")
					interrupt()
				case err = <-errCh:
					if err != nil {
						ThrowOttoException(&call, "Failed to update sync: "+err.Error())
						return otto.NullValue()
					}
				}
				return otto.NullValue()
			},
			"gohan_sync_watch": func(call otto.FunctionCall) otto.Value {
				var path string
				var timeoutMsec int64
				var revision int64
				var err error
				var value otto.Value

				VerifyCallArguments(&call, "gohan_sync_watch", 3)

				if path, err = GetString(call.Argument(0)); err != nil {
					ThrowOttoException(&call, "Invalid type of first argument: expected a string")
					return otto.NullValue()
				}

				if timeoutMsec, err = GetInt64(call.Argument(1)); err != nil {
					ThrowOttoException(&call, "Invalid type of first argument: expected an int64")
					return otto.NullValue()
				}

				if revision, err = GetInt64(call.Argument(2)); err != nil {
					ThrowOttoException(&call, "Invalid type of first argument: expected an int64")
					return otto.NullValue()
				}

				eventChan := make(chan *sync.Event, 32) // non-blocking
				stopChan := make(chan bool, 1)          // non-blocking
				defer close(stopChan)
				errorChan := make(chan error, 1) // non-blocking

				go func() {
					if err := env.Sync.Watch(path, eventChan, stopChan, revision); err != nil {
						errorChan <- err
					}
				}()

				select {
				case interrupt := <-call.Otto.Interrupt:
					log.Debug("Received otto interrupt in gohan_sync_watch when watching on %s since revision %d", path, revision)
					interrupt()
				case event := <-eventChan:
					if value, err = vm.ToValue(convertSyncEvent(event)); err == nil {
						return value
					}
				case <-time.NewTimer(time.Duration(timeoutMsec) * time.Millisecond).C:
					log.Debug("Watch on %s since revision %d timed out after %d [ms]", path, revision, timeoutMsec)
					if value, err = vm.ToValue(map[string]interface{}{}); err == nil {
						return value
					}
				case err := <-errorChan:
					ThrowOttoException(&call, fmt.Sprintf("Watching on %s since revision %d failed: %s", path, revision, err.Error()))
				}
				return otto.NullValue()
			},
		}
		for name, object := range builtins {
			vm.Set(name, object)
		}
	}
	RegisterInit(gohanSyncInit)
}
