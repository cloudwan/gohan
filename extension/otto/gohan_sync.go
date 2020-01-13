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
	"context"

	"github.com/robertkrimen/otto"

	"github.com/cloudwan/gohan/sync"
)

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

				errCh := make(chan error, 1)
				go func() {
					err = env.Sync.Update(context.Background(), path, value)
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
		}
		for name, object := range builtins {
			vm.Set(name, object)
		}
	}
	RegisterInit(gohanSyncInit)
}
