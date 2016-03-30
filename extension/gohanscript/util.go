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

package gohanscript

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nati/yaml"
	"github.com/op/go-logging"

	"github.com/cloudwan/gohan/util"
	"github.com/k0kubun/pp"
)

var log = logging.MustGetLogger("gohanscript")

func init() {
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	var format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} ▶ %{level:.7s} %{color:reset} %{message}`,
	)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter)
}

//LoadYAMLFile loads YAML from file
func LoadYAMLFile(file string) (*yaml.Node, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return LoadYAML(buf), nil
}

//LoadYAML loads YAML from byte buffer
func LoadYAML(buf []byte) *yaml.Node {
	return yaml.ParseYAML(buf).Children[0]
}

func convertMapformat(d interface{}) interface{} {
	switch data := d.(type) {
	case map[interface{}]interface{}:
		result := map[string]interface{}{}
		for key, value := range data {
			result[key.(string)] = convertMapformat(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(data))
		for i, item := range data {
			result[i] = convertMapformat(item)
		}
		return result
	default:
		return data
	}
}

func pprint(obj interface{}) {
	pp.Print(obj)
}

func forEachList(vm *VM, obj []interface{}, workerNum int, f func(item interface{})) {
	if workerNum == 0 {
		for _, value := range obj {
			select {
			case f := <-vm.StopChan:
				f()
				return
			default:
				f(value)
			}
		}
		return
	}
	if obj == nil {
		return
	}
	size := len(obj)
	if size == 0 {
		return
	}
	var wg sync.WaitGroup
	if workerNum > size {
		workerNum = size
	}
	numJob := size / workerNum
	for start := 0; start < size; start += numJob {
		wg.Add(1)
		go func(start, end int) {
			defer func() {
				if err := recover(); err != nil {
					log.Error(fmt.Sprintf("panic: %s", err))
				}
			}()
			for i := start; i < end; i++ {
				f(obj[i])
			}
			wg.Done()
		}(start, start+numJob)
	}
	wg.Wait()
	return
}

//RunTests find tests under a directory and it executes them.
func RunTests(dir string) (total, success, failed int) {
	vm := NewVM()
	filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, "_test.yaml") {
				return nil
			}
			result, err := vm.RunFile(path)
			if err != nil {
				log.Warning("test failed: %s", err)
			}
			if result == nil {
				log.Warning("No result")
			}
			resultMap := result.(map[string]interface{})
			total += resultMap["count"].(int)
			success += resultMap["success"].(int)
			failed += resultMap["failed"].(int)
			return nil
		})
	fmt.Printf("total: %d success: %d  failed: %d \n", total, success, failed)
	return
}

func extractID(value interface{}) string {
	id := util.MaybeString(value)
	id = strings.Replace(id, " ", "", -1)
	id = strings.Replace(id, "{{", "", -1)
	id = strings.Replace(id, "}}", "", -1)
	return id
}
