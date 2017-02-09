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

	"github.com/k0kubun/pp"
	"github.com/nati/yaml"

	"github.com/cloudwan/gohan/job"
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/util"
)

var log = l.NewLoggerForModule("gohanscript")

func init() {
	l.SetUpBasicLogging(os.Stderr, l.DefaultFormat)
}

//LoadYAMLFile loads YAML from file
func LoadYAMLFile(file string) (*yaml.Node, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return LoadYAML(buf)
}

//LoadYAML loads YAML from byte buffer
func LoadYAML(buf []byte) (*yaml.Node, error) {
	node, err := yaml.ParseYAML(buf)
	if err != nil {
		return nil, err
	}
	return node.Children[0], nil
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

func forEachList(vm *VM, obj []interface{}, workerNum int, f func(item interface{}) error) error {
	if obj == nil {
		return nil
	}
	size := len(obj)
	if size == 0 {
		return nil
	}
	if workerNum == 0 {
		for _, value := range obj {
			select {
			case f := <-vm.StopChan:
				f()
				return nil
			default:
				err := f(value)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	queue := job.NewQueue(uint(workerNum))
	for _, item := range obj {
		localItem := item
		queue.Add(job.NewJob(func() {
			f(localItem)
		}))
	}
	queue.Wait()
	return nil
}

//RunTests find tests under a directory and it executes them.
func RunTests(dir string) (total, success, failed int) {
	vm := NewVM()
	abs, _ := filepath.Abs(dir)
	filepath.Walk(abs,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println(err)
				return err
			}
			if info == nil {
				return nil
			}
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
